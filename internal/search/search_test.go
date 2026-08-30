package search_test

import (
	"context"
	"strings"
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

func TestFormatResults(t *testing.T) {
	out := search.FormatResults([]search.Result{{
		Chunk: store.Chunk{
			Path: "foo.go", ChunkType: store.ChunkTypeCode,
			StartLine: 1, EndLine: 4, Content: "func Foo() {}",
		},
		Score: 0.91,
	}})
	if strings.Count(out, "foo.go") != 1 {
		t.Fatalf("path should appear once:\n%s", out)
	}
	if !strings.Contains(out, "code foo.go:1-4") {
		t.Fatalf("unexpected format:\n%s", out)
	}
	if !strings.Contains(out, "   func Foo() {}") {
		t.Fatalf("expected indented snippet:\n%s", out)
	}
}

func TestFormatResultsKeepsNewlines(t *testing.T) {
	out := search.FormatResults([]search.Result{{
		Chunk: store.Chunk{
			Path: "foo.go", ChunkType: store.ChunkTypeCode,
			StartLine: 1, EndLine: 4,
			Content: "package foo\n\nfunc Foo() {}",
		},
		Score: 0.91,
	}})
	if !strings.Contains(out, "   package foo\n") {
		t.Fatalf("expected multi-line snippet:\n%s", out)
	}
	if strings.Contains(out, "package foo  func Foo") {
		t.Fatalf("should not flatten newlines:\n%s", out)
	}
}

func TestSearchScope(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	fake := &embed.FakeEmbedder{Dim: 8}
	ctx := context.Background()
	emb, err := fake.Embed(ctx, "authentication middleware")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for _, path := range []string{"internal/auth.go", "cmd/auth.go"} {
		if _, err := st.InsertChunk(store.Chunk{
			Path: path, ChunkType: store.ChunkTypeCode,
			Content: "authentication middleware", ContentHash: path,
			Embedding: emb, CreatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}

	results, err := search.Search(ctx, st, fake, "authentication", search.Options{
		TopK:  5,
		Scope: "internal",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Chunk.Path != "internal/auth.go" {
		t.Fatalf("scoped search: %#v", results)
	}
}

func TestMatchScope(t *testing.T) {
	cases := []struct {
		scope string
		path  string
		want  bool
	}{
		{"", "internal/foo.go", true},
		{"internal", "internal/foo.go", true},
		{"internal/", "internal/foo.go", true},
		{"internal/**", "internal/store/foo.go", true},
		{"*.go", "foo.go", true},
		{"cmd", "internal/foo.go", false},
		{"internal", "internalize/foo.go", false},
	}
	for _, tc := range cases {
		if got := search.MatchScope(tc.scope, tc.path); got != tc.want {
			t.Fatalf("MatchScope(%q, %q)=%v, want %v", tc.scope, tc.path, got, tc.want)
		}
	}
}

func TestMatchADRScope(t *testing.T) {
	repo := store.Chunk{ChunkType: store.ChunkTypeADR, Metadata: map[string]string{"adr_scope": "repo"}}
	global := store.Chunk{ChunkType: store.ChunkTypeADR, Metadata: map[string]string{"adr_scope": "global"}}
	legacy := store.Chunk{ChunkType: store.ChunkTypeADR}
	code := store.Chunk{ChunkType: store.ChunkTypeCode}

	if !search.MatchADRScope("", repo) || !search.MatchADRScope("", code) {
		t.Fatal("empty filter should match everything")
	}
	if !search.MatchADRScope("repo", repo) || !search.MatchADRScope("repo", legacy) {
		t.Fatal("repo filter should match repo and legacy ADRs")
	}
	if search.MatchADRScope("repo", global) || search.MatchADRScope("repo", code) {
		t.Fatal("repo filter should exclude global ADRs and non-ADRs")
	}
	if !search.MatchADRScope("global", global) {
		t.Fatal("global filter should match global ADRs")
	}
	if search.MatchADRScope("global", repo) || search.MatchADRScope("global", legacy) {
		t.Fatal("global filter should exclude repo ADRs")
	}
}

func TestSearchADRScope(t *testing.T) {
	st, err := store.Open(t.TempDir() + "/test.db")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	fake := &embed.FakeEmbedder{Dim: 8}
	ctx := context.Background()
	emb, err := fake.Embed(ctx, "use sqlite")
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for _, c := range []store.Chunk{
		{
			Path: "docs/decisions/001.md", ChunkType: store.ChunkTypeADR,
			Content: "use sqlite", ContentHash: "r", Embedding: emb, CreatedAt: now,
			Metadata: map[string]string{"adr_scope": "repo"},
		},
		{
			Path: "docs/global-decisions/001.md", ChunkType: store.ChunkTypeADR,
			Content: "use sqlite", ContentHash: "g", Embedding: emb, CreatedAt: now,
			Metadata: map[string]string{"adr_scope": "global"},
		},
	} {
		if _, err := st.InsertChunk(c); err != nil {
			t.Fatal(err)
		}
	}

	results, err := search.Search(ctx, st, fake, "sqlite", search.Options{
		TopK:     5,
		Type:     store.ChunkTypeADR,
		ADRScope: "global",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Chunk.Path != "docs/global-decisions/001.md" {
		t.Fatalf("adr-scope search: %#v", results)
	}
}

func TestFormatChunkType(t *testing.T) {
	if got := search.FormatChunkType(store.Chunk{ChunkType: store.ChunkTypeCode}); got != "code" {
		t.Fatalf("code: %s", got)
	}
	got := search.FormatChunkType(store.Chunk{
		ChunkType: store.ChunkTypeADR,
		Metadata:  map[string]string{"adr_scope": "global"},
	})
	if got != "adr/global" {
		t.Fatalf("global adr: %s", got)
	}
}
