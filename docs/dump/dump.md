# Archivist context dump

Retrieved chunks from a local code index. Use only facts present here.

- Query: dump markdown llm context
- Chunks: 20

## `.cursor/rules/archivist-dump.mdc` (doc, lines 1-5, score 0.688)

```
---
description: Dump indexed decisions into docs/dump for later LLM sessions
alwaysApply: true
---
```


## `docs/decisions/001-use-qwen3-embedding.md` (adr, lines 1-5, score 0.561)

```md
# Use qwen3-embedding:0.6b for this repository

- Status: accepted
- Date: 2026-08-28
```


## `docs/decisions/001-use-qwen3-embedding.md` (adr, lines 9-11, score 0.555)

```md
## Decision
Pin `.archivist.json` to `qwen3-embedding:0.6b` on `http://localhost:11434`. Leave the tool default in `config.Default()` as `nomic-embed-text`.
```


## `.cursor/rules/archivist-consult.mdc` (doc, lines 6-23, score 0.549)

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


## `.archivist.json` (code, lines 1-25, score 0.546)

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


## `.cursor/rules/archivist-consult.mdc` (doc, lines 1-5, score 0.525)

```
---
description: Consult Archivist, then docs, before doing work
alwaysApply: true
---
```


## `.cursor/rules/archivist-dump.mdc` (doc, lines 6-22, score 0.508)

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


## `.cursor/rules/archivist-decisions.mdc` (doc, lines 28-36, score 0.505)

````
## Consequences
- What becomes easier
- What we are accepting
```

Status is `proposed`, `accepted`, `deprecated`, or `superseded`. Only state facts from this repo; do not invent APIs.

Then `make refresh-docs` (index + dump). If Ollama is down, still write the ADR; skip `make index` and dump whatever is already indexed.
````


## `internal/config/testdata/full.json` (code, lines 1-31, score 0.501)

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


## `.cursor/rules/archivist-decisions.mdc` (doc, lines 1-5, score 0.475)

```
---
description: Record architecture decisions as ADRs that Archivist indexes
alwaysApply: true
---
```


## `docs/decisions/001-use-qwen3-embedding.md` (adr, lines 6-8, score 0.462)

```md
## Context
`archivist init` writes `nomic-embed-text` as the default Ollama embed model. This repository needed a model that is actually installed locally so `make index` can run.
```


## `.cursor/rules/archivist-decisions.mdc` (doc, lines 22-24, score 0.459)

```
## Context
Why the question came up.
```


## `87bb4a48dc76cfc76681db7fc61a58171700050f` (commit, no line range, score 0.438)

```
Commit: 87bb4a48dc76cfc76681db7fc61a58171700050f
Author: bwireman
Date: 2026-08-28T18:39:14-05:00
Subject: Pin this repo to qwen3-embedding:0.6b.

The local Ollama instance does not have nomic-embed-text; keep that as the init default and override it here.

Files:
.archivist.json
docs/decisions/001-use-qwen3-embedding.md
```


## `.cursor/rules/archivist-decisions.mdc` (doc, lines 10-16, score 0.430)

````
## File

- Path: `docs/decisions/NNN-slug.md`
- `NNN` is the next number already used in that directory (`001`, `002`, …)
- `slug` is a lowercase hyphenated title

```markdown
````


## `.cursor/rules/archivist-index.mdc` (doc, lines 1-5, score 0.423)

```
---
description: Regularly re-index the repository with Archivist
alwaysApply: true
---
```


## `internal/cmd/root.go` (code, lines 199-281, score 0.422)

Blame: bwireman (0f8e04b32575af25336945a0a2df91593c91b8e9)

```go
File: internal/cmd/root.go

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/dump"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/index"
	"github.com/bwireman/archivist/internal/search"
	"github.com/bwireman/archivist/internal/store"
	"github.com/spf13/cobra"
)

var (
	repoPath string
)

func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "archivist",
		Short:         "Local code indexer",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&repoPath, "path", ".", "repository root path")

func newDumpCmd() *cobra.Command {
	var (
		output    string
		scope     string
		chunkType string
		topK      int
	)
	cmd := &cobra.Command{
		Use:   "dump [query]",
		Short: "Dump indexed context for an LLM",
		Long: `Write retrieved index chunks as markdown you can paste into an LLM.

With a query, dump the closest matching chunks. Without a query, dump all
indexed chunks (optionally limited by --scope and --type).

Output:
  (default)     stdout, so you can pipe the dump into another command
  -o file.md    a single markdown file
  -o dumps/     one markdown file per source path, plus index.md
                (trailing slash, or an existing directory)

Examples:
  archivist dump "how does indexing work"
  archivist dump "auth middleware" -o context.md
  archivist dump --scope internal -o dumps/
  archivist dump --type adr
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			st, err := openStore(root, cfg)
			if err != nil {
				return err
			}
			defer st.Close()

			query := strings.TrimSpace(strings.Join(args, " "))
			opts := dump.Options{
				Query:  query,
				Scope:  scope,
				TopK:   topK,
				Output: output,
			}
			if chunkType != "" {
				opts.Type = store.ChunkType(chunkType)
			}

			var embedder embed.Embedder
			if query != "" {
				client := embed.NewOllamaClientFromConfig(cfg.Ollama)
				if err := client.Healthy(cmd.Context()); err != nil {
					return fmt.Errorf("%w (run: ollama serve)", err)
				}
				embedder = client
			}

			result, err := dump.Collect(cmd.Context(), st, embedder, opts)
			if err != nil {
				return err
			}
			written, err := dump.Write(result, output, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			if written.Stdout {
				return nil
			}
			if len(written.Files) == 1 {
				fmt.Fprintf(cmd.ErrOrStderr(), "Wrote %s (%d chunks)\n", written.Files[0], len(result.Items))
				return nil
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Wrote %d files (%d chunks)\n", len(written.Files), len(result.Items))
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "write to a file, a directory (trailing /), or stdout (default)")
	cmd.Flags().StringVar(&scope, "scope", "", "limit to a path prefix or glob")
	cmd.Flags().StringVar(&chunkType, "type", "", "filter by chunk type: code|doc|commit|adr|comment")
	cmd.Flags().IntVar(&topK, "top", 0, "max chunks (default 20 with a query, all without)")
	return cmd
}
```


## `internal/embed/embedder.go` (code, lines 9-12, score 0.421)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/embed/embedder.go

package embed

import (
	"context"

	"github.com/bwireman/archivist/internal/config"
)

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Dimensions() int
}

type Client interface {
	Embedder
	Healthy(ctx context.Context) error
}

type HealthStatus struct {
	EmbedderOK    bool   `json:"embedder_ok"`
	EmbedderError string `json:"embedder_error,omitempty"`
}

func CheckHealth(ctx context.Context, cfg *config.Config) HealthStatus {
	var status HealthStatus
	client := NewOllamaClientFromConfig(cfg.Ollama)
	if err := client.Healthy(ctx); err != nil {
		status.EmbedderError = err.Error()
	} else {
		status.EmbedderOK = true

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Dimensions() int
}
```


## `91f3e743fe67b5f66747d3602f127ae019418b22` (commit, no line range, score 0.420)

```
Commit: 91f3e743fe67b5f66747d3602f127ae019418b22
Author: bwireman
Date: 2026-08-28T18:35:35-05:00
Subject: consult

Files:
.cursor/rules/archivist-consult.mdc
```


## `internal/dump/dump.go` (code, lines 313-331, score 0.415)

Blame: bwireman (0f8e04b32575af25336945a0a2df91593c91b8e9)

```go
File: internal/dump/dump.go

package dump

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/search"
	"github.com/bwireman/archivist/internal/store"
)

const DefaultTopK = 20

type Options struct {
	Query  string
	Scope  string
	Type   store.ChunkType
	TopK   int
	Output string
}

type Item struct {
	Chunk store.Chunk
	Score float64
}

func codeFence(content string) string {
	longest := 0
	run := 0
	for _, r := range content {
		if r == '`' {
			run++
			if run > longest {
				longest = run
			}
		} else {
			run = 0
		}
	}
	n := 3
	if longest >= n {
		n = longest + 1
	}
	return strings.Repeat("`", n)
}
```


## `internal/chunk/chunk_test.go` (comment, line 22, score 0.413)

```go
content := "package main\n\n// TODO: fix this\nfunc main() {}\n"
```

