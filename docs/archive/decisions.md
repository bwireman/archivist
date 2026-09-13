# Decisions

## --once drains the whole embed queue

- Status: accepted
- Scope: global
- Applies to: internal/embed/**, internal/store/**
- Tags: embed, queue, worker

## Context

`archivist embed --worker --once` (used by `make embed` and the refresh skill) peeked `LIMIT 16` per store and returned on the first Ollama error. A missing record counted as success without deleting its queue row. Any home or repo queue larger than 16 never emptied, so the queue looked stuck.

Alternatives: keep `--once` as a single batch and tell callers to loop; raise the batch size; make `--once` drain until empty (which would retry failed rows forever).

## Decision

`--once` snapshots the entire queue on each store and processes every item once. A sibling failure does not abort the rest of the pass. Queue rows whose records are gone are dropped. Without `--once`, the worker repeats until the queue is empty or a pass makes no progress.

## Consequences

A healthy Ollama run with `--once` leaves an empty queue. Failed items stay for a later retry. CLI and MCP `status` count records and queue depth across repo and home stores.

---

## Add feature as a record type for living capability docs

- Status: accepted
- Scope: global
- Tags: records

## Context

Typed records were `decision`, `rule`, `guide`, `map`, and `pitfall`. Decisions capture a choice; rules capture constraints; `guide` was unused. There was no type for how a capability works, which packages and stores it uses, or how to invoke it. That knowledge lived in chat or was stuffed into decisions.

Alternatives: overload `guide`; put capability notes only in `docs/archive/map.md`; add a `feature` type.

## Decision

Add `feature` as a first-class record type. A feature document is living: Purpose, Behavior, Connects to, and Entry points. `applies_to` globs point at the implementing packages. Export lists features after decisions. `guide` remains available for how-to procedures; `map` remains the generated code structure tree.

When behavior changes, update the feature. When the capability is removed, retire it. Do not encode a design choice as a feature or a subsystem description as a decision.

## Consequences

- `archivist remember --type feature` and MCP `remember` accept the type without a schema bump (type is a string column).
- Agents should search `--type feature` before changing a subsystem.
- Skills include `record-feature`; the always-on record rule mentions features next to decisions and rules.

---

## Archive hygiene: current facts, stubs for superseded, delete non-records

- Status: accepted
- Scope: global
- Tags: hygiene

## Context

The archive still held pre-pivot indexer ADRs (dump, ADR globs, TUIs, code-body chunks), session "gap" notes marked accepted, and empty Untitled export leftovers. Agents treat this archive as the source of truth, so stale accepted records are worse than missing history.

## Decision

- An accepted record must match current code. If the choice still holds, update the body in place. If a later record replaced it, retire it (`superseded` + `superseded_by`) and leave a short stub, not the obsolete spec.
- Delete records that are not decisions: empty/untitled files, one-off session notes, and "gap catalog" stubs.
- Do not duplicate a typed `rule` inside a `decision` when the rule already exists.
- Prefer one current document per topic (config, retrieval, code map) over a chain of overlapping accepted ADRs.

## Consequences

- Search and `docs/archive/INDEX.md` stay small enough to consult.
- Historical TUI/glob/chunking choices remain findable as superseded stubs.
- New session friction belongs in chat or a confirmed pitfall, not as an accepted decision.

---

## Default ADR globs are the product directories

- Status: superseded
- Scope: global

Superseded by rec_config014. Default `records.repo` is `docs/decisions`. Empty `records.global` is `~/.archivist`; this product sets `docs/global-decisions` explicitly. Extra `**/adr/**` globs were dropped.

---

## Default embed model is qwen3-embedding:0.6b

- Status: accepted
- Scope: global

## Context

`archivist init` used to write `nomic-embed-text`. New repos should match the model this checkout already uses.

## Decision

`config.Default()` and `archivist init` use `qwen3-embedding:0.6b`. A repo may override `ollama.embed_model` in `.archivist.json`.

## Consequences

`ollama pull qwen3-embedding:0.6b` is the documented setup step. Changing the embed model later requires re-embedding; existing vectors are not comparable across models.

---

## Default records.global is ~/.archivist

- Status: accepted
- Scope: global
- Tags: config, records

## Context

`records.global` defaulted to in-repo `docs/global-decisions`. Global records were already stored in `~/.archivist/archive.db`, but the markdown lived in every checkout. `records.dev` already defaulted to `~/.archivist/records`.

## Decision

- Empty `records.global` means `~/.archivist`. Absolute and `~/…` values are also outside the checkout.
- `archivist init` creates `~/.archivist` for global docs and does not create `docs/global-decisions`.
- Scope `global` writes `~/.archivist/<slug>.md` (virtual path `global/…`). The indexer walks that directory for `.md` files and skips `records.dev` (`~/.archivist/records/`).
- A checkout-relative `records.global` (this repository uses `docs/global-decisions`) keeps product-wide records in git.

## Consequences

- New repos share machine-wide global docs without copying `docs/global-decisions`.
- This product archive stays in-repo because `.archivist.json` sets `records.global` explicitly.
- Markdown files at the top of `~/.archivist` are records; `archive.db` and `records/` stay out of that walk.

---

## Drop interactive TUIs; CLI is always plain

- Status: accepted
- Scope: global

## Context

`archivist index` showed a Bubble Tea progress view on a TTY, and `archivist init` opened a Huh form. Agents and scripts already passed `--plain`. Charm libraries were a large dependency surface for a product whose primary interface is MCP plus one-line CLI.

## Decision

Remove the TUI package and Charm dependencies. `init` always writes `config.Default()`. `index` always prints a one-line summary. `--plain` remains as a hidden no-op so existing scripts keep working.

## Consequences

- No Bubble Tea, Huh, Lip Gloss, or go-isatty in the CLI.
- Interactive setup is editing `.archivist.json`.
- `make index` and leftover `--plain` flags still work.

---

## Export archive grouped by record type with type digest files

- Status: accepted
- Scope: global
- Tags: export, records

## Context

Export copied every record under `records/<scope>/` regardless of type. `INDEX.md` already grouped titles by type, and only rules had a compiled digest (`rules.md`). Agents reading the generated tree mixed features, decisions, and rules in one directory and had to open many files to learn one type.

Alternatives: keep mixed copies and only add digest files; nest copies as `records/<type>/<scope>/`; nest copies as `records/<scope>/<type>/`; split source record directories by type.

## Decision

`archivist export` groups generated copies as `records/<scope>/<type>/<slug>.md` (scope first, then type) and writes a full-text digest file per type that has records (`rules.md`, `decisions.md`, `features.md`, `guides.md`, `maps.md`, `pitfalls.md`). `rules.md` is always written so consult instructions can name that path. `map.md` stays the code map; map-type records go to `maps.md`. Source record directories stay organized by scope; type lives in front matter.

The `records/` export directory is replaced each run so the old `records/<scope>/` and `records/<type>/<scope>/` layouts do not linger.

## Consequences

- Bots can read one digest per type or glob `docs/archive/records/<scope>/<type>/`.
- Scope stays the primary folder, matching source dirs and overlay (repo / global / dev).
- `INDEX.md` links include the type segment and point at type digests.
- The consult rule names type digest files next to `INDEX.md`.
- Publish bundles pick up the same layout.

---

## Honor .gitignore when indexing

- Status: accepted
- Scope: global

## Context

`skip_dirs` duplicated `.gitignore`. Build artifacts were easy to index by accident.

## Decision

The indexer always honors `.gitignore` (root and nested). `.git` and `.archivist` are always skipped. Extra excludes go in `index.skip_globs`. There is no `honor_gitignore` flag. Dev records under `~/.archivist/records/` are not filtered by a checkout's gitignore.

## Consequences

Gitignored files already in the index are pruned on the next index.

---

## Language-agnostic code map backup with fixtures per language

- Status: accepted
- Scope: global
- Tags: codemap

## Context

The generic fallback only matched `func `, `def `, `class `, and `export ` and stored the whole line. Unsupported languages produced empty maps when tree-sitter failed or was absent.

## Decision

- Tree-sitter (Go, Python, JS/JSX/MJS/CJS, TS/TSX/MTS/CTS, Rust, Java) and the Gleam line extractor remain primary.
- Backup is language-agnostic: a declaration keyword plus an identifier, with a short modifier list. Imports use `import`/`use`/`require`/`include`/`from`. Store the identifier, not the whole line.
- Skip docs and data extensions. Skip English stopword names.
- If a primary extractor errors or returns no symbols and no imports, run the backup and keep any package name already found.
- `codemap.Version` is 2. Adding a language also requires a fixture (see the fixture rule).

## Consequences

Keyword languages without a grammar appear in the map. Docs, configs, and lockfiles stay off it. Tree-sitter still wins when it finds symbols.

---

## Line-based Gleam map and remapping on extractor version

- Status: accepted
- Scope: global
- Tags: codemap, gleam

## Context

Tree-sitter covered Go, Python, JS, TS, Rust, and Java. Gleam (`pub fn`, `fn`, `pub type`, `const`) produced no symbols, and `go-tree-sitter` has no Gleam grammar. Reindexing after an extractor fix skipped files whose content hashes had not changed.

## Decision

- Use a line-based Gleam extractor (`.gleam`) for top-level `import`, `fn`, `type` (including constructors), and `const`. Do not add tree-sitter-gleam.
- Store `codemap_version` in index meta. Bump `codemap.Version` when extractors change so a full index remaps unchanged files. Do not stamp the version after a scoped index.

## Consequences

Gleam sources appear in `map.md` after a full index. Existing indexes remap once without deleting `.archivist/index.db`. The generic fallback for other languages is a later decision (language-agnostic backup).

---

## Nest ADR globs under index.adr

- Status: superseded
- Scope: global

Superseded by rec_config014. ADR globs (`index.adr`) were removed; record locations are `records.repo` / `records.global` directories.

---

## Pivot to knowledge archive with MCP primary surface

- Status: accepted
- Scope: global

## Context

Archivist indexed code bodies, comments, and commits. The product goal is a curated archive of design decisions, rules, features, guides, and code structure — not source contents. Agents should query via MCP; humans and other tools read generated markdown under `docs/archive/`.

## Decision

- Typed records (`decision`, `rule`, `feature`, `guide`, `map`, `pitfall`) with YAML front matter are the unit of knowledge; markdown files are source of truth.
- Schema v3 SQLite stores records, FTS5, embed queue, and a structural code map (symbols/imports, no bodies).
- MCP server (`archivist mcp`, mcp-go) is the primary query surface; CLI mirrors MCP tools.
- Embedding is asynchronous: writes enqueue; `archivist embed --worker` calls Ollama.
- Hybrid retrieval fuses FTS5 and cosine vectors (default 20 hits); keyword-only when Ollama is down.
- `archivist export` generates `docs/archive/` for non-MCP consumers.
- Publish destinations are configured shell commands over a portable bundle.

## Consequences

- Code-body chunks and `dump` are removed; existing indexes require reindex after schema v3.
- Indexing no longer requires Ollama; embedding is optional and deferred.
- Scope is a column (`dev`, `repo`, `global`) with overlay merge at query time.
- Skills install generates host-specific agent instructions from neutral `skills/` templates.

---

## Record last search in index meta

- Status: accepted
- Scope: global

## Context

`archivist status` shows `last_indexed_at`. Without a stored last query, it is hard to tell whether the index had been searched.

## Decision

After a successful search, write `last_search` (trimmed query) and `last_search_at` (RFC3339 UTC) on the repo store. Empty queries are not stored. No schema bump: these are extra `meta` rows.

## Consequences

Status currently prints last indexed time, not last search; the keys remain on the store. Search stamps even on keyword-only retrieval.

---

## Record schema and CLI version

- Status: accepted
- Scope: global

## Context

The SQLite index has a `meta` table. Without a schema and product version, a later store change would be silent.

## Decision

Keep a product `Version` (`0.1.0`, overridable at build with `-ldflags`) and an integer `Schema` (currently `3`). The CLI prints both via `--version` and `archivist version`. On open, the store migrates toward the current schema and writes `schema_version`. A newer database refuses to open. After a successful index, `archivist_version` is stored next to `last_indexed_at`.

## Consequences

Bump `Schema` and add a case in `applyMigrations` when the store contract changes; bump `Version` for releases. `make build VERSION=x.y.z` injects the product version; the default lives in `internal/version`.

---

## Reshape config around records directories

- Status: accepted
- Scope: global

## Context

`.archivist.json` still described the old code indexer: ADR globs, skip_dirs overlapping gitignore, and configurable SQLite paths. Record locations were hardcoded. This checkout's `store.global_path` still pointed at `global.db` while the default was `archive.db`.

## Decision

- `records.repo` / `records.global` / `records.dev` / `records.export` are directories.
- Empty `dev` means `~/.archivist/records`. Empty `global` means `~/.archivist` (not `docs/global-decisions`).
- This product sets `records.global` to `docs/global-decisions` so product-wide records stay in git.
- Always skip `.git` and `.archivist`. Always honor `.gitignore`. Extra excludes go in `index.skip_globs`.
- SQLite is always `.archivist/index.db` and `~/.archivist/archive.db`.
- Keep `ollama.base_url`, `embed_model`, `embed_timeout` (default `5m`). Empty URL uses `$OLLAMA_HOST`.
- Keep `publish.destinations`. Drop `adr_paths` / `index.adr` / `store.*` / `honor_gitignore` / `skip_dirs`.

## Consequences

- Existing configs that only set Ollama still load.
- Configs that relied on `store.global_path` or `index.adr` need a one-line edit.
- `vendor/` and `node_modules/` are skipped only if gitignored or listed in `skip_globs`.

---

## Richer chunks and longer CLI results

- Status: superseded
- Scope: global

Superseded by rec_ac188553efd793db61e5. Code-body chunks and `dump` were removed. Search is over typed records (default 20 hits).

---

## Separate SQLite index for global ADRs

- Status: superseded
- Scope: global

Superseded by rec_config014. Two SQLite files remain, but paths and roles are fixed: `.archivist/index.db` (checkout) and `~/.archivist/archive.db` (global + dev). `global.db`, `store.global_path`, and dump are gone.

---

## Separate global and repo ADRs

- Status: superseded
- Scope: global

Superseded by rec_ac188553efd793db61e5 (knowledge-archive pivot). Typed records use a `scope` column (`dev`, `repo`, `global`) instead of ADR path globs, `--adr-scope`, and `dump`.

---

## Ship rule and skill templates in the CLI

- Status: accepted
- Scope: global
- Tags: skills, embed

## Context

`archivist skills install` read `rules/` and `skills/` from the target repository. That works in this checkout, but a consumer repo has no template tree, so install failed.

Alternatives: copy templates into every repo; require running install from this source tree; look up files next to the executable; embed the templates in the binary.

## Decision

Embed `rules/*.md` and `skills/*/SKILL.md` in the CLI. `skills install` uses the shipped templates. If the target checkout already has `rules/` or `skills/` with real files, those override the embedded copies so this repo can iterate without a rebuild.

## Consequences

- `go:embed` lives next to the files (`rules/fs.go`, `skills/fs.go`) because embed cannot reach outside its package.
- Consumer repos get Cursor rules and skills from `archivist skills install --target cursor` after `go install`.
- Editing templates in this repo still requires `make install` (or a local `rules/`/`skills/` tree) before host files pick up the change.

---

## Split always-on rules from on-demand skills

- Status: accepted
- Scope: global

## Context

`archivist skills install --target cursor` copied `skills/*/SKILL.md` into `.cursor/rules/` with `alwaysApply: false`. Those files were short “always do this” lists, so they behaved like rules, while Cursor skills actually live under `.cursor/skills/`. This repo also had hand-written always-on rules that duplicated the generated files.

## Decision

Keep two template trees:

- `rules/` — always-on constraints (consult, record, refresh).
- `skills/` — on-demand procedures with SKILL.md front matter (`record-decision`, `record-rule`, `record-feature`, `refresh-archive`, `publish-archive`).

Install maps them to the host: Cursor rules + `.cursor/skills/`; Claude skills + `CLAUDE.md`; `agents-md` and Copilot get concatenated rules only.

## Consequences

- Consult is a rule, not a skill.
- `skills install` is the generator; do not hand-edit the host files it writes.

---

## Treat FTS MATCH input as natural language

- Status: accepted
- Scope: global
- Tags: search, fts

## Context

User and agent queries include paths (`docs/foo.md`), config keys (`records.global`), and model names (`qwen3-embedding:0.6b`). Passing them unchanged to FTS5 MATCH treats `/`, `.`, `:`, quotes, and `AND`/`OR`/`NOT` as syntax, so `search` and `check` fail with `fts5: syntax error`.

Alternatives: document FTS5 syntax and require callers to escape; wrap the whole query as one phrase; split into letter/number tokens and quote each as a term.

## Decision

Treat MATCH input as natural language, not FTS5 syntax. Split on non-alphanumeric characters, quote each token (so operators in the input are terms), and combine with implicit AND. If a query has no tokens or MATCH still reports a syntax error, return no FTS hits instead of failing hybrid search. Vector search still uses the raw string.

## Consequences

- Paths and dotted names search as tokens and no longer crash the CLI or MCP.
- Callers cannot use FTS5 operators or prefix `*` as syntax; punctuation is a separator.
- `check` descriptions that happen to be file paths become token queries instead of errors.

---

## Use Bubble Tea for the index TUI

- Status: superseded
- Scope: global

Superseded by rec_plaincli013. `archivist index` no longer shows a Bubble Tea TUI; the CLI always prints a one-line summary.

---

## Use Huh for the init TUI

- Status: superseded
- Scope: global

Superseded by rec_plaincli013. `archivist init` always writes `config.Default()` with no Huh form.

---

## Use qwen3-embedding:0.6b for this repository

- Status: superseded
- Scope: repo

Superseded by rec_330b72a24aff842edf5a. The tool default is `qwen3-embedding:0.6b`; this checkout does not need a special pin.

---

