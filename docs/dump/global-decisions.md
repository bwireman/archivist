# Archivist context dump

Retrieved chunks from a local code index. Use only facts present here.

- Query: (all matching chunks)
- ADR scope: global
- Type: adr
- Chunks: 11

## `docs/global-decisions/001-bubbletea-index-tui.md` (adr/global, lines 1-17)

```md
File: docs/global-decisions/001-bubbletea-index-tui.md
Kind: adr

# Use Bubble Tea for the index TUI

- Status: accepted
- Date: 2026-08-28
- Scope: global

## Context
`archivist index` printed one line when it finished. Embedding is per-chunk and can take a while, so a live view of the current file and counts is useful. `make index` and piped runs still need a log-friendly summary.

## Decision
Show a Bubble Tea TUI when stdout is a TTY. Fall back to a one-line summary when it is not, or when `--plain` is set (`make index` uses `--plain`). The indexer reports progress through a callback so tests do not need a terminal.

## Consequences
- Interactive runs of `archivist index` show scan/file/git progress and can cancel with q.
- `make index` passes `--plain` so agent and CI logs stay a single summary line.
- Charm libraries (bubbletea, bubbles, lipgloss) are dependencies of the CLI.
```


## `docs/global-decisions/002-huh-init-tui.md` (adr/global, lines 1-17)

```md
File: docs/global-decisions/002-huh-init-tui.md
Kind: adr

# Use Huh for the init TUI

- Status: accepted
- Date: 2026-08-28
- Scope: global

## Context
`archivist init` wrote `config.Default()` with no prompts. The embed model is a real choice (`nomic-embed-text` vs a locally installed model). The index command already uses a Charm TUI on a TTY.

## Decision
Show a Huh form on a TTY so `archivist init` can edit Ollama, index, and store settings before writing `.archivist.json`. `--plain` writes `config.Default()` with no prompt.

## Consequences
- Interactive setup can set the embed model without editing JSON.
- Scripts and `--plain` still get the tool defaults.
- Huh is an additional Charm dependency.
```


## `docs/global-decisions/003-global-and-repo-adrs.md` (adr/global, lines 1-20)

```md
File: docs/global-decisions/003-global-and-repo-adrs.md
Kind: adr

# Separate global and repo ADRs

- Status: accepted
- Date: 2026-08-28
- Scope: global

## Context
Product decisions (index TUI, init TUI) were stored next to this checkout's repo-specific choices (embed model). Agents need to tell tool-wide decisions from project-local ones, and a user may keep ADRs that apply to every repo.

`--scope` already means a path prefix or glob. Reusing it for ADR kind would collide with dump and index flags.

## Decision
Classify ADRs as `repo` or `global`. Repo ADRs live in `docs/decisions/`. In-repo global ADRs live in `docs/global-decisions/` (a sibling directory, so `docs/decisions/**` does not match them). User-global ADRs live in `~/.archivist/decisions/`. Filter with `--adr-scope repo|global`. `make dump-docs` writes `docs/dump/decisions.md` and `docs/dump/global-decisions.md`. Storage is two SQLite files; see `006-separate-global-index.md`.

## Consequences
- Search and dump can list only repo or only global ADRs.
- Path globs are the source of truth for classification; a `- Scope:` line in the markdown is for humans.
- Existing ADR chunks without `adr_scope` metadata are treated as repo.
- `archivist init` creates the two in-repo directories. It does not create `~/.archivist/decisions/`.
```


## `docs/global-decisions/004-schema-and-cli-version.md` (adr/global, lines 1-18)

```md
File: docs/global-decisions/004-schema-and-cli-version.md
Kind: adr

# Record schema and CLI version

- Status: accepted
- Date: 2026-08-28
- Scope: global

## Context
The SQLite index already has a `meta` table (`last_indexed_at`). There was no record of which Archivist binary or store layout wrote it, so a later schema change would be silent.

## Decision
Keep a product `Version` (`0.1.0`, overridable at build with `-ldflags`) and an integer `Schema` (`1`). The CLI prints both via `--version` and `archivist version`. On open, the store migrates toward the current schema and writes `schema_version`. A newer database refuses to open. After a successful index, `archivist_version` is stored next to `last_indexed_at`. `archivist status` shows both.

## Consequences
- Existing indexes without `schema_version` are treated as schema 0 and migrated to 1 (current CREATE TABLE layout).
- Older CLIs error if they see a higher `schema_version`.
- Bump `Schema` and add a case in `applyMigrations` when the store contract changes; bump `Version` for releases.
- `make build VERSION=x.y.z` injects the product version; the default lives in `internal/version`.
```


## `docs/global-decisions/005-nested-adr-config.md` (adr/global, lines 1-17)

```md
File: docs/global-decisions/005-nested-adr-config.md
Kind: adr

# Nest ADR globs under index.adr

- Status: accepted
- Date: 2026-08-28
- Scope: global

## Context
Repo vs global ADRs were configured as sibling lists `adr_paths` and `global_adr_paths`. The names did not say what the lists were for, or how they related.

## Decision
Put both under `index.adr`, with `repo` and `global` glob lists. Global wins if a path matches both. `~/.archivist/decisions/` stays implicit (not a repo glob) and is indexed as `global/<file>`. Old `adr_paths` / `global_adr_paths` keys still load.

## Consequences
- `.archivist.json` reads as "these paths are repo ADRs, these are global ADRs".
- Save writes the nested shape only.
- Empty lists disable that kind; omitting `adr` keeps the defaults.
```


## `docs/global-decisions/006-separate-global-index.md` (adr/global, lines 1-20)

```md
File: docs/global-decisions/006-separate-global-index.md
Kind: adr

# Separate SQLite index for global ADRs

- Status: accepted
- Date: 2026-08-28
- Scope: global

## Context
In-repo `docs/global-decisions/` and `~/.archivist/decisions/` were stored in each checkout's `.archivist/index.db`. Default search mixed product decisions into project context, and one repo could not consult another checkout's global ADRs without also ingesting that repo's code.

## Decision
Keep two SQLite files. The repo index (`.archivist/index.db`) holds this checkout only: code, docs, git, and repo ADRs. The machine-wide index (`~/.archivist/global.db` by default) holds user ADRs plus files that match `index.adr.global`. `store.global_path` is optional; a relative value resolves under `~/.archivist/`, not the repo. Default search and dump use the repo DB. `--adr-scope global` (and `make dump-docs`'s global half) use the global DB. User ADRs are stored as `user/<file>`. In-repo global chunks stamp `origin=repo` and `origin_root=<abs repo>` so prune only removes this checkout's missing files. Git history is never written to the global DB.

## Consequences
- Agents can search global ADRs from any repo after that machine has indexed them, without pulling in another project's code.
- Default repo search no longer returns global ADRs.
- The same relative path from two products last-writer-wins in `global.db`.
- Global queries use the current embed model; vectors written with a different model will not rank well until reindexed.
- Untagged in-repo global chunks (no `origin_root`) are left in place until overwritten.
- Scoped `--scope` index does not prune in-repo globals outside that prefix.
```


## `docs/global-decisions/007-honor-gitignore.md` (adr/global, lines 1-17)

```md
File: docs/global-decisions/007-honor-gitignore.md
Kind: adr

# Honor .gitignore when indexing

- Status: accepted
- Date: 2026-08-28
- Scope: global

## Context
`skip_dirs` and `skip_globs` were a second, incomplete copy of what most repos already list in `.gitignore`. Build artifacts, local binaries, and editor dirs were easy to index by accident.

## Decision
`index.honor_gitignore` defaults to true. When set, the indexer skips files and directories that match the repo `.gitignore` (root and nested), including directory patterns so ignored trees are not walked. `skip_dirs` and `skip_globs` still apply. User-global ADRs under `~/.archivist/decisions/` are not filtered by a checkout's gitignore. Set the flag false to index gitignored paths.

## Consequences
- `archivist init` writes `honor_gitignore: true` and the setup TUI can turn it off.
- Omitting the key in an existing `.archivist.json` still means true.
- Gitignored files that were already in the index are pruned on the next index.
```


## `docs/global-decisions/008-default-adr-globs.md` (adr/global, lines 1-17)

```md
File: docs/global-decisions/008-default-adr-globs.md
Kind: adr

# Default ADR globs are the product directories

- Status: accepted
- Date: 2026-08-28
- Scope: global

## Context
Default `index.adr.repo` also matched `**/adr/**` and `**/ADR*.md` so other layouts would classify as ADRs. This product only uses `docs/decisions/` and `docs/global-decisions/`, which is what the Cursor rules and `archivist init` already create. Extra globs made the config look unrelated to those directories.

## Decision
Default `index.adr.repo` is `docs/decisions/**`. Default `index.adr.global` is `docs/global-decisions/**`. Other layouts stay possible as an explicit override. Empty lists still disable that kind; omitting `adr` still keeps these defaults.

## Consequences
- `.archivist.json` `index.adr` matches the directories agents write to.
- Files like `docs/ADR-001.md` are ordinary docs unless the repo adds a glob.
- `~/.archivist/decisions/` is still implicit, not a glob.
```


## `docs/global-decisions/009-qwen3-embed-default.md` (adr/global, lines 1-18)

```md
File: docs/global-decisions/009-qwen3-embed-default.md
Kind: adr

# Default embed model is qwen3-embedding:0.6b

- Status: accepted
- Date: 2026-08-28
- Scope: global

## Context
`archivist init` wrote `nomic-embed-text` via `config.Default()`. This checkout already indexes with `qwen3-embedding:0.6b` because that model is installed locally. New repos should get the same default so they can index without a per-repo pin.

## Decision
`config.Default()` and `archivist init --plain` use `qwen3-embedding:0.6b`. The init TUI still suggests `nomic-embed-text` and `mxbai-embed-large` as alternatives. A repo may override `ollama.embed_model` in `.archivist.json`.

## Consequences
- `ollama pull qwen3-embedding:0.6b` is the documented setup step.
- Existing configs that set `nomic-embed-text` keep that model.
- Changing the embed model later requires a full reindex; existing vectors are not comparable across models.
- The repo-only pin in `docs/decisions/001-use-qwen3-embedding.md` is superseded.
```


## `docs/global-decisions/010-richer-chunks-and-cli.md` (adr/global, lines 1-18)

```md
File: docs/global-decisions/010-richer-chunks-and-cli.md
Kind: adr

# Richer chunks and longer CLI results

- Status: accepted
- Date: 2026-08-29
- Scope: global

## Context
Chunks were capped at 2000 bytes and markdown/ADRs split on every heading, so hits were often a title or a five-line section. `archivist search` flattened each hit to 200 bytes on one line and defaulted to 10 results. Agents using search and dump did not get enough surrounding code or decision text.

## Decision
Default max chunk size is 4000 bytes with 400 bytes of overlap. Docs and ADRs pack consecutive sections until that size; they only split at a heading once the current chunk is already substantial. Every file chunk is prefixed with `File` and `Kind` (tree-sitter nodes also get `Node`/`Parent`, the file preamble, and the doc comment above the symbol). Comment chunks include nearby source lines. `search` defaults to 20 hits and prints a multi-line excerpt (up to 4000 bytes). Query `dump` defaults to 40 chunks and prints node metadata. Schema 2 drops stored file chunks so the next index rebuilds them.

## Consequences
- Small ADRs, Cursor rules, and most functions embed as one unit with path and neighbors.
- Existing indexes reopen as schema 2 with files cleared; `archivist index` is required before search is useful again. Commit chunks are left in place.
- Larger embeddings take longer per file. Token-dense files (lockfiles, `go.sum`) still fit the default 4000-byte cap on `qwen3-embedding:0.6b`; a higher cap overflows that model even when the advertised context is 32k.
- Heading-only splits of tiny markdown files no longer happen.
```


## `docs/global-decisions/011-last-search-meta.md` (adr/global, lines 1-18)

```md
File: docs/global-decisions/011-last-search-meta.md
Kind: adr

# Record last search in index meta

- Status: accepted
- Date: 2026-08-30
- Scope: global

## Context
`archivist status` already shows `last_indexed_at` from the `meta` table. There was no record of what was last retrieved, so it was hard to tell whether the index had been queried or with which prompt.

## Decision
After a successful semantic retrieval (`archivist search`, or `archivist dump` with a query), write `last_search` (the trimmed query) and `last_search_at` (RFC3339 UTC) on the store that was queried. `archivist status` prints both for the repo index, and the same pair for the global index when those keys are set. Empty queries are not stored.

## Consequences
- Status can show the last retrieval without parsing shell history.
- `--adr-scope global` stamps `~/.archivist/global.db`; default search stamps the repo DB.
- A failed embed does not update the keys.
- No schema bump: these are extra `meta` rows.
```

