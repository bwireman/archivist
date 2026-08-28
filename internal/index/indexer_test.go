package index_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/index"
	"github.com/bwireman/archivist/internal/store"
)

type failEmbedder struct {
	embed.FakeEmbedder
	fail error
}

func (f *failEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	if f.fail != nil {
		return nil, f.fail
	}
	return f.FakeEmbedder.Embed(ctx, text)
}

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func newIndexer(t *testing.T, root string, emb embed.Embedder) (*index.Indexer, *store.Store) {
	t.Helper()
	cfg := config.Default()
	cfg.Index.SkipGlobs = []string{"*.pb.go"}
	st, err := store.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return &index.Indexer{
		RepoRoot: root,
		Cfg:      cfg,
		Store:    st,
		Embedder: emb,
	}, st
}

func TestIndexFailedEmbedKeepsPreviousChunks(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "foo.go", "package foo\n\nfunc Old() {}\n")

	fake := &failEmbedder{FakeEmbedder: embed.FakeEmbedder{Dim: 8}}
	idx, st := newIndexer(t, root, fake)
	if err := idx.Index(context.Background(), ""); err != nil {
		t.Fatal(err)
	}

	writeFile(t, root, "foo.go", "package foo\n\nfunc New() {}\n")
	fake.fail = errors.New("ollama down")
	if err := idx.Index(context.Background(), ""); err == nil {
		t.Fatal("expected embed error")
	}

	chunks, err := st.AllChunks()
	if err != nil {
		t.Fatal(err)
	}
	foundOld := false
	for _, c := range chunks {
		if c.Path == "foo.go" && strings.Contains(c.Content, "Old") {
			foundOld = true
		}
		if c.Path == "foo.go" && strings.Contains(c.Content, "func New") {
			t.Fatal("failed reindex replaced chunks")
		}
	}
	if !foundOld {
		t.Fatalf("expected previous chunks to remain, got %#v", chunks)
	}
}

func TestIndexScopeDoesNotPruneSiblingPrefix(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/a.go", "package internal\n\nfunc A() {}\n")
	writeFile(t, root, "internalize/b.go", "package internalize\n\nfunc B() {}\n")

	idx, st := newIndexer(t, root, &embed.FakeEmbedder{Dim: 8})
	if err := idx.Index(context.Background(), ""); err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(filepath.Join(root, "internal/a.go")); err != nil {
		t.Fatal(err)
	}
	if err := idx.Index(context.Background(), "internal"); err != nil {
		t.Fatal(err)
	}

	paths, err := st.AllFilePaths()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, p := range paths {
		got[p] = true
	}
	if got["internal/a.go"] {
		t.Fatal("deleted file should be pruned from scoped index")
	}
	if !got["internalize/b.go"] {
		t.Fatal("scoped index should not prune internalize/")
	}
}

func TestIndexSkipGlobs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "pkg/api.pb.go", "package api\n")
	writeFile(t, root, "pkg/api.go", "package api\n\nfunc API() {}\n")

	idx, st := newIndexer(t, root, &embed.FakeEmbedder{Dim: 8})
	if err := idx.Index(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	paths, err := st.AllFilePaths()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, p := range paths {
		got[p] = true
	}
	if got["pkg/api.pb.go"] {
		t.Fatal("*.pb.go should be skipped in subdirectories")
	}
	if !got["pkg/api.go"] {
		t.Fatal("api.go should be indexed")
	}
}

func TestIndexReportsProgress(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "foo.go", "package foo\n\nfunc Foo() {}\n")

	idx, _ := newIndexer(t, root, &embed.FakeEmbedder{Dim: 8})
	var last index.Progress
	var sawScan, sawFiles bool
	idx.Reporter = func(p index.Progress) {
		last = p
		if p.Phase == index.PhaseScan {
			sawScan = true
		}
		if p.Phase == index.PhaseFiles {
			sawFiles = true
		}
	}
	if err := idx.Index(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	if !sawScan || !sawFiles {
		t.Fatal("expected scan and files phases")
	}
	if last.Phase != index.PhaseDone {
		t.Fatalf("last phase %s", last.Phase)
	}
	if last.FilesIndexed < 1 {
		t.Fatalf("expected indexed files, got %+v", last)
	}
	if last.FilesTotal < 1 {
		t.Fatalf("expected files total, got %+v", last)
	}
}

func TestIndexDropsFileThatBecameBinary(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "data.bin", "package data\n\nfunc Data() {}\n")

	idx, st := newIndexer(t, root, &embed.FakeEmbedder{Dim: 8})
	if err := idx.Index(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "data.bin"), []byte{0, 1, 2, 0, 3}, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := idx.Index(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	paths, err := st.AllFilePaths()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		if p == "data.bin" {
			t.Fatal("binary file should be removed from the index")
		}
	}
}
