package search

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/store"
)

type Result struct {
	Chunk store.Chunk
	Score float64
}

type Options struct {
	TopK   int
	Type   store.ChunkType
	Scope  string
	AsJSON bool
}

func Search(ctx context.Context, st *store.Store, embedder embed.Embedder, query string, opts Options) ([]Result, error) {
	if opts.TopK <= 0 {
		opts.TopK = 10
	}
	qEmb, err := embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	var chunks []store.Chunk
	if opts.Type != "" {
		chunks, err = st.ChunksByType(opts.Type)
	} else {
		chunks, err = st.AllChunks()
	}
	if err != nil {
		return nil, err
	}

	var results []Result
	for _, c := range chunks {
		if !MatchScope(opts.Scope, c.Path) {
			continue
		}
		if len(c.Embedding) == 0 {
			continue
		}
		score := store.CosineSimilarity(qEmb, c.Embedding)
		results = append(results, Result{Chunk: c, Score: score})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > opts.TopK {
		results = results[:opts.TopK]
	}
	return results, nil
}

func FormatResults(results []Result) string {
	var b strings.Builder
	for i, r := range results {
		snippet := r.Chunk.Content
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		snippet = strings.ReplaceAll(snippet, "\n", " ")
		fmt.Fprintf(&b, "%d. [%.3f] %s (%s) %s:%d-%d\n   %s\n",
			i+1, r.Score, r.Chunk.ChunkType, r.Chunk.Path,
			r.Chunk.Path, r.Chunk.StartLine, r.Chunk.EndLine, snippet)
	}
	if len(results) == 0 {
		b.WriteString("No results.\n")
	}
	return b.String()
}

// MatchScope reports whether path is inside scope. Scope may be a directory
// prefix (internal, internal/) or a glob (internal/**, *.go).
func MatchScope(scope, path string) bool {
	scope = filepath.ToSlash(strings.TrimSpace(scope))
	path = filepath.ToSlash(path)
	if scope == "" || scope == "." || scope == "./" {
		return true
	}
	scope = strings.TrimPrefix(scope, "./")
	path = strings.TrimPrefix(path, "./")
	if scope == path {
		return true
	}
	prefix := strings.TrimSuffix(scope, "/")
	prefix = strings.TrimSuffix(prefix, "/**")
	prefix = strings.TrimSuffix(prefix, "/*")
	if prefix != "" && (path == prefix || strings.HasPrefix(path, prefix+"/")) {
		return true
	}
	if matched, _ := filepath.Match(scope, path); matched {
		return true
	}
	if matched, _ := filepath.Match(scope, filepath.Base(path)); matched {
		return true
	}
	if strings.Contains(scope, "**") {
		glob := strings.ReplaceAll(scope, "**", "*")
		if matched, _ := filepath.Match(glob, path); matched {
			return true
		}
	}
	return false
}
