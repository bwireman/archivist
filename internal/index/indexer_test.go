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
	gst, err := store.Open(filepath.Join(t.TempDir(), "global.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = st.Close()
		_ = gst.Close()
	})
	return &index.Indexer{
		RepoRoot:      root,
		Cfg:           cfg,
		Store:         st,
		Global:        gst,
		Embedder:      emb,
		UserGlobalDir: filepath.Join(t.TempDir(), "no-user-global"),
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

func TestIndexRepoAndGlobalADRs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "docs/decisions/001-repo.md", "# Repo\n\nUse SQLite.\n")
	writeFile(t, root, "docs/global-decisions/001-global.md", "# Global\n\nUse Bubble Tea.\n")

	idx, st := newIndexer(t, root, &embed.FakeEmbedder{Dim: 8})
	if err := idx.Index(context.Background(), ""); err != nil {
		t.Fatal(err)
	}

	chunks, err := st.AllChunks()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, c := range chunks {
		if c.ChunkType != store.ChunkTypeADR {
			continue
		}
		got[c.Path] = c.Metadata["adr_scope"]
	}
	if got["docs/decisions/001-repo.md"] != "repo" {
		t.Fatalf("repo ADR: %v", got)
	}
	if _, ok := got["docs/global-decisions/001-global.md"]; ok {
		t.Fatalf("in-repo global ADR must not land in the repo store: %v", got)
	}

	gchunks, err := idx.Global.AllChunks()
	if err != nil {
		t.Fatal(err)
	}
	ggot := map[string]store.Chunk{}
	for _, c := range gchunks {
		if c.ChunkType != store.ChunkTypeADR {
			continue
		}
		ggot[c.Path] = c
	}
	gc, ok := ggot["docs/global-decisions/001-global.md"]
	if !ok {
		t.Fatalf("global ADR missing from global store: %v", ggot)
	}
	if gc.Metadata["adr_scope"] != "global" {
		t.Fatalf("global ADR scope: %v", gc.Metadata)
	}
	if gc.Metadata["origin"] != "repo" {
		t.Fatalf("global ADR origin: %v", gc.Metadata)
	}
	if gc.Metadata["origin_root"] == "" {
		t.Fatal("expected origin_root on in-repo global ADR")
	}
}

func TestIndexUserGlobalADRs(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "foo.go", "package foo\n\nfunc Foo() {}\n")
	userDir := t.TempDir()
	writeFile(t, userDir, "001-home.md", "# Home\n\nA user-global ADR.\n")

	idx, st := newIndexer(t, root, &embed.FakeEmbedder{Dim: 8})
	idx.UserGlobalDir = userDir
	if err := idx.Index(context.Background(), ""); err != nil {
		t.Fatal(err)
	}

	paths, err := st.AllFilePaths()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		if p == "user/001-home.md" || p == "global/001-home.md" {
			t.Fatalf("user-global file must not land in the repo store, got %v", paths)
		}
	}

	gpaths, err := idx.Global.AllFilePaths()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, p := range gpaths {
		got[p] = true
	}
	if !got["user/001-home.md"] {
		t.Fatalf("expected user-global file in global store, got %v", gpaths)
	}

	chunks, err := idx.Global.ChunksByType(store.ChunkTypeADR)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range chunks {
		if c.Path == "user/001-home.md" {
			found = true
			if c.Metadata["adr_scope"] != "global" {
				t.Fatalf("user-global scope: %v", c.Metadata)
			}
			if c.Metadata["origin"] != "user" {
				t.Fatalf("user-global origin: %v", c.Metadata)
			}
		}
	}
	if !found {
		t.Fatal("expected ADR chunk for user-global file")
	}
}

func TestIndexScopeDoesNotPruneUserGlobal(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "internal/a.go", "package internal\n\nfunc A() {}\n")
	writeFile(t, root, "docs/global-decisions/001-keep.md", "# Keep\n\nIn-repo global.\n")
	userDir := t.TempDir()
	writeFile(t, userDir, "001-home.md", "# Home\n\nKeep me.\n")

	idx, st := newIndexer(t, root, &embed.FakeEmbedder{Dim: 8})
	idx.UserGlobalDir = userDir
	if err := idx.Index(context.Background(), ""); err != nil {
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
	if !got["internal/a.go"] {
		t.Fatal("expected internal/a.go")
	}

	gpaths, err := idx.Global.AllFilePaths()
	if err != nil {
		t.Fatal(err)
	}
	ggot := map[string]bool{}
	for _, p := range gpaths {
		ggot[p] = true
	}
	if !ggot["user/001-home.md"] {
		t.Fatal("scoped index should not prune user-global ADRs")
	}
	if !ggot["docs/global-decisions/001-keep.md"] {
		t.Fatal("scoped index should not prune other-path in-repo globals")
	}
}

func TestIndexDoesNotPruneOtherRepoGlobals(t *testing.T) {
	shared, err := store.Open(filepath.Join(t.TempDir(), "global.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = shared.Close() })

	rootA := t.TempDir()
	writeFile(t, rootA, "docs/global-decisions/001-a.md", "# A\n\nFrom repo A.\n")
	idxA, _ := newIndexer(t, rootA, &embed.FakeEmbedder{Dim: 8})
	idxA.Global = shared
	if err := idxA.Index(context.Background(), ""); err != nil {
		t.Fatal(err)
	}

	rootB := t.TempDir()
	writeFile(t, rootB, "foo.go", "package foo\n\nfunc Foo() {}\n")
	idxB, _ := newIndexer(t, rootB, &embed.FakeEmbedder{Dim: 8})
	idxB.Global = shared
	if err := idxB.Index(context.Background(), ""); err != nil {
		t.Fatal(err)
	}

	paths, err := shared.AllFilePaths()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, p := range paths {
		got[p] = true
	}
	if !got["docs/global-decisions/001-a.md"] {
		t.Fatalf("repo B must not prune repo A's global ADRs, got %v", paths)
	}

	if err := os.Remove(filepath.Join(rootA, "docs/global-decisions/001-a.md")); err != nil {
		t.Fatal(err)
	}
	if err := idxA.Index(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	paths, err = shared.AllFilePaths()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		if p == "docs/global-decisions/001-a.md" {
			t.Fatal("repo A should prune its own deleted global ADR")
		}
	}
}

func TestIndexRestampOriginRoot(t *testing.T) {
	shared, err := store.Open(filepath.Join(t.TempDir(), "global.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = shared.Close() })

	content := "# Shared\n\nSame bytes in both checkouts.\n"
	rootA := t.TempDir()
	writeFile(t, rootA, "docs/global-decisions/001.md", content)
	idxA, _ := newIndexer(t, rootA, &embed.FakeEmbedder{Dim: 8})
	idxA.Global = shared
	if err := idxA.Index(context.Background(), ""); err != nil {
		t.Fatal(err)
	}

	rootB := t.TempDir()
	writeFile(t, rootB, "docs/global-decisions/001.md", content)
	idxB, _ := newIndexer(t, rootB, &embed.FakeEmbedder{Dim: 8})
	idxB.Global = shared
	if err := idxB.Index(context.Background(), ""); err != nil {
		t.Fatal(err)
	}

	c, ok, err := shared.FirstChunk("docs/global-decisions/001.md")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected global chunk")
	}
	absB, err := filepath.Abs(rootB)
	if err != nil {
		t.Fatal(err)
	}
	if resolved, err := filepath.EvalSymlinks(absB); err == nil {
		absB = resolved
	}
	if c.Metadata["origin_root"] != absB {
		t.Fatalf("origin_root after hash-match reindex: got %q want %q", c.Metadata["origin_root"], absB)
	}

	if err := os.Remove(filepath.Join(rootB, "docs/global-decisions/001.md")); err != nil {
		t.Fatal(err)
	}
	if err := idxB.Index(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	paths, err := shared.AllFilePaths()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range paths {
		if p == "docs/global-decisions/001.md" {
			t.Fatal("repo B should prune after claiming origin_root")
		}
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

func TestIndexHonorGitignore(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".gitignore", "ignored.md\nsecret/\n")
	writeFile(t, root, "keep.go", "package keep\n\nfunc Keep() {}\n")
	writeFile(t, root, "ignored.md", "# secret\n")
	writeFile(t, root, "secret/x.go", "package secret\n")

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
	if !got["keep.go"] {
		t.Fatal("expected keep.go")
	}
	if got["ignored.md"] {
		t.Fatal(".gitignore ignored.md should be skipped")
	}
	if got["secret/x.go"] {
		t.Fatal(".gitignore secret/ should be skipped")
	}
}

func TestIndexHonorGitignoreDisabled(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, ".gitignore", "ignored.md\n")
	writeFile(t, root, "ignored.md", "# still index me\n")

	idx, st := newIndexer(t, root, &embed.FakeEmbedder{Dim: 8})
	idx.Cfg.Index.HonorGitignore = false
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
	if !got["ignored.md"] {
		t.Fatal("honor_gitignore false should still index ignored.md")
	}
}
