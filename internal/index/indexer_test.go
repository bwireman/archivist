package index_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/index"
	"github.com/bwireman/archivist/internal/store"
)

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

func newIndexer(t *testing.T, root string) (*index.Indexer, *store.Store, *store.Store) {
	t.Helper()
	cfg := config.Default()
	st, err := store.Open(filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	home, err := store.Open(filepath.Join(t.TempDir(), "home.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = st.Close()
		_ = home.Close()
	})
	return &index.Indexer{
		RepoRoot: root,
		Cfg:      cfg,
		Store:    st,
		Home:     home,
	}, st, home
}

func TestIndexCodeMap(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "foo.go", "package foo\n\nfunc Bar() {}\n")
	idx, st, _ := newIndexer(t, root)
	if err := idx.Index(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	syms, err := st.SymbolsForFile("foo.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(syms) == 0 {
		t.Fatal("expected symbols")
	}
}

func TestIndexRecord(t *testing.T) {
	root := t.TempDir()
	content := `---
id: rec_abc
type: decision
scope: repo
status: accepted
title: Use SQLite
---

## Decision
Yes.
`
	writeFile(t, root, "docs/decisions/001-sqlite.md", content)
	idx, st, _ := newIndexer(t, root)
	if err := idx.Index(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	rec, ok, err := st.GetRecordByID("rec_abc")
	if err != nil || !ok {
		t.Fatalf("record: ok=%v err=%v", ok, err)
	}
	if rec.Title != "Use SQLite" {
		t.Fatalf("title: %s", rec.Title)
	}
	depth, _ := st.QueueDepth()
	if depth != 1 {
		t.Fatalf("expected queue item, got %d", depth)
	}
}

func TestIndexSkipsArchive(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/archive/INDEX.md", "# index\n")
	idx, st, _ := newIndexer(t, root)
	if err := idx.Index(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	n, _ := st.FileCount()
	if n != 0 {
		t.Fatalf("expected archive skipped, file count %d", n)
	}
}

func TestFormatSummary(t *testing.T) {
	upToDate := index.FormatSummary(index.Progress{}, 215)
	if upToDate != "Index up to date (215 records)" {
		t.Fatalf("got %q", upToDate)
	}
	changed := index.FormatSummary(index.Progress{FilesIndexed: 3, CommitsNew: 1}, 40)
	if changed != "Indexed 3 files, 1 commit (40 records)" {
		t.Fatalf("got %q", changed)
	}
}
