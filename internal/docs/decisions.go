package docs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/search"
	"github.com/bwireman/archivist/internal/store"
)

type DecisionInput struct {
	Title   string   `json:"title"`
	Summary string   `json:"summary"`
	Context string   `json:"context,omitempty"`
	Status  string   `json:"status,omitempty"`
	Files   []string `json:"files,omitempty"`
}

type DecisionResult struct {
	Path         string   `json:"path"`
	Title        string   `json:"title"`
	Status       string   `json:"status"`
	RelatedFiles []string `json:"related_files"`
	Content      string   `json:"content,omitempty"`
	DryRun       bool     `json:"dry_run"`
}

type RecordDecisionOptions struct {
	RepoRoot string
	Cfg      *config.Config
	Store    *store.Store
	Embedder embed.Embedder
	Gen      embed.Generator
	Input    DecisionInput
	DryRun   bool
	Output   string
	TopK     int
}

func DecisionsDir(cfg *config.Config) string {
	return filepath.Join(cfg.Docs.Root, "decisions")
}

func RecordDecision(ctx context.Context, opts RecordDecisionOptions) (*DecisionResult, error) {
	if strings.TrimSpace(opts.Input.Title) == "" {
		return nil, fmt.Errorf("title is required")
	}
	if strings.TrimSpace(opts.Input.Summary) == "" {
		return nil, fmt.Errorf("summary is required")
	}

	status := opts.Input.Status
	if status == "" {
		status = "accepted"
	}

	related, err := gatherRelatedCode(ctx, opts)
	if err != nil {
		return nil, err
	}

	content, err := renderDecision(ctx, opts, status, related)
	if err != nil {
		return nil, err
	}

	outPath := opts.Output
	if outPath == "" {
		outPath, err = nextDecisionPath(opts.RepoRoot, DecisionsDir(opts.Cfg), opts.Input.Title)
		if err != nil {
			return nil, err
		}
	}

	result := &DecisionResult{
		Path:         outPath,
		Title:        opts.Input.Title,
		Status:       status,
		RelatedFiles: relatedPaths(related),
		DryRun:       opts.DryRun,
	}
	if opts.DryRun {
		result.Content = content
		return result, nil
	}

	abs := filepath.Join(opts.RepoRoot, outPath)
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		return nil, err
	}
	return result, nil
}

type relatedChunk struct {
	Path      string
	StartLine int
	EndLine   int
	Content   string
	Source    string
}

func gatherRelatedCode(ctx context.Context, opts RecordDecisionOptions) ([]relatedChunk, error) {
	seen := make(map[string]struct{})
	var related []relatedChunk

	add := func(path string, start, end int, content, source string) {
		key := fmt.Sprintf("%s:%d:%d", path, start, end)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		related = append(related, relatedChunk{
			Path: path, StartLine: start, EndLine: end, Content: content, Source: source,
		})
	}

	for _, file := range opts.Input.Files {
		file = filepath.ToSlash(strings.TrimSpace(file))
		if file == "" {
			continue
		}
		abs := file
		if !filepath.IsAbs(abs) {
			abs = filepath.Join(opts.RepoRoot, file)
		}
		data, err := os.ReadFile(abs)
		if err != nil {
			return nil, fmt.Errorf("read file %s: %w", file, err)
		}
		lines := strings.Count(string(data), "\n") + 1
		add(file, 1, lines, string(data), "explicit")
	}

	query := strings.TrimSpace(strings.Join([]string{
		opts.Input.Title,
		opts.Input.Summary,
		opts.Input.Context,
	}, "\n"))
	topK := opts.TopK
	if topK <= 0 {
		topK = 8
	}

	results, err := search.Search(ctx, opts.Store, opts.Embedder, query, search.Options{
		TopK: topK,
		Type: store.ChunkTypeCode,
	})
	if err != nil {
		return nil, err
	}
	for _, r := range results {
		add(r.Chunk.Path, r.Chunk.StartLine, r.Chunk.EndLine, r.Chunk.Content, "search")
	}

	commitResults, err := search.Search(ctx, opts.Store, opts.Embedder, query, search.Options{
		TopK: 3,
		Type: store.ChunkTypeCommit,
	})
	if err != nil {
		return nil, err
	}
	for _, r := range commitResults {
		add(r.Chunk.Path, 0, 0, r.Chunk.Content, "commit")
	}

	return related, nil
}

func relatedPaths(chunks []relatedChunk) []string {
	seen := make(map[string]struct{})
	var paths []string
	for _, c := range chunks {
		if _, ok := seen[c.Path]; ok {
			continue
		}
		seen[c.Path] = struct{}{}
		paths = append(paths, c.Path)
	}
	sort.Strings(paths)
	return paths
}

func nextDecisionPath(repoRoot, decisionsDir, title string) (string, error) {
	absDir := filepath.Join(repoRoot, decisionsDir)
	if err := os.MkdirAll(absDir, 0o755); err != nil {
		return "", err
	}
	entries, err := os.ReadDir(absDir)
	if err != nil {
		return "", err
	}

	re := regexp.MustCompile(`^(\d+)-.*\.md$`)
	maxNum := 0
	for _, e := range entries {
		m := re.FindStringSubmatch(e.Name())
		if len(m) != 2 {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err == nil && n > maxNum {
			maxNum = n
		}
	}

	slug := slugify(title)
	return filepath.ToSlash(filepath.Join(decisionsDir, fmt.Sprintf("%03d-%s.md", maxNum+1, slug))), nil
}

func slugify(title string) string {
	s := strings.ToLower(strings.TrimSpace(title))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "decision"
	}
	return s
}

func renderDecision(ctx context.Context, opts RecordDecisionOptions, status string, related []relatedChunk) (string, error) {
	if opts.Gen != nil {
		prompt := buildDecisionPrompt(opts.Input, status, related)
		generated, err := opts.Gen.Generate(ctx, prompt)
		if err == nil && strings.TrimSpace(generated) != "" {
			return strings.TrimSpace(generated) + "\n", nil
		}
	}
	return renderDecisionTemplate(opts.Input, status, related), nil
}

func buildDecisionPrompt(input DecisionInput, status string, related []relatedChunk) string {
	var b strings.Builder
	b.WriteString(`You are writing an Architecture Decision Record (ADR) in markdown.

Rules:
- Only use facts from the provided decision input and code context.
- Do not invent APIs, files, or behavior not present in the context.
- Include a "Relevant code" section with fenced code blocks for important snippets.
- Use this structure:
  # <title>
  - Status: <status>
  - Date: <date>
  ## Context
  ## Decision
  ## Consequences
  ## Relevant code

`)
	fmt.Fprintf(&b, "Decision input:\nTitle: %s\nSummary: %s\n", input.Title, input.Summary)
	if input.Context != "" {
		fmt.Fprintf(&b, "Bot context:\n%s\n", input.Context)
	}
	fmt.Fprintf(&b, "Status: %s\nDate: %s\n\nCode context:\n", status, time.Now().UTC().Format("2006-01-02"))
	for _, c := range related {
		fmt.Fprintf(&b, "---\n[%s] %s:%d-%d\n%s\n", c.Source, c.Path, c.StartLine, c.EndLine, c.Content)
	}
	b.WriteString("\nWrite the complete ADR markdown:\n")
	return b.String()
}

func renderDecisionTemplate(input DecisionInput, status string, related []relatedChunk) string {
	var b strings.Builder
	date := time.Now().UTC().Format("2006-01-02")
	fmt.Fprintf(&b, "# %s\n\n", input.Title)
	fmt.Fprintf(&b, "- Status: %s\n", status)
	fmt.Fprintf(&b, "- Date: %s\n\n", date)
	fmt.Fprintf(&b, "## Context\n\n%s\n", input.Summary)
	if strings.TrimSpace(input.Context) != "" {
		fmt.Fprintf(&b, "\n%s\n", strings.TrimSpace(input.Context))
	}
	fmt.Fprintf(&b, "\n## Decision\n\n%s\n", input.Summary)
	fmt.Fprintf(&b, "\n## Consequences\n\n- Documented by archivist from observed development activity.\n")
	fmt.Fprintf(&b, "\n## Relevant code\n\n")
	for _, c := range related {
		lang := codeFenceLang(c.Path)
		if c.StartLine > 0 {
			fmt.Fprintf(&b, "### `%s` (lines %d-%d)\n\n", c.Path, c.StartLine, c.EndLine)
		} else {
			fmt.Fprintf(&b, "### `%s`\n\n", c.Path)
		}
		fmt.Fprintf(&b, "```%s\n%s\n```\n\n", lang, strings.TrimSpace(c.Content))
	}
	return b.String()
}

func codeFenceLang(path string) string {
	ext := strings.TrimPrefix(filepath.Ext(path), ".")
	switch ext {
	case "go", "py", "js", "ts", "tsx", "jsx", "rs", "java", "json", "yaml", "yml", "md", "sh", "bash":
		return ext
	default:
		return ""
	}
}

func ParseDecisionInputJSON(data []byte) (DecisionInput, error) {
	var input DecisionInput
	if err := json.Unmarshal(data, &input); err != nil {
		return DecisionInput{}, fmt.Errorf("parse decision JSON: %w", err)
	}
	return input, nil
}
