package dump

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/search"
	"github.com/bwireman/archivist/internal/store"
)

const DefaultTopK = 20

type Options struct {
	Query  string
	Scope  string
	Type   store.ChunkType
	TopK   int
	Output string
}

type Item struct {
	Chunk store.Chunk
	Score float64
}

type Result struct {
	Query string
	Scope string
	Type  store.ChunkType
	Items []Item
}

type WriteResult struct {
	Stdout bool
	Files  []string
}

func Collect(ctx context.Context, st *store.Store, embedder embed.Embedder, opts Options) (*Result, error) {
	query := strings.TrimSpace(opts.Query)
	out := &Result{
		Query: query,
		Scope: strings.TrimSpace(opts.Scope),
		Type:  opts.Type,
	}

	if query != "" {
		if embedder == nil {
			return nil, fmt.Errorf("embedder is required for query dumps")
		}
		topK := opts.TopK
		if topK <= 0 {
			topK = DefaultTopK
		}
		results, err := search.Search(ctx, st, embedder, query, search.Options{
			TopK:  topK,
			Type:  opts.Type,
			Scope: out.Scope,
		})
		if err != nil {
			return nil, err
		}
		for _, r := range results {
			out.Items = append(out.Items, Item{Chunk: r.Chunk, Score: r.Score})
		}
		return out, nil
	}

	var (
		chunks []store.Chunk
		err    error
	)
	if opts.Type != "" {
		chunks, err = st.ChunksByType(opts.Type)
	} else {
		chunks, err = st.AllChunks()
	}
	if err != nil {
		return nil, err
	}

	for _, c := range chunks {
		if !search.MatchScope(out.Scope, c.Path) {
			continue
		}
		out.Items = append(out.Items, Item{Chunk: c})
	}
	sort.Slice(out.Items, func(i, j int) bool {
		a, b := out.Items[i].Chunk, out.Items[j].Chunk
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.StartLine != b.StartLine {
			return a.StartLine < b.StartLine
		}
		return a.ID < b.ID
	})
	if opts.TopK > 0 && len(out.Items) > opts.TopK {
		out.Items = out.Items[:opts.TopK]
	}
	return out, nil
}

func Format(r *Result) string {
	var b strings.Builder
	b.WriteString("# Archivist context dump\n\n")
	b.WriteString("Retrieved chunks from a local code index. Use only facts present here.\n\n")
	if r.Query != "" {
		fmt.Fprintf(&b, "- Query: %s\n", r.Query)
	} else {
		b.WriteString("- Query: (all matching chunks)\n")
	}
	if r.Scope != "" {
		fmt.Fprintf(&b, "- Scope: %s\n", r.Scope)
	}
	if r.Type != "" {
		fmt.Fprintf(&b, "- Type: %s\n", r.Type)
	}
	fmt.Fprintf(&b, "- Chunks: %d\n\n", len(r.Items))
	if len(r.Items) == 0 {
		b.WriteString("No matching chunks.\n")
		return b.String()
	}
	b.WriteString(formatItems(r.Items))
	return b.String()
}

func formatItems(items []Item) string {
	var b strings.Builder
	for i, item := range items {
		if i > 0 {
			b.WriteString("\n")
		}
		c := item.Chunk
		loc := location(c)
		if item.Score > 0 {
			fmt.Fprintf(&b, "## `%s` (%s, %s, score %.3f)\n\n", c.Path, c.ChunkType, loc, item.Score)
		} else {
			fmt.Fprintf(&b, "## `%s` (%s, %s)\n\n", c.Path, c.ChunkType, loc)
		}
		if c.Metadata != nil {
			if author := c.Metadata["blame_author"]; author != "" {
				commit := c.Metadata["blame_commit"]
				if commit != "" {
					fmt.Fprintf(&b, "Blame: %s (%s)\n\n", author, commit)
				} else {
					fmt.Fprintf(&b, "Blame: %s\n\n", author)
				}
			}
		}
		lang := fenceLang(c.Path, c.ChunkType)
		fence := codeFence(c.Content)
		fmt.Fprintf(&b, "%s%s\n%s\n%s\n", fence, lang, strings.TrimRight(c.Content, "\n"), fence)
		b.WriteByte('\n')
	}
	return b.String()
}

func Write(r *Result, output string, stdout io.Writer) (*WriteResult, error) {
	if stdout == nil {
		stdout = os.Stdout
	}
	if !splitOutput(output) {
		body := Format(r)
		if output == "" || output == "-" {
			if _, err := io.WriteString(stdout, body); err != nil {
				return nil, err
			}
			return &WriteResult{Stdout: true}, nil
		}
		if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(output, []byte(body), 0o644); err != nil {
			return nil, err
		}
		return &WriteResult{Files: []string{output}}, nil
	}

	dir := strings.TrimRight(output, `/\`)
	if dir == "" {
		dir = output
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	groups := groupByPath(r.Items)
	var files []string
	var index strings.Builder
	index.WriteString("# Archivist dump index\n\n")
	if r.Query != "" {
		fmt.Fprintf(&index, "- Query: %s\n", r.Query)
	}
	if r.Scope != "" {
		fmt.Fprintf(&index, "- Scope: %s\n", r.Scope)
	}
	fmt.Fprintf(&index, "- Files: %d\n- Chunks: %d\n\n", len(groups), len(r.Items))

	paths := make([]string, 0, len(groups))
	for path := range groups {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	for _, path := range paths {
		items := groups[path]
		rel := dumpFileName(path)
		abs := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return nil, err
		}
		var body strings.Builder
		fmt.Fprintf(&body, "# `%s`\n\n", path)
		body.WriteString(formatItems(items))
		if err := os.WriteFile(abs, []byte(body.String()), 0o644); err != nil {
			return nil, err
		}
		files = append(files, abs)
		fmt.Fprintf(&index, "- `%s` (%d chunks)\n", filepath.ToSlash(rel), len(items))
	}

	indexPath := filepath.Join(dir, "index.md")
	if err := os.WriteFile(indexPath, []byte(index.String()), 0o644); err != nil {
		return nil, err
	}
	return &WriteResult{Files: append([]string{indexPath}, files...)}, nil
}

func splitOutput(output string) bool {
	if output == "" || output == "-" {
		return false
	}
	if strings.HasSuffix(output, "/") || strings.HasSuffix(output, string(filepath.Separator)) {
		return true
	}
	info, err := os.Stat(output)
	return err == nil && info.IsDir()
}

func groupByPath(items []Item) map[string][]Item {
	groups := make(map[string][]Item)
	for _, item := range items {
		path := item.Chunk.Path
		if path == "" {
			path = "unknown"
		}
		groups[path] = append(groups[path], item)
	}
	for path := range groups {
		sort.Slice(groups[path], func(i, j int) bool {
			a, b := groups[path][i].Chunk, groups[path][j].Chunk
			if a.StartLine != b.StartLine {
				return a.StartLine < b.StartLine
			}
			return a.ID < b.ID
		})
	}
	return groups
}

func dumpFileName(path string) string {
	clean := filepath.ToSlash(path)
	clean = strings.TrimPrefix(clean, "/")
	if clean == "" {
		clean = "unknown"
	}
	parts := strings.Split(clean, "/")
	safe := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" || p == "." {
			continue
		}
		if p == ".." {
			safe = append(safe, "_")
			continue
		}
		safe = append(safe, p)
	}
	if len(safe) == 0 {
		safe = []string{"unknown"}
	}
	return filepath.Join(safe...) + ".md"
}

func location(c store.Chunk) string {
	if c.StartLine > 0 || c.EndLine > 0 {
		if c.StartLine == c.EndLine {
			return fmt.Sprintf("line %d", c.StartLine)
		}
		return fmt.Sprintf("lines %d-%d", c.StartLine, c.EndLine)
	}
	return "no line range"
}

func fenceLang(path string, chunkType store.ChunkType) string {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	switch ext {
	case "go", "py", "js", "ts", "tsx", "jsx", "rs", "java", "json", "yaml", "yml", "md", "sh", "bash", "toml":
		return ext
	}
	if chunkType == store.ChunkTypeCommit {
		return ""
	}
	return ""
}

func codeFence(content string) string {
	longest := 0
	run := 0
	for _, r := range content {
		if r == '`' {
			run++
			if run > longest {
				longest = run
			}
		} else {
			run = 0
		}
	}
	n := 3
	if longest >= n {
		n = longest + 1
	}
	return strings.Repeat("`", n)
}
