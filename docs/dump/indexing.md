# Archivist context dump

Retrieved chunks from a local code index. Use only facts present here.

- Query: incremental indexing skip dump files
- Chunks: 20

## `.cursor/rules/archivist-index.mdc` (doc, lines 6-19, score 0.631)

````
# Index regularly

Keep the Archivist index current. Run `make index` (needs Ollama) when you finish a turn that changed indexed files: Go sources, markdown docs, ADRs under `docs/decisions/`, or anything else the indexer walks.

```bash
make index
```

- Do this at the end of the turn, not only after writing an ADR
- Skip if you only touched `docs/dump/`, `.archivist/`, or files the indexer ignores
- Skip if you already indexed those changes this turn
- If Ollama is down, say so and continue; do not retry in a loop
- After indexing new or updated ADRs, also dump: `make dump-docs`
````


## `.cursor/rules/archivist-dump.mdc` (doc, lines 1-5, score 0.609)

```
---
description: Dump indexed decisions into docs/dump for later LLM sessions
alwaysApply: true
---
```


## `.cursor/rules/archivist-index.mdc` (doc, lines 1-5, score 0.609)

```
---
description: Regularly re-index the repository with Archivist
alwaysApply: true
---
```


## `.cursor/rules/archivist-decisions.mdc` (doc, lines 17-21, score 0.597)

```
# Use SQLite for the local index

- Status: accepted
- Date: 2026-08-28
```


## `.cursor/rules/archivist-dump.mdc` (doc, lines 6-22, score 0.560)

````
# Dump docs

Archivist does not generate documentation. After recording a decision, or when wrapping up work that changed architecture, dump indexed context into `docs/dump/`.

```bash
make refresh-docs                         # index, then export all ADRs
./archivist dump --type adr -o docs/dump/decisions.md
./archivist dump "sqlite vs postgres" -o docs/dump/sqlite.md
```

- `make dump-docs` writes `docs/dump/decisions.md` from indexed ADRs (no Ollama)
- Index first (`make index`) so new ADRs and code show up; Ollama is required for that
- Query dumps are for one topic; use a short filename under `docs/dump/`
- Do not hand-edit `docs/dump/` — regenerate it
- Do not write dumps into `docs/decisions/` (that directory is source ADRs only)
- `docs/dump/` is not indexed, so dumps will not feed back into the index
````


## `.cursor/rules/archivist-consult.mdc` (doc, lines 6-23, score 0.547)

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


## `92e4a604557abb93d0d1ecf0acada7bac73b2782` (commit, no line range, score 0.529)

```
Commit: 92e4a604557abb93d0d1ecf0acada7bac73b2782
Author: bwireman
Date: 2026-08-28T18:24:52-05:00
Subject: lets start doing this

Files:
.cursor/rules/archivist-decisions.mdc
.cursor/rules/archivist-dump.mdc
.cursor/rules/archivist-index.mdc
Makefile
docs/decisions/.gitkeep
docs/dump/.gitkeep
internal/config/config.go
internal/index/indexer.go
```


## `internal/index/indexer_test.go` (code, lines 149-172, score 0.528)

Blame: bwireman (b98ffc6e8be04621393bec1b26f10a649a2e8eed)

```go
File: internal/index/indexer_test.go

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
```


## `internal/index/indexer.go` (code, lines 95-99, score 0.518)

Blame: bwireman (b98ffc6e8be04621393bec1b26f10a649a2e8eed)

```go
File: internal/index/indexer.go

package index

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/chunk"
	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/gitindex"
	"github.com/bwireman/archivist/internal/store"
)

type Indexer struct {
	RepoRoot string
	Cfg      *config.Config
	Store    *store.Store
	Embedder embed.Embedder
}

func (idx *Indexer) Index(ctx context.Context, scopePath string) error {
	root := idx.RepoRoot
	if scopePath != "" {

func isDumpPath(rel string) bool {
	rel = filepath.ToSlash(rel)
	dumpDir := filepath.ToSlash(config.DefaultDumpDir)
	return rel == dumpDir || strings.HasPrefix(rel, dumpDir+"/")
}
```


## `internal/index/indexer.go` (code, lines 83-93, score 0.513)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/index/indexer.go

package index

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/chunk"
	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/gitindex"
	"github.com/bwireman/archivist/internal/store"
)

type Indexer struct {
	RepoRoot string
	Cfg      *config.Config
	Store    *store.Store
	Embedder embed.Embedder
}

func (idx *Indexer) Index(ctx context.Context, scopePath string) error {
	root := idx.RepoRoot
	if scopePath != "" {

func (idx *Indexer) shouldSkipFile(rel string) bool {
	rel = filepath.ToSlash(rel)
	if isDumpPath(rel) {
		return true
	}
	ext := strings.ToLower(filepath.Ext(rel))
	if binaryExts[ext] {
		return true
	}
	return chunk.MatchAnyPattern(rel, idx.Cfg.Index.SkipGlobs)
}
```


## `internal/index/indexer.go` (code, lines 69-81, score 0.510)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/index/indexer.go

package index

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/chunk"
	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/gitindex"
	"github.com/bwireman/archivist/internal/store"
)

type Indexer struct {
	RepoRoot string
	Cfg      *config.Config
	Store    *store.Store
	Embedder embed.Embedder
}

func (idx *Indexer) Index(ctx context.Context, scopePath string) error {
	root := idx.RepoRoot
	if scopePath != "" {

func (idx *Indexer) shouldSkipDir(path string) bool {
	base := filepath.Base(path)
	for _, skip := range idx.Cfg.Index.SkipDirs {
		if base == skip {
			return true
		}
	}
	rel, err := filepath.Rel(idx.RepoRoot, path)
	if err != nil {
		return false
	}
	return isDumpPath(rel)
}
```


## `internal/index/indexer_test.go` (code, lines 124-147, score 0.508)

Blame: bwireman (b98ffc6e8be04621393bec1b26f10a649a2e8eed)

```go
File: internal/index/indexer_test.go

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
```


## `internal/config/testdata/full.json` (code, lines 1-31, score 0.506)

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


## `.cursor/rules/archivist-decisions.mdc` (doc, lines 1-5, score 0.505)

```
---
description: Record architecture decisions as ADRs that Archivist indexes
alwaysApply: true
---
```


## `internal/index/indexer_test.go` (code, lines 57-89, score 0.496)

Blame: bwireman (b98ffc6e8be04621393bec1b26f10a649a2e8eed)

```go
File: internal/index/indexer_test.go

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
```


## `.archivist.json` (code, lines 1-25, score 0.489)

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


## `.cursor/rules/archivist-consult.mdc` (doc, lines 1-5, score 0.485)

```
---
description: Consult Archivist, then docs, before doing work
alwaysApply: true
---
```


## `internal/index/indexer.go` (code, lines 191-221, score 0.484)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/index/indexer.go

package index

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/chunk"
	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/gitindex"
	"github.com/bwireman/archivist/internal/store"
)

type Indexer struct {
	RepoRoot string
	Cfg      *config.Config
	Store    *store.Store
	Embedder embed.Embedder
}

func (idx *Indexer) Index(ctx context.Context, scopePath string) error {
	root := idx.RepoRoot
	if scopePath != "" {

func (idx *Indexer) indexGit(ctx context.Context) error {
	commits, err := gitindex.ListCommits(idx.RepoRoot, 500)
	if err != nil {
		return err
	}
	for _, c := range commits {
		if err := ctx.Err(); err != nil {
			return err
		}
		if _, ok, err := idx.Store.GetCommit(c.Hash); err != nil {
			return err
		} else if ok {
			continue
		}
		stored, err := idx.embedChunks(ctx, gitindex.CommitChunks(c))
		if err != nil {
			return err
		}
		if err := idx.Store.ReplaceCommit(store.CommitRecord{
			Hash:       c.Hash,
			Subject:    c.Subject,
			Body:       c.Body,
			Author:     c.Author,
			AuthoredAt: c.AuthoredAt,
			IndexedAt:  time.Now().UTC(),
		}, stored); err != nil {
			return err
		}
	}
	return nil
}
```


## `docs/decisions/001-use-qwen3-embedding.md` (adr, lines 12-16, score 0.483)

```md
## Consequences
- Indexing this repo works without pulling `nomic-embed-text`.
- Other repos created with `archivist init` still get `nomic-embed-text` unless they override it.
- Changing the embed model later requires a full reindex; existing vectors are not comparable across models.
```


## `internal/index/indexer.go` (code, lines 21-26, score 0.479)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/index/indexer.go

package index

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/chunk"
	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/gitindex"
	"github.com/bwireman/archivist/internal/store"
)

type Indexer struct {
	RepoRoot string
	Cfg      *config.Config
	Store    *store.Store
	Embedder embed.Embedder
}

func (idx *Indexer) Index(ctx context.Context, scopePath string) error {
	root := idx.RepoRoot
	if scopePath != "" {

type Indexer struct {
	RepoRoot string
	Cfg      *config.Config
	Store    *store.Store
	Embedder embed.Embedder
}
```

