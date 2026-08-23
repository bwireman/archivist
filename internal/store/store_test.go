package store_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/bwireman/archivist/internal/store"
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
