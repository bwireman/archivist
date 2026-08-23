package docs_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/docs"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/store"
)

func TestDocsUpdateDryRun(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.Docs.Pages = map[string]string{
		"internal/**": "docs/internal.md",
	}

	dbPath := filepath.Join(root, ".archivist", "index.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	_, err = st.InsertChunk(store.Chunk{
		Path:        "internal/foo.go",
		ChunkType:   store.ChunkTypeCode,
		StartLine:   1,
		EndLine:     5,
		Content:     "package internal\n\nfunc Foo() {}",
		ContentHash: "hash1",
		Embedding:   []float32{1, 0, 0, 0, 0, 0, 0, 0},
		CreatedAt:   time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}

	fake := &embed.FakeEmbedder{Dim: 8}
	gen := &embed.FakeGenerator{Response: "# Internal\n\nPurpose\n\nHow it works\n"}

	err = docs.Update(context.Background(), docs.UpdateOptions{
		RepoRoot: root,
		Cfg:      cfg,
		Store:    st,
		Embedder: fake,
		Gen:      gen,
		DryRun:   true,
		All:      true,
	})
	if err != nil {
		t.Fatal(err)
	}

	// file should not exist in dry-run mode
	if _, err := os.Stat(filepath.Join(root, "docs/internal.md")); !os.IsNotExist(err) {
		t.Fatal("expected no file written in dry-run mode")
	}
}

func TestResolvePagesDefault(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"cmd", "internal", "pkg"} {
		if err := os.MkdirAll(filepath.Join(root, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	cfg := config.Default()
	pages, err := docs.ResolvePages(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 3 {
		t.Fatalf("expected 3 default pages, got %d", len(pages))
	}
}
