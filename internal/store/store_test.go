package store_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bwireman/archivist/internal/store"
	"github.com/bwireman/archivist/internal/version"
)

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	emb := []float32{0.1, 0.2, 0.3, 0.4}
	id, err := st.InsertChunk(store.Chunk{
		Path:        "foo.go",
		ChunkType:   store.ChunkTypeCode,
		StartLine:   1,
		EndLine:     10,
		Content:     "func Foo() {}",
		ContentHash: "abc",
		Embedding:   emb,
		Metadata:    map[string]string{"node_type": "function_declaration"},
		CreatedAt:   time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if id == 0 {
		t.Fatal("expected chunk id")
	}

	chunks, err := st.AllChunks()
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}
	if len(chunks[0].Embedding) != 4 {
		t.Fatalf("expected 4-dim embedding, got %d", len(chunks[0].Embedding))
	}

	if err := st.UpsertFile(store.FileRecord{
		Path:        "foo.go",
		ContentHash: "abc",
		IndexedAt:   time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteFile("foo.go"); err != nil {
		t.Fatal(err)
	}
	count, err := st.ChunkCount()
	if err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected 0 chunks after delete, got %d", count)
	}
}

func TestReplaceFileChunksIsAtomicReplace(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	now := time.Now().UTC()
	if err := st.ReplaceFileChunks(store.FileRecord{
		Path: "foo.go", ContentHash: "old", IndexedAt: now,
	}, []store.Chunk{{
		Path: "foo.go", ChunkType: store.ChunkTypeCode,
		Content: "old", ContentHash: "old", CreatedAt: now,
		Embedding: []float32{1, 0},
	}}); err != nil {
		t.Fatal(err)
	}

	if err := st.ReplaceFileChunks(store.FileRecord{
		Path: "foo.go", ContentHash: "new", IndexedAt: now,
	}, []store.Chunk{{
		Path: "foo.go", ChunkType: store.ChunkTypeCode,
		Content: "new", ContentHash: "new", CreatedAt: now,
		Embedding: []float32{0, 1},
	}}); err != nil {
		t.Fatal(err)
	}

	chunks, err := st.AllChunks()
	if err != nil {
		t.Fatal(err)
	}
	if len(chunks) != 1 || chunks[0].Content != "new" {
		t.Fatalf("expected replaced chunk, got %#v", chunks)
	}
	rec, ok, err := st.GetFile("foo.go")
	if err != nil || !ok {
		t.Fatalf("file: ok=%v err=%v", ok, err)
	}
	if rec.ContentHash != "new" {
		t.Fatalf("hash: %q", rec.ContentHash)
	}
}

func TestReplaceCommit(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	now := time.Now().UTC()
	if err := st.ReplaceCommit(store.CommitRecord{
		Hash: "abc", Subject: "s", Author: "a", AuthoredAt: now, IndexedAt: now,
	}, []store.Chunk{{
		Path: "abc", ChunkType: store.ChunkTypeCommit,
		Content: "commit abc", ContentHash: "abc", CreatedAt: now,
		Embedding: []float32{1, 0},
	}}); err != nil {
		t.Fatal(err)
	}
	got, ok, err := st.GetCommit("abc")
	if err != nil || !ok || got.Subject != "s" {
		t.Fatalf("commit: ok=%v err=%v %#v", ok, err, got)
	}
	chunks, err := st.ChunksByType(store.ChunkTypeCommit)
	if err != nil || len(chunks) != 1 {
		t.Fatalf("chunks: %v %v", chunks, err)
	}
}

func TestCosineSimilarity(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{1, 0, 0}
	if store.CosineSimilarity(a, b) < 0.99 {
		t.Fatal("identical vectors should have similarity ~1")
	}
	c := []float32{0, 1, 0}
	if store.CosineSimilarity(a, c) > 0.01 {
		t.Fatal("orthogonal vectors should have similarity ~0")
	}
}

func TestOpenStampsSchema(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	got, err := st.SchemaVersion()
	if err != nil {
		t.Fatal(err)
	}
	if got != version.Schema {
		t.Fatalf("schema: got %d want %d", got, version.Schema)
	}
	_, ok, err := st.ArchivistVersion()
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("archivist_version should be set when indexing, not on open")
	}
}

func TestStampIndexedWritesVersion(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	now := time.Now().UTC().Truncate(time.Second)
	if err := st.StampIndexed(now); err != nil {
		t.Fatal(err)
	}
	got, ok, err := st.ArchivistVersion()
	if err != nil || !ok || got != version.Version {
		t.Fatalf("archivist_version: ok=%v got=%q err=%v", ok, got, err)
	}
	indexed, ok, err := st.LastIndexedAt()
	if err != nil || !ok {
		t.Fatalf("last_indexed_at: ok=%v err=%v", ok, err)
	}
	if !indexed.Equal(now) {
		t.Fatalf("last_indexed_at: got %s want %s", indexed, now)
	}
}

func TestOpenRejectsNewerSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetMeta(store.MetaSchemaVersion, "99"); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = store.Open(path)
	var se *store.SchemaError
	if !errors.As(err, &se) {
		t.Fatalf("expected SchemaError, got %v", err)
	}
	if se.Have != 99 || se.Want != version.Schema {
		t.Fatalf("SchemaError: %#v", se)
	}
}

func TestOpenMigratesMissingSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.SetMeta(store.MetaSchemaVersion, "0"); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	st, err = store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	got, err := st.SchemaVersion()
	if err != nil {
		t.Fatal(err)
	}
	if got != version.Schema {
		t.Fatalf("schema after migrate: got %d want %d", got, version.Schema)
	}
}

func TestFirstChunk(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	_, ok, err := st.FirstChunk("missing.md")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected no chunk")
	}

	if err := st.ReplaceFileChunks(store.FileRecord{
		Path: "docs/global-decisions/001.md", ContentHash: "h", IndexedAt: time.Now().UTC(),
	}, []store.Chunk{{
		Path:      "docs/global-decisions/001.md",
		ChunkType: store.ChunkTypeADR,
		Content:   "hello",
		Metadata:  map[string]string{"origin_root": "/tmp/repo"},
		CreatedAt: time.Now().UTC(),
	}}); err != nil {
		t.Fatal(err)
	}
	c, ok, err := st.FirstChunk("docs/global-decisions/001.md")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("expected chunk")
	}
	if c.Metadata["origin_root"] != "/tmp/repo" {
		t.Fatalf("metadata %v", c.Metadata)
	}

	all, err := st.ChunksForPath("docs/global-decisions/001.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Fatalf("ChunksForPath: got %d", len(all))
	}
}

func TestOpenIfExists(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.db")
	st, ok, err := store.OpenIfExists(missing)
	if err != nil {
		t.Fatal(err)
	}
	if ok || st != nil {
		t.Fatal("missing db should not open")
	}
	if _, err := os.Stat(missing); !os.IsNotExist(err) {
		t.Fatal("OpenIfExists must not create the file")
	}

	path := filepath.Join(dir, "exists.db")
	created, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := created.Close(); err != nil {
		t.Fatal(err)
	}
	st, ok, err = store.OpenIfExists(path)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || st == nil {
		t.Fatal("expected existing db")
	}
	_ = st.Close()
}

func TestSchema2DropsFileChunksKeepsCommits(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := st.ReplaceFileChunks(store.FileRecord{
		Path: "foo.go", ContentHash: "h", IndexedAt: now,
	}, []store.Chunk{{
		Path: "foo.go", ChunkType: store.ChunkTypeCode,
		Content: "old", ContentHash: "c", CreatedAt: now,
	}}); err != nil {
		t.Fatal(err)
	}
	if err := st.ReplaceCommit(store.CommitRecord{
		Hash: "abc123", Subject: "init", Author: "ada", AuthoredAt: now, IndexedAt: now,
	}, []store.Chunk{{
		Path: "abc123", ChunkType: store.ChunkTypeCommit,
		Content: "commit", ContentHash: "g", CreatedAt: now,
	}}); err != nil {
		t.Fatal(err)
	}
	if err := st.SetMeta(store.MetaSchemaVersion, "1"); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	st, err = store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	got, err := st.SchemaVersion()
	if err != nil {
		t.Fatal(err)
	}
	if got != version.Schema {
		t.Fatalf("schema: got %d want %d", got, version.Schema)
	}
	files, err := st.AllFilePaths()
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("expected files dropped, got %v", files)
	}
	code, err := st.ChunksByType(store.ChunkTypeCode)
	if err != nil {
		t.Fatal(err)
	}
	if len(code) != 0 {
		t.Fatalf("expected code chunks dropped, got %d", len(code))
	}
	commits, err := st.ChunksByType(store.ChunkTypeCommit)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected commit chunk kept, got %d", len(commits))
	}
}
