# Archivist context dump

Retrieved chunks from a local code index. Use only facts present here.

- Query: (all matching chunks)
- ADR scope: global
- Type: adr
- Chunks: 32

## `docs/global-decisions/001-bubbletea-index-tui.md` (adr/global, lines 1-6)

```md
# Use Bubble Tea for the index TUI

- Status: accepted
- Date: 2026-08-28
- Scope: global
```


## `docs/global-decisions/001-bubbletea-index-tui.md` (adr/global, lines 7-9)

```md
## Context
`archivist index` printed one line when it finished. Embedding is per-chunk and can take a while, so a live view of the current file and counts is useful. `make index` and piped runs still need a log-friendly summary.
```


## `docs/global-decisions/001-bubbletea-index-tui.md` (adr/global, lines 10-12)

```md
## Decision
Show a Bubble Tea TUI when stdout is a TTY. Fall back to a one-line summary when it is not, or when `--plain` is set (`make index` uses `--plain`). The indexer reports progress through a callback so tests do not need a terminal.
```


## `docs/global-decisions/001-bubbletea-index-tui.md` (adr/global, lines 13-17)

```md
## Consequences
- Interactive runs of `archivist index` show scan/file/git progress and can cancel with q.
- `make index` passes `--plain` so agent and CI logs stay a single summary line.
- Charm libraries (bubbletea, bubbles, lipgloss) are dependencies of the CLI.
```


## `docs/global-decisions/002-huh-init-tui.md` (adr/global, lines 1-6)

```md
# Use Huh for the init TUI

- Status: accepted
- Date: 2026-08-28
- Scope: global
```


## `docs/global-decisions/002-huh-init-tui.md` (adr/global, lines 7-9)

```md
## Context
`archivist init` wrote `config.Default()` with no prompts. The embed model is a real choice (`nomic-embed-text` vs a locally installed model). The index command already uses a Charm TUI on a TTY.
```


## `docs/global-decisions/002-huh-init-tui.md` (adr/global, lines 10-12)

```md
## Decision
Show a Huh form on a TTY so `archivist init` can edit Ollama, index, and store settings before writing `.archivist.json`. `--plain` writes `config.Default()` with no prompt.
```


## `docs/global-decisions/002-huh-init-tui.md` (adr/global, lines 13-17)

```md
## Consequences
- Interactive setup can set the embed model without editing JSON.
- Scripts and `--plain` still get the tool defaults.
- Huh is an additional Charm dependency.
```


## `docs/global-decisions/003-global-and-repo-adrs.md` (adr/global, lines 1-6)

```md
# Separate global and repo ADRs

- Status: accepted
- Date: 2026-08-28
- Scope: global
```


## `docs/global-decisions/003-global-and-repo-adrs.md` (adr/global, lines 7-11)

```md
## Context
Product decisions (index TUI, init TUI) were stored next to this checkout's repo-specific choices (embed model). Agents need to tell tool-wide decisions from project-local ones, and a user may keep ADRs that apply to every repo.

`--scope` already means a path prefix or glob. Reusing it for ADR kind would collide with dump and index flags.
```


## `docs/global-decisions/003-global-and-repo-adrs.md` (adr/global, lines 12-14)

```md
## Decision
Classify ADRs as `repo` or `global`. Repo ADRs live in `docs/decisions/`. In-repo global ADRs live in `docs/global-decisions/` (a sibling directory, so `docs/decisions/**` does not match them). User-global ADRs live in `~/.archivist/decisions/`. Filter with `--adr-scope repo|global`. `make dump-docs` writes `docs/dump/decisions.md` and `docs/dump/global-decisions.md`. Storage is two SQLite files; see `006-separate-global-index.md`.
```


## `docs/global-decisions/003-global-and-repo-adrs.md` (adr/global, lines 15-20)

```md
## Consequences
- Search and dump can list only repo or only global ADRs.
- Path globs are the source of truth for classification; a `- Scope:` line in the markdown is for humans.
- Existing ADR chunks without `adr_scope` metadata are treated as repo.
- `archivist init` creates the two in-repo directories. It does not create `~/.archivist/decisions/`.
```


## `docs/global-decisions/004-schema-and-cli-version.md` (adr/global, lines 1-6)

```md
# Record schema and CLI version

- Status: accepted
- Date: 2026-08-28
- Scope: global
```


## `docs/global-decisions/004-schema-and-cli-version.md` (adr/global, lines 7-9)

```md
## Context
The SQLite index already has a `meta` table (`last_indexed_at`). There was no record of which Archivist binary or store layout wrote it, so a later schema change would be silent.
```


## `docs/global-decisions/004-schema-and-cli-version.md` (adr/global, lines 10-12)

```md
## Decision
Keep a product `Version` (`0.1.0`, overridable at build with `-ldflags`) and an integer `Schema` (`1`). The CLI prints both via `--version` and `archivist version`. On open, the store migrates toward the current schema and writes `schema_version`. A newer database refuses to open. After a successful index, `archivist_version` is stored next to `last_indexed_at`. `archivist status` shows both.
```


## `docs/global-decisions/004-schema-and-cli-version.md` (adr/global, lines 13-18)

```md
## Consequences
- Existing indexes without `schema_version` are treated as schema 0 and migrated to 1 (current CREATE TABLE layout).
- Older CLIs error if they see a higher `schema_version`.
- Bump `Schema` and add a case in `applyMigrations` when the store contract changes; bump `Version` for releases.
- `make build VERSION=x.y.z` injects the product version; the default lives in `internal/version`.
```


## `docs/global-decisions/005-nested-adr-config.md` (adr/global, lines 1-6)

```md
# Nest ADR globs under index.adr

- Status: accepted
- Date: 2026-08-28
- Scope: global
```


## `docs/global-decisions/005-nested-adr-config.md` (adr/global, lines 7-9)

```md
## Context
Repo vs global ADRs were configured as sibling lists `adr_paths` and `global_adr_paths`. The names did not say what the lists were for, or how they related.
```


## `docs/global-decisions/005-nested-adr-config.md` (adr/global, lines 10-12)

```md
## Decision
Put both under `index.adr`, with `repo` and `global` glob lists. Global wins if a path matches both. `~/.archivist/decisions/` stays implicit (not a repo glob) and is indexed as `global/<file>`. Old `adr_paths` / `global_adr_paths` keys still load.
```


## `docs/global-decisions/005-nested-adr-config.md` (adr/global, lines 13-17)

```md
## Consequences
- `.archivist.json` reads as "these paths are repo ADRs, these are global ADRs".
- Save writes the nested shape only.
- Empty lists disable that kind; omitting `adr` keeps the defaults.
```


## `docs/global-decisions/006-separate-global-index.md` (adr/global, lines 1-6)

```md
# Separate SQLite index for global ADRs

- Status: accepted
- Date: 2026-08-28
- Scope: global
```


## `docs/global-decisions/006-separate-global-index.md` (adr/global, lines 7-9)

```md
## Context
In-repo `docs/global-decisions/` and `~/.archivist/decisions/` were stored in each checkout's `.archivist/index.db`. Default search mixed product decisions into project context, and one repo could not consult another checkout's global ADRs without also ingesting that repo's code.
```


## `docs/global-decisions/006-separate-global-index.md` (adr/global, lines 10-12)

```md
## Decision
Keep two SQLite files. The repo index (`.archivist/index.db`) holds this checkout only: code, docs, git, and repo ADRs. The machine-wide index (`~/.archivist/global.db` by default) holds user ADRs plus files that match `index.adr.global`. `store.global_path` is optional; a relative value resolves under `~/.archivist/`, not the repo. Default search and dump use the repo DB. `--adr-scope global` (and `make dump-docs`'s global half) use the global DB. User ADRs are stored as `user/<file>`. In-repo global chunks stamp `origin=repo` and `origin_root=<abs repo>` so prune only removes this checkout's missing files. Git history is never written to the global DB.
```


## `docs/global-decisions/006-separate-global-index.md` (adr/global, lines 13-20)

```md
## Consequences
- Agents can search global ADRs from any repo after that machine has indexed them, without pulling in another project's code.
- Default repo search no longer returns global ADRs.
- The same relative path from two products last-writer-wins in `global.db`.
- Global queries use the current embed model; vectors written with a different model will not rank well until reindexed.
- Untagged in-repo global chunks (no `origin_root`) are left in place until overwritten.
- Scoped `--scope` index does not prune in-repo globals outside that prefix.
```


## `docs/global-decisions/007-honor-gitignore.md` (adr/global, lines 1-6)

```md
# Honor .gitignore when indexing

- Status: accepted
- Date: 2026-08-28
- Scope: global
```


## `docs/global-decisions/007-honor-gitignore.md` (adr/global, lines 7-9)

```md
## Context
`skip_dirs` and `skip_globs` were a second, incomplete copy of what most repos already list in `.gitignore`. Build artifacts, local binaries, and editor dirs were easy to index by accident.
```


## `docs/global-decisions/007-honor-gitignore.md` (adr/global, lines 10-12)

```md
## Decision
`index.honor_gitignore` defaults to true. When set, the indexer skips files and directories that match the repo `.gitignore` (root and nested), including directory patterns so ignored trees are not walked. `skip_dirs` and `skip_globs` still apply. User-global ADRs under `~/.archivist/decisions/` are not filtered by a checkout's gitignore. Set the flag false to index gitignored paths.
```


## `docs/global-decisions/007-honor-gitignore.md` (adr/global, lines 13-17)

```md
## Consequences
- `archivist init` writes `honor_gitignore: true` and the setup TUI can turn it off.
- Omitting the key in an existing `.archivist.json` still means true.
- Gitignored files that were already in the index are pruned on the next index.
```


## `docs/global-decisions/008-default-adr-globs.md` (adr/global, lines 1-6)

```md
# Default ADR globs are the product directories

- Status: accepted
- Date: 2026-08-28
- Scope: global
```


## `docs/global-decisions/008-default-adr-globs.md` (adr/global, lines 7-9)

```md
## Context
Default `index.adr.repo` also matched `**/adr/**` and `**/ADR*.md` so other layouts would classify as ADRs. This product only uses `docs/decisions/` and `docs/global-decisions/`, which is what the Cursor rules and `archivist init` already create. Extra globs made the config look unrelated to those directories.
```


## `docs/global-decisions/008-default-adr-globs.md` (adr/global, lines 10-12)

```md
## Decision
Default `index.adr.repo` is `docs/decisions/**`. Default `index.adr.global` is `docs/global-decisions/**`. Other layouts stay possible as an explicit override. Empty lists still disable that kind; omitting `adr` still keeps these defaults.
```


## `docs/global-decisions/008-default-adr-globs.md` (adr/global, lines 13-17)

```md
## Consequences
- `.archivist.json` `index.adr` matches the directories agents write to.
- Files like `docs/ADR-001.md` are ordinary docs unless the repo adds a glob.
- `~/.archivist/decisions/` is still implicit, not a glob.
```

