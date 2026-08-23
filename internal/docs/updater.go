package docs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/search"
	"github.com/bwireman/archivist/internal/store"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type UpdateOptions struct {
	RepoRoot string
	Cfg      *config.Config
	Store    *store.Store
	Embedder embed.Embedder
	Gen      embed.Generator
	DryRun   bool
	All      bool
	Scope    string
}

type Page struct {
	SourceGlob string
	DocPath    string
}

func ResolvePages(repoRoot string, cfg *config.Config) ([]Page, error) {
	var pages []Page
	if len(cfg.Docs.Pages) > 0 {
		for src, doc := range cfg.Docs.Pages {
			pages = append(pages, Page{SourceGlob: src, DocPath: doc})
		}
		sort.Slice(pages, func(i, j int) bool {
			return pages[i].DocPath < pages[j].DocPath
		})
		return pages, nil
	}
	// default: one page per top-level directory
	entries, err := os.ReadDir(repoRoot)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if shouldSkipTopLevel(name) {
			continue
		}
		pages = append(pages, Page{
			SourceGlob: name + "/**",
			DocPath:    filepath.Join(cfg.Docs.Root, name+".md"),
		})
	}
	sort.Slice(pages, func(i, j int) bool {
		return pages[i].DocPath < pages[j].DocPath
	})
	return pages, nil
}

func shouldSkipTopLevel(name string) bool {
	skip := map[string]bool{
		".git": true, ".archivist": true, "vendor": true, "node_modules": true,
		"docs": true,
	}
	return skip[name]
}

func Update(ctx context.Context, opts UpdateOptions) error {
	pages, err := ResolvePages(opts.RepoRoot, opts.Cfg)
	if err != nil {
		return err
	}
	if len(pages) == 0 {
		return fmt.Errorf("no doc pages resolved")
	}

	syncAt, hasSync, err := opts.Store.DocsSyncAt()
	if err != nil {
		return err
	}

	for _, page := range pages {
		needsUpdate, err := pageNeedsUpdate(opts, page, syncAt, hasSync)
		if err != nil {
			return err
		}
		if !needsUpdate && !opts.All {
			continue
		}
		if err := updatePage(ctx, opts, page); err != nil {
			return err
		}
	}

	if !opts.DryRun {
		return opts.Store.SetDocsSyncAt(time.Now().UTC())
	}
	return nil
}

func pageNeedsUpdate(opts UpdateOptions, page Page, syncAt time.Time, hasSync bool) (bool, error) {
	if opts.All || !hasSync {
		return true, nil
	}
	prefix := strings.TrimSuffix(page.SourceGlob, "/**")
	prefix = strings.TrimSuffix(prefix, "/*")
	changed, err := opts.Store.ChunksChangedSince(syncAt)
	if err != nil {
		return false, err
	}
	for _, c := range changed {
		if matchSource(page.SourceGlob, c.Path) || strings.HasPrefix(c.Path, prefix) {
			return true, nil
		}
	}
	return false, nil
}

func matchSource(glob, path string) bool {
	if glob == path {
		return true
	}
	glob = strings.ReplaceAll(glob, "**", "*")
	matched, _ := filepath.Match(glob, path)
	if matched {
		return true
	}
	prefix := strings.TrimSuffix(glob, "/**")
	prefix = strings.TrimSuffix(prefix, "/*")
	return prefix != "" && strings.HasPrefix(path, prefix)
}

func updatePage(ctx context.Context, opts UpdateOptions, page Page) error {
	query := fmt.Sprintf("documentation for %s", page.SourceGlob)
	results, err := search.Search(ctx, opts.Store, opts.Embedder, query, search.Options{TopK: 20})
	if err != nil {
		return err
	}

	var contextParts []string
	for _, r := range results {
		if matchSource(page.SourceGlob, r.Chunk.Path) ||
			r.Chunk.ChunkType == store.ChunkTypeCommit ||
			r.Chunk.ChunkType == store.ChunkTypeADR {
			contextParts = append(contextParts, fmt.Sprintf("---\n[%s] %s\n%s", r.Chunk.ChunkType, r.Chunk.Path, r.Chunk.Content))
		}
	}

	docAbs := filepath.Join(opts.RepoRoot, page.DocPath)
	existing, _ := os.ReadFile(docAbs)
	if len(existing) > 0 {
		contextParts = append(contextParts, "---\n[current doc]\n"+string(existing))
	}

	prompt := buildPrompt(page, contextParts)
	generated, err := opts.Gen.Generate(ctx, prompt)
	if err != nil {
		return fmt.Errorf("generate %s: %w", page.DocPath, err)
	}
	generated = strings.TrimSpace(generated)

	if opts.DryRun {
		dmp := diffmatchpatch.New()
		diffs := dmp.DiffMain(string(existing), generated, false)
		fmt.Printf("=== %s ===\n", page.DocPath)
		fmt.Println(dmp.DiffPrettyText(diffs))
		fmt.Println()
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(docAbs), 0o755); err != nil {
		return err
	}
	return os.WriteFile(docAbs, []byte(generated+"\n"), 0o644)
}

func buildPrompt(page Page, contextParts []string) string {
	var b strings.Builder
	b.WriteString(`You are a technical documentation writer. Write complete markdown documentation for the given scope.

Rules:
- Only use facts from the provided context chunks.
- If something is unknown, say "Unknown" rather than inventing APIs or behavior.
- Use this structure:
  1. Purpose
  2. How it works
  3. Decisions and tradeoffs
  4. How to change it

`)
	fmt.Fprintf(&b, "Scope: %s\nOutput file: %s\n\nContext:\n%s\n\nWrite the full markdown document:\n",
		page.SourceGlob, page.DocPath, strings.Join(contextParts, "\n"))
	return b.String()
}
