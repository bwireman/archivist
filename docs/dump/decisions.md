# Archivist context dump

Retrieved chunks from a local code index. Use only facts present here.

- Query: (all matching chunks)
- Type: adr
- Chunks: 12

## `docs/decisions/001-use-qwen3-embedding.md` (adr, lines 1-5)

```md
# Use qwen3-embedding:0.6b for this repository

- Status: accepted
- Date: 2026-08-28
```


## `docs/decisions/001-use-qwen3-embedding.md` (adr, lines 6-8)

```md
## Context
`archivist init` writes `nomic-embed-text` as the default Ollama embed model. This repository needed a model that is actually installed locally so `make index` can run.
```


## `docs/decisions/001-use-qwen3-embedding.md` (adr, lines 9-11)

```md
## Decision
Pin `.archivist.json` to `qwen3-embedding:0.6b` on `http://localhost:11434`. Leave the tool default in `config.Default()` as `nomic-embed-text`.
```


## `docs/decisions/001-use-qwen3-embedding.md` (adr, lines 12-16)

```md
## Consequences
- Indexing this repo works without pulling `nomic-embed-text`.
- Other repos created with `archivist init` still get `nomic-embed-text` unless they override it.
- Changing the embed model later requires a full reindex; existing vectors are not comparable across models.
```


## `docs/decisions/002-bubbletea-index-tui.md` (adr, lines 1-5)

```md
# Use Bubble Tea for the index TUI

- Status: accepted
- Date: 2026-08-28
```


## `docs/decisions/002-bubbletea-index-tui.md` (adr, lines 6-8)

```md
## Context
`archivist index` printed one line when it finished. Embedding is per-chunk and can take a while, so a live view of the current file and counts is useful. `make index` and piped runs still need a log-friendly summary.
```


## `docs/decisions/002-bubbletea-index-tui.md` (adr, lines 9-11)

```md
## Decision
Show a Bubble Tea TUI when stdout is a TTY. Fall back to a one-line summary when it is not, or when `--plain` is set (`make index` uses `--plain`). The indexer reports progress through a callback so tests do not need a terminal.
```


## `docs/decisions/002-bubbletea-index-tui.md` (adr, lines 12-16)

```md
## Consequences
- Interactive runs of `archivist index` show scan/file/git progress and can cancel with q.
- `make index` passes `--plain` so agent and CI logs stay a single summary line.
- Charm libraries (bubbletea, bubbles, lipgloss) are dependencies of the CLI.
```


## `docs/decisions/003-huh-init-tui.md` (adr, lines 1-5)

```md
# Use Huh for the init TUI

- Status: accepted
- Date: 2026-08-28
```


## `docs/decisions/003-huh-init-tui.md` (adr, lines 6-8)

```md
## Context
`archivist init` wrote `config.Default()` with no prompts. The embed model is a real choice (`nomic-embed-text` vs a locally installed model). The index command already uses a Charm TUI on a TTY.
```


## `docs/decisions/003-huh-init-tui.md` (adr, lines 9-11)

```md
## Decision
Show a Huh form on a TTY so `archivist init` can edit Ollama, index, and store settings before writing `.archivist.json`. `--plain` writes `config.Default()` with no prompt.
```


## `docs/decisions/003-huh-init-tui.md` (adr, lines 12-16)

```md
## Consequences
- Interactive setup can set the embed model without editing JSON.
- Scripts and `--plain` still get the tool defaults.
- Huh is an additional Charm dependency.
```

