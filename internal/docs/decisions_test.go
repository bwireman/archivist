package docs_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/docs"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/store"
)

func TestRecordDecisionDryRun(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	st, err := store.Open(filepath.Join(root, ".archivist", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	_, err = st.InsertChunk(store.Chunk{
		Path: "internal/store/store.go", ChunkType: store.ChunkTypeCode,
		StartLine: 1, EndLine: 5,
		Content:     "package store\n\nfunc Open(path string) {}",
		ContentHash: "hash1",
		Embedding:   []float32{1, 0, 0, 0, 0, 0, 0, 0},
		CreatedAt:   time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}

	fake := &embed.FakeEmbedder{Dim: 8}
	gen := &embed.FakeGenerator{Response: "# Use SQLite\n\n- Status: accepted\n\n## Context\n\nFor local index.\n"}

	storeFile := filepath.Join(root, "internal", "store")
	if err := os.MkdirAll(storeFile, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(storeFile, "store.go"), []byte("package store\n\nfunc Open(path string) {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := docs.RecordDecision(context.Background(), docs.RecordDecisionOptions{
		RepoRoot: root,
		Cfg:      cfg,
		Store:    st,
		Embedder: fake,
		Gen:      gen,
		Input: docs.DecisionInput{
			Title:   "Use SQLite for local index",
			Summary: "Store embeddings in SQLite for inspectability and portability.",
			Context: "Bot noticed repeated discussion of embedded storage tradeoffs.",
			Files:   []string{"internal/store/store.go"},
		},
		DryRun: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	if !result.DryRun {
		t.Fatal("expected dry run result")
	}
	if !strings.HasPrefix(result.Path, "docs/decisions/001-") {
		t.Fatalf("unexpected path: %s", result.Path)
	}
	if len(result.RelatedFiles) == 0 {
		t.Fatal("expected related files")
	}
	if result.Content == "" {
		t.Fatal("expected content in dry-run output")
	}

	if _, err := os.Stat(filepath.Join(root, result.Path)); !os.IsNotExist(err) {
		t.Fatal("dry run should not write file")
	}
}

func TestRecordDecisionWritesFile(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	st, err := store.Open(filepath.Join(root, ".archivist", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	fake := &embed.FakeEmbedder{Dim: 8}
	result, err := docs.RecordDecision(context.Background(), docs.RecordDecisionOptions{
		RepoRoot: root,
		Cfg:      cfg,
		Store:    st,
		Embedder: fake,
		Gen:      nil,
		Input: docs.DecisionInput{
			Title:   "Prefer JSON config",
			Summary: "Use JSON instead of YAML for configuration files.",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(root, result.Path))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, want := range []string{"# Prefer JSON config", "## Context", "## Decision", "## Relevant code"} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected %q in written ADR:\n%s", want, content)
		}
	}
}

func TestRecordDecisionIncrementsNumber(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	st, err := store.Open(filepath.Join(root, ".archivist", "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	dir := filepath.Join(root, docs.DecisionsDir(cfg))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "001-first.md"), []byte("# first\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	fake := &embed.FakeEmbedder{Dim: 8}
	result, err := docs.RecordDecision(context.Background(), docs.RecordDecisionOptions{
		RepoRoot: root,
		Cfg:      cfg,
		Store:    st,
		Embedder: fake,
		Input: docs.DecisionInput{
			Title:   "Second decision",
			Summary: "Another decision.",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Path != "docs/decisions/002-second-decision.md" {
		t.Fatalf("unexpected next path: %s", result.Path)
	}
}

func TestParseDecisionInputJSON(t *testing.T) {
	raw := `{
		"title": "Use tree-sitter",
		"summary": "Parse code with tree-sitter for semantic chunks.",
		"context": "Bot observed polyglot chunking requirement.",
		"status": "proposed",
		"files": ["internal/chunk/treesitter.go"]
	}`
	input, err := docs.ParseDecisionInputJSON([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if input.Title != "Use tree-sitter" || input.Status != "proposed" || len(input.Files) != 1 {
		t.Fatalf("unexpected input: %#v", input)
	}
}
