package search_test

import (
	"context"
	"testing"
	"time"

	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/search"
	"github.com/bwireman/archivist/internal/store"
)

func TestSearchRanking(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	fake := &embed.FakeEmbedder{Dim: 8}
	ctx := context.Background()

	authEmb, err := fake.Embed(ctx, "authentication middleware")
	if err != nil {
		t.Fatal(err)
	}
	dbEmb, err := fake.Embed(ctx, "database connection pool")
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().UTC()
	_, err = st.InsertChunk(store.Chunk{
		Path: "auth.go", ChunkType: store.ChunkTypeCode,
		Content: "authentication middleware", ContentHash: "a",
		Embedding: authEmb, CreatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.InsertChunk(store.Chunk{
		Path: "db.go", ChunkType: store.ChunkTypeCode,
		Content: "database connection pool", ContentHash: "b",
		Embedding: dbEmb, CreatedAt: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	results, err := search.Search(ctx, st, fake, "authentication", search.Options{TopK: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Chunk.Path != "auth.go" {
		t.Fatalf("expected auth.go, got %s", results[0].Chunk.Path)
	}
}
