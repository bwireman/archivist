package dump_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bwireman/archivist/internal/dump"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/store"
)

func TestFormatDump(t *testing.T) {
	r := &dump.Result{
		Query: "authentication",
		Scope: "internal",
		Items: []dump.Item{
			{
				Chunk: store.Chunk{
					Path:      "internal/auth.go",
					ChunkType: store.ChunkTypeCode,
					StartLine: 1,
					EndLine:   4,
					Content:   "package auth\n\nfunc Check() {}\n",
					Metadata:  map[string]string{"blame_author": "ada", "blame_commit": "abc123"},
				},
				Score: 0.91,
			},
		},
	}
	out := dump.Format(r)
	for _, want := range []string{
		"# Archivist context dump",
		"Query: authentication",
		"Scope: internal",
		"`internal/auth.go`",
		"score 0.910",
		"Blame: ada (abc123)",
		"```go",
		"func Check() {}",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected %q in dump:\n%s", want, out)
		}
	}
}

func TestFormatUsesLongerFence(t *testing.T) {
	r := &dump.Result{
		Items: []dump.Item{{
			Chunk: store.Chunk{
				Path:      "readme.md",
				ChunkType: store.ChunkTypeDoc,
				Content:   "Example:\n```\ncode\n```\n",
			},
		}},
	}
	out := dump.Format(r)
	if !strings.Contains(out, "````md\n") {
		t.Fatalf("expected a 4-backtick fence around nested fences:\n%s", out)
	}
}

func TestCollectAllAndQuery(t *testing.T) {
	st := openStore(t)
	fake := &embed.FakeEmbedder{Dim: 8}
	ctx := context.Background()
	now := time.Now().UTC()

	authEmb, err := fake.Embed(ctx, "authentication middleware")
	if err != nil {
		t.Fatal(err)
	}
	dbEmb, err := fake.Embed(ctx, "database connection pool")
	if err != nil {
		t.Fatal(err)
	}

	insert := func(path, content string, emb []float32) {
		t.Helper()
		if _, err := st.InsertChunk(store.Chunk{
			Path: path, ChunkType: store.ChunkTypeCode,
			StartLine: 1, EndLine: 2,
			Content: content, ContentHash: path,
			Embedding: emb, CreatedAt: now,
		}); err != nil {
			t.Fatal(err)
		}
	}
	insert("internal/auth.go", "authentication middleware", authEmb)
	insert("internal/db.go", "database connection pool", dbEmb)
	insert("cmd/main.go", "authentication middleware", authEmb)

	all, err := dump.Collect(ctx, st, nil, dump.Options{Scope: "internal"})
	if err != nil {
		t.Fatal(err)
	}
	if len(all.Items) != 2 {
		t.Fatalf("expected 2 scoped chunks, got %d", len(all.Items))
	}

	ranked, err := dump.Collect(ctx, st, fake, dump.Options{
		Query: "authentication",
		Scope: "internal",
		TopK:  1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ranked.Items) != 1 {
		t.Fatalf("expected 1 query result, got %d", len(ranked.Items))
	}
	if ranked.Items[0].Chunk.Path != "internal/auth.go" {
		t.Fatalf("expected internal/auth.go, got %s", ranked.Items[0].Chunk.Path)
	}

	if _, err := dump.Collect(ctx, st, nil, dump.Options{Query: "authentication"}); err == nil {
		t.Fatal("expected error when dumping a query without an embedder")
	}
}

func TestWriteStdoutFileAndDir(t *testing.T) {
	r := &dump.Result{
		Query: "pool",
		Items: []dump.Item{
			{Chunk: store.Chunk{Path: "internal/db.go", ChunkType: store.ChunkTypeCode, StartLine: 1, EndLine: 2, Content: "pool"}},
			{Chunk: store.Chunk{Path: "cmd/main.go", ChunkType: store.ChunkTypeCode, StartLine: 3, EndLine: 4, Content: "main"}},
		},
	}

	var buf bytes.Buffer
	written, err := dump.Write(r, "-", &buf)
	if err != nil {
		t.Fatal(err)
	}
	if !written.Stdout || !strings.Contains(buf.String(), "internal/db.go") {
		t.Fatalf("stdout dump: %#v\n%s", written, buf.String())
	}

	dir := t.TempDir()
	file := filepath.Join(dir, "context.md")
	written, err = dump.Write(r, file, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(written.Files) != 1 {
		t.Fatalf("expected 1 file, got %v", written.Files)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "cmd/main.go") {
		t.Fatalf("single file missing chunk:\n%s", data)
	}

	outDir := filepath.Join(dir, "dumps") + string(filepath.Separator)
	written, err = dump.Write(r, outDir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(written.Files) != 3 {
		t.Fatalf("expected index + 2 files, got %v", written.Files)
	}
	index, err := os.ReadFile(filepath.Join(dir, "dumps", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(index), "internal/db.go.md") {
		t.Fatalf("index missing file entry:\n%s", index)
	}
	split, err := os.ReadFile(filepath.Join(dir, "dumps", "internal", "db.go.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(split), "pool") {
		t.Fatalf("split file missing content:\n%s", split)
	}
}

func openStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}
