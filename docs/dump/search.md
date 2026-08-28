# Archivist context dump

Retrieved chunks from a local code index. Use only facts present here.

- Query: semantic search cosine similarity
- Chunks: 20

## `.cursor/rules/archivist-dump.mdc` (doc, lines 1-5, score 0.547)

```
---
description: Dump indexed decisions into docs/dump for later LLM sessions
alwaysApply: true
---
```


## `docs/decisions/001-use-qwen3-embedding.md` (adr, lines 9-11, score 0.522)

```md
## Decision
Pin `.archivist.json` to `qwen3-embedding:0.6b` on `http://localhost:11434`. Leave the tool default in `config.Default()` as `nomic-embed-text`.
```


## `docs/decisions/001-use-qwen3-embedding.md` (adr, lines 1-5, score 0.517)

```md
# Use qwen3-embedding:0.6b for this repository

- Status: accepted
- Date: 2026-08-28
```


## `internal/search/search.go` (code, lines 26-63, score 0.504)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/search/search.go

package search

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/store"
)

type Result struct {
	Chunk store.Chunk
	Score float64
}

type Options struct {
	TopK  int
	Type  store.ChunkType
	Scope string
}

func Search(ctx context.Context, st *store.Store, embedder embed.Embedder, query string, opts Options) ([]Result, error) {
	if opts.TopK <= 0 {
		opts.TopK = 10
	}
	qEmb, err := embedder.Embed(ctx, query)

func Search(ctx context.Context, st *store.Store, embedder embed.Embedder, query string, opts Options) ([]Result, error) {
	if opts.TopK <= 0 {
		opts.TopK = 10
	}
	qEmb, err := embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}

	var chunks []store.Chunk
	if opts.Type != "" {
		chunks, err = st.ChunksByType(opts.Type)
	} else {
		chunks, err = st.AllChunks()
	}
	if err != nil {
		return nil, err
	}

	var results []Result
	for _, c := range chunks {
		if !MatchScope(opts.Scope, c.Path) {
			continue
		}
		if len(c.Embedding) == 0 {
			continue
		}
		score := store.CosineSimilarity(qEmb, c.Embedding)
		results = append(results, Result{Chunk: c, Score: score})
	}
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > opts.TopK {
		results = results[:opts.TopK]
	}
	return results, nil
}
```


## `.cursor/rules/archivist-consult.mdc` (doc, lines 6-23, score 0.502)

````
# Consult before working

Before implementing, researching, or changing architecture, look things up in this order. Do not guess APIs, defaults, or past decisions.

1. **Archivist** (needs Ollama). Search, then dump if you need full chunks:

```bash
./archivist search "how indexing skips dump files"
./archivist dump "how indexing skips dump files"
./archivist search --type adr "sqlite"
```

2. **Docs** next, using what the index pointed at:
   - `docs/decisions/` — source ADRs (authoritative)
   - `docs/dump/` — generated context, start with `docs/dump/decisions.md`

If Ollama is down, skip step 1 and read `docs/` directly. If both are empty or silent, read the code. Skip this lookup for typos, formatting, and one-line obvious fixes.
````


## `.archivist.json` (code, lines 1-25, score 0.502)

```json
{
  "ollama": {
    "base_url": "http://localhost:11434",
    "embed_model": "qwen3-embedding:0.6b",
    "embed_timeout": "2m"
  },
  "index": {
    "skip_dirs": [
      ".git",
      "vendor",
      "node_modules",
      ".archivist"
    ],
    "skip_globs": [],
    "adr_paths": [
      "**/adr/**",
      "docs/decisions/**",
      "**/ADR*.md"
    ]
  },
  "store": {
    "path": ".archivist/index.db"
  }
}
```


## `.cursor/rules/archivist-consult.mdc` (doc, lines 1-5, score 0.492)

```
---
description: Consult Archivist, then docs, before doing work
alwaysApply: true
---
```


## `internal/config/testdata/full.json` (code, lines 1-31, score 0.475)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```json
{
  "ollama": {
    "base_url": "http://ollama.example:11434",
    "embed_model": "mxbai-embed-large",
    "embed_timeout": "5m"
  },
  "index": {
    "skip_dirs": [
      ".git",
      "vendor",
      "node_modules",
      ".archivist",
      "dist",
      "build"
    ],
    "skip_globs": [
      "*.min.js",
      "*.pb.go"
    ],
    "adr_paths": [
      "**/adr/**",
      "docs/decisions/**",
      "**/ADR*.md",
      "architecture/decisions/**"
    ]
  },
  "store": {
    "path": ".archivist/custom-index.db"
  }
}
```


## `.cursor/rules/archivist-index.mdc` (doc, lines 1-5, score 0.461)

```
---
description: Regularly re-index the repository with Archivist
alwaysApply: true
---
```


## `internal/search/search.go` (code, lines 81-111, score 0.454)

Blame: bwireman (0f8e04b32575af25336945a0a2df91593c91b8e9)

```go
File: internal/search/search.go

package search

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/store"
)

type Result struct {
	Chunk store.Chunk
	Score float64
}

type Options struct {
	TopK  int
	Type  store.ChunkType
	Scope string
}

func Search(ctx context.Context, st *store.Store, embedder embed.Embedder, query string, opts Options) ([]Result, error) {
	if opts.TopK <= 0 {
		opts.TopK = 10
	}
	qEmb, err := embedder.Embed(ctx, query)

func MatchScope(scope, path string) bool {
	scope = filepath.ToSlash(strings.TrimSpace(scope))
	path = filepath.ToSlash(path)
	if scope == "" || scope == "." || scope == "./" {
		return true
	}
	scope = strings.TrimPrefix(scope, "./")
	path = strings.TrimPrefix(path, "./")
	if scope == path {
		return true
	}
	prefix := strings.TrimSuffix(scope, "/")
	prefix = strings.TrimSuffix(prefix, "/**")
	prefix = strings.TrimSuffix(prefix, "/*")
	if prefix != "" && (path == prefix || strings.HasPrefix(path, prefix+"/")) {
		return true
	}
	if matched, _ := filepath.Match(scope, path); matched {
		return true
	}
	if matched, _ := filepath.Match(scope, filepath.Base(path)); matched {
		return true
	}
	if strings.Contains(scope, "**") {
		glob := strings.ReplaceAll(scope, "**", "*")
		if matched, _ := filepath.Match(glob, path); matched {
			return true
		}
	}
	return false
}
```


## `internal/search/search_test.go` (code, lines 14-61, score 0.446)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/search/search_test.go

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
```


## `internal/search/search_test.go` (code, lines 79-113, score 0.444)

Blame: bwireman (0f8e04b32575af25336945a0a2df91593c91b8e9)

```go
File: internal/search/search_test.go

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
```


## `.cursor/rules/archivist-decisions.mdc` (doc, lines 1-5, score 0.441)

```
---
description: Record architecture decisions as ADRs that Archivist indexes
alwaysApply: true
---
```


## `.gitignore` (code, lines 1-4, score 0.436)

Blame: bwireman (b98ffc6e8be04621393bec1b26f10a649a2e8eed)

```
/archivist
/main
.archivist/
```


## `internal/search/search.go` (code, lines 113-125, score 0.436)

Blame: bwireman (b98ffc6e8be04621393bec1b26f10a649a2e8eed)

```go
File: internal/search/search.go

package search

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/store"
)

type Result struct {
	Chunk store.Chunk
	Score float64
}

type Options struct {
	TopK  int
	Type  store.ChunkType
	Scope string
}

func Search(ctx context.Context, st *store.Store, embedder embed.Embedder, query string, opts Options) ([]Result, error) {
	if opts.TopK <= 0 {
		opts.TopK = 10
	}
	qEmb, err := embedder.Embed(ctx, query)

func truncateBytes(s string, n int) string {
	if n <= 0 || s == "" {
		return ""
	}
	if len(s) <= n {
		return s
	}
	i := n
	for i > 0 && !utf8.RuneStart(s[i]) {
		i--
	}
	return s[:i] + "..."
}
```


## `internal/search/search.go` (code, lines 15-18, score 0.434)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/search/search.go

package search

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/store"
)

type Result struct {
	Chunk store.Chunk
	Score float64
}

type Options struct {
	TopK  int
	Type  store.ChunkType
	Scope string
}

func Search(ctx context.Context, st *store.Store, embedder embed.Embedder, query string, opts Options) ([]Result, error) {
	if opts.TopK <= 0 {
		opts.TopK = 10
	}
	qEmb, err := embedder.Embed(ctx, query)

type Result struct {
	Chunk store.Chunk
	Score float64
}
```


## `internal/search/search.go` (code, lines 20-24, score 0.428)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/search/search.go

package search

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/store"
)

type Result struct {
	Chunk store.Chunk
	Score float64
}

type Options struct {
	TopK  int
	Type  store.ChunkType
	Scope string
}

func Search(ctx context.Context, st *store.Store, embedder embed.Embedder, query string, opts Options) ([]Result, error) {
	if opts.TopK <= 0 {
		opts.TopK = 10
	}
	qEmb, err := embedder.Embed(ctx, query)

type Options struct {
	TopK  int
	Type  store.ChunkType
	Scope string
}
```


## `internal/search/search_test.go` (code, lines 115-134, score 0.426)

Blame: bwireman (0f8e04b32575af25336945a0a2df91593c91b8e9)

```go
File: internal/search/search_test.go

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
```


## `internal/search/search.go` (code, lines 65-77, score 0.421)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/search/search.go

package search

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/store"
)

type Result struct {
	Chunk store.Chunk
	Score float64
}

type Options struct {
	TopK  int
	Type  store.ChunkType
	Scope string
}

func Search(ctx context.Context, st *store.Store, embedder embed.Embedder, query string, opts Options) ([]Result, error) {
	if opts.TopK <= 0 {
		opts.TopK = 10
	}
	qEmb, err := embedder.Embed(ctx, query)

func FormatResults(results []Result) string {
	var b strings.Builder
	for i, r := range results {
		snippet := strings.ReplaceAll(truncateBytes(r.Chunk.Content, 200), "\n", " ")
		fmt.Fprintf(&b, "%d. [%.3f] %s %s:%d-%d\n   %s\n",
			i+1, r.Score, r.Chunk.ChunkType, r.Chunk.Path,
			r.Chunk.StartLine, r.Chunk.EndLine, snippet)
	}
	if len(results) == 0 {
		b.WriteString("No results.\n")
	}
	return b.String()
}
```


## `internal/search/search_test.go` (code, lines 63-77, score 0.408)

Blame: bwireman (b98ffc6e8be04621393bec1b26f10a649a2e8eed)

```go
File: internal/search/search_test.go

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
}
```

