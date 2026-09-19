# Features

## Agent rules and skills

- Status: accepted
- Scope: global
- Applies to: rules/**, skills/**, internal/skills/**
- Tags: agents, skills

## Purpose

Generate host-specific always-on rules and on-demand skills from templates shipped in the CLI.

## Behavior

`archivist skills install --target cursor|claude|agents-md|copilot` writes consult, record, and refresh rules, and (for Cursor/Claude) the record, refresh, and publish skills. If the target checkout already has `rules/` or `skills/` with real files, those override the embedded copies.

The record rule tells agents to scan this conversation and distill lasting decisions, rules, and features without dumping chat. Skills are the per-type procedure (search first, short body). Cursor rule front matter is `alwaysApply: true` plus a one-line description (`cursorRuleDescription`). `agents-md` and Copilot get concatenated rules only.

## Connects to

- Template trees `rules/` and `skills/` (`go:embed` via `rules/fs.go`, `skills/fs.go`).
- MCP initialize instructions reuse consult + record.
- Decision: split always-on rules from on-demand skills; ship templates in the CLI.

## Entry points

- CLI: `archivist skills install --target cursor`
- Types: `skills.Install`

---

## Archive export

- Status: accepted
- Scope: global
- Applies to: internal/export/**, internal/cmd/export.go
- Tags: export

## Purpose

Generate a markdown tree under `records.export` (default `docs/archive/`) so humans and agents can read the archive without MCP.

## Behavior

`archivist export` collects records from the repo and home stores, then writes:

- `INDEX.md` — catalog grouped by `record.IndexOrder`, with links to type digest files.
- One full-text digest per type that has records: `rules.md`, `decisions.md`, `features.md`, `guides.md`, `maps.md`, `pitfalls.md`. `maps.md` is the map-type digest so it does not collide with `map.md` (the generated code map). `rules.md` is always written, even if empty, because consult instructions name that path. Empty types other than rule remove a leftover digest file.
- Individual copies at `records/<scope>/<type>/<slug>.md` (scope first, then type). The `records/` directory is replaced each export so old layout files do not linger.
- `map.md` — code structure (symbols per file).
- `archive.json` — machine-readable manifest.

Do not hand-edit `docs/archive/`; regenerate with `archivist export`. `--bundle` writes the same tree to a chosen directory. Publish uses `WriteBundle` for that tree.

## Connects to

- Config: `records.export`.
- `record.IndexOrder` for section and digest order.
- Typed records and the consult rule (`docs/archive/INDEX.md` then type digests).
- Publish destinations consume the same bundle layout.

## Entry points

- CLI: `archivist export`, `archivist export --bundle <dir>`
- Types: `export.Run`, `export.WriteBundle`

---

## Command log

- Status: accepted
- Scope: global
- Applies to: internal/cmdlog/**, internal/cmd/**, internal/mcp/**, internal/config/**

## Purpose

Opt-in audit log of archive CLI commands and MCP tool calls, written as JSONL under `.archivist/commands.log`.

## Behavior

`log_commands` defaults false. When true, each archive command or MCP tool appends two lines: `dir=in` with arguments, then `dir=out` with parsed JSON result (MCP) or ok/error (CLI) plus `duration_ms`. `init`, `version`, `skills`, and `archivist mcp` itself are skipped; MCP tool calls are still logged. Write errors are swallowed. Fields over 64KiB become `{truncated, bytes, preview}`.

Off: no file is created. On: `.archivist/` is created if needed. The path is not configurable.

## Connects to

- Config: `log_commands`.
- Path: `.archivist/commands.log` (`config.CommandsLogPath`).
- CLI wrap in `internal/cmd`; MCP `WithToolHandlerMiddleware`.

## Entry points

- Config: `.archivist.json` `log_commands`
- Types: `cmdlog.Logger`, `cmdlog.FromConfig`
- CLI: wrapped `search`, `status`, `remember`, `update`, `retire`, `check`, `index`, `embed`, `export`, `publish`, `migrate records`
- MCP: all tools when the flag is on

---

## Embed queue and worker

- Status: accepted
- Scope: global
- Applies to: internal/embed/**, internal/store/**, internal/cmd/**, internal/mcp/**
- Tags: embed, queue, ollama

## Purpose

Defer Ollama embedding so `index`, `remember`, and `update` never need the embedder. A later worker drains SQLite queues and writes vectors for hybrid search.

## Behavior

Any `UpsertRecord` inserts or replaces a row in `embed_queue` keyed by `record_id` (`text_hash`, `enqueued_at`, `attempts`, `last_error`). Unchanged records (same `content_hash` and `source_path`) skip upsert, so they are not re-queued. Deleting a record also deletes its queue row and vector. Opening a store also deletes FTS rows whose `record_id` is gone.

`UpsertRecord` adopts the id already stored at `source_path` before writing FTS and the queue, then upserts on `id`. A second remember at the same path cannot enqueue an id that is not in `records`.

The worker lists **all** queued rows on each store (not a 16-item peek) and embeds them with `--concurrency` goroutines (default 2, including when `Worker.Run` is called with concurrency ≤ 0). Each item is looked up on the store it was dequeued from, then on the other worker stores, so a home record is still embedded if the row was dequeued from the repo connection. Success is `SetRecordVector` on the store that holds the record, which upserts `record_vectors` and deletes the queue row. Opening a store deletes queue rows and vectors whose `record_id` is gone. A queue row with no matching record anywhere is dropped and logged. Ollama failure calls `FailQueueItem` and **does not** stop siblings in the same pass.

`archivist embed --once` runs one full pass over the current queue and exits. Without `--once`, it repeats until the queue is empty (or a pass embeds nothing and items remain). `make embed` and the refresh skill use `--once`. `--worker` remains as a hidden no-op for older scripts.

Keyword search works with a full queue. Hybrid ranking only includes records that already have vectors. `archivist status` and the MCP `status` tool report record count and queue depth as the sum of both stores.

## Connects to

- Stores: `.archivist/index.db` (repo) and `~/.archivist/archive.db` (global + dev).
- Config: `ollama.base_url`, `embed_model`, `embed_timeout`.
- Retrieval: vector half of hybrid search.
- Writes: indexer, `remember`, `update`.

## Entry points

- CLI: `archivist embed [--once] [--concurrency N]`
- MCP: `status`
- Types: `embed.Worker`, `store.DequeueEmbed`, `store.SetRecordVector`, `store.FailQueueItem`, `store.DropQueueItem`

---

## Hybrid search

- Status: accepted
- Scope: global
- Applies to: internal/retrieve/**, internal/store/**
- Tags: search, fts, embed

## Purpose

Find archive records by meaning and by keywords. Agents query via MCP `search` or `archivist search`; results overlay repo over global over dev for the same slug.

## Behavior

`retrieve.Engine.Search` queries FTS5 and, if an embedder is healthy, cosine similarity over `record_vectors`. Ranks are fused with RRF. Default `top_k` is 20. Optional filters: `type`, `scope`.

User text is not FTS5 syntax. `fts5Query` splits on non-alphanumeric characters, quotes each token, and ANDs them, so paths (`docs/foo.md`) and dotted names (`records.global`) cannot produce `fts5: syntax error`. An empty token list or a leftover MATCH syntax error yields no FTS hits; vector search still uses the raw string. If Ollama is down at startup, search is keyword-only. If embedding fails mid-query, search logs `search embed: ...; using keyword-only` to stderr and continues with FTS only.

Vector ranking scores embedding blobs in place (`RankEmbeddings`) and keeps only the top `top_k*3` ids per store — it does not decode every vector into a `[]float32` first. `type` and `scope` filters apply in SQL for both FTS and vector listing. Full records are hydrated in batches (`GetRecordsByIDs`) for the union of FTS hits and those top vector ids before RRF and slug overlay.

After a successful search, the repo store stamps `last_search` and `last_search_at`. Status currently prints last indexed time, not last search.

## Connects to

- FTS table `records_fts` (title, body, tags) updated on upsert.
- `record_vectors` filled by the embed worker.
- Record types including `feature`, `decision`, and `rule`.

## Entry points

- CLI: `archivist search <query> [--type] [--scope] [--top]`
- MCP: `search`
- Types: `retrieve.Engine`, `store.SearchFTS`, `store.RankEmbeddings`, `store.GetRecordsByIDs`, `store.fts5Query`

---

## MCP server

- Status: accepted
- Scope: global
- Applies to: internal/mcp/**, internal/cmd/mcp.go
- Tags: mcp, agents

## Purpose

Primary agent API over stdio. Clients consult and curate the archive without a separate CLI round-trip.

## Behavior

`archivist mcp` serves `search`, `get`, `check`, `map`, `remember`, `update`, `retire`, and `status` over JSON-RPC stdio. Initialize `instructions` are `rules/consult.md` plus `rules/record.md` so hosts without `skills install` still look things up and distill lasting facts from conversation. Tool descriptions tell agents to search before writing, prefer `update` on an existing record, and skip chat transcripts.

Search works if Ollama is down (keyword-only). Writes never need Ollama. The process cwd must be the repo, or the client must pass `--path`.

`remember` splits `applies_to` and `tags` on commas or newlines (same as CLI `--applies-to` / `--tags`). `check` splits `paths` the same way. `check` `HasViolation` follows the rule-check feature (glob matches only).

When `log_commands` is true, each tool call is appended to `.archivist/commands.log` (`dir=in` then `dir=out`) via tool-handler middleware. The `mcp` process itself is not logged as a CLI command. Stdio JSON-RPC is never written to the log.

## Connects to

- `retrieve.Engine`, `archive.Service`, `check`, embed health/`status`.
- Embedded rule templates in `rules/` (`consult.md`, `record.md`).
- On-demand skills for the per-type write procedure.
- Command log: `log_commands`, `cmdlog.Logger`.

## Entry points

- CLI: `archivist mcp`
- Types: `mcp.Server`, `mcp.ServeStdio`

---

## Rule check

- Status: accepted
- Scope: global
- Applies to: internal/check/**, internal/cmd/remember.go, internal/mcp/**
- Tags: check, rules

## Purpose

Match archive rules against a proposed change so agents and CI can see which constraints apply.

## Behavior

`check.Run` loads every `rule` record from the repo and home stores. Rules whose `applies_to` globs match the supplied paths are listed first. An optional description runs hybrid search over rules; those hits are advisory. Diff text adds paths from `diff --git`, `---` and `+++` lines. `/dev/null` is ignored; a rename contributes both old and new paths. MCP `paths` may be a comma- or newline-separated list.

`HasViolation` is true only when a must/must-not rule matched via `applies_to`. Semantic-only hits never set it. CLI `--strict` exits nonzero on `HasViolation`.

## Connects to

- `retrieve.Engine` for semantic rule search.
- `record.MatchesPaths` / `IsEnforceable`.
- Typed records and the record-rule skill.

## Entry points

- CLI: `archivist check [description] [--paths] [--diff] [--strict]`
- MCP: `check`
- Types: `check.Run`, `check.Result`

---

## Typed archive records

- Status: accepted
- Scope: global
- Applies to: internal/record/**, internal/archive/**
- Tags: records

## Purpose

Markdown files with YAML front matter are the source of truth for the archive. SQLite indexes them for search; export mirrors them under `docs/archive/`.

## Behavior

Types: `decision`, `rule`, `feature`, `guide`, `map`, `pitfall`. Scopes: `dev`, `repo`, `global`. Status: `proposed`, `accepted`, `deprecated`, `superseded`. Rules may set `severity` and `applies_to` globs for `archivist check`. Features should set `applies_to` to the packages they document.

`remember` writes a file under the configured records directory and upserts the store (FTS + embed queue). If that default path already exists (on disk or in the index), `remember` reuses the existing record id and overwrites the file — same identity as `update`. Writes are atomic (temp file + rename) then `UpsertRecord`; `archivist index` repairs the store if the file lands and the upsert fails. `update` rewrites the file in place. `retire` sets `status=superseded` and optional `superseded_by`. Query overlay prefers repo over global over dev for the same slug.

`UpsertRecord` keys on record id after adopting any row already stored at `source_path`, so a second write at the same path cannot leave FTS or `embed_queue` pointing at a new id that is not in `records`.

In this product checkout, `records.global` is `docs/global-decisions` so product records stay in git. Empty `records.global` in other repos is `~/.archivist`.

## Connects to

- Config: `records.repo`, `records.global`, `records.dev`, `records.export`.
- Indexer walks those directories (and home global/dev) and prunes missing files.
- Export writes `INDEX.md` by `record.IndexOrder`, a full-text digest per type (`rules.md`, `features.md`, …), and copies under `records/<scope>/<type>/`.
- Check only enforces `rule` records (`applies_to` glob matches); semantic hits are advisory.
- On-demand skills: `record-decision`, `record-rule`, `record-feature`. Always-on `rules/record.md` tells agents to distill lasting facts from this conversation (search first, skip chat glut). MCP initialize `instructions` are the consult + record templates so hosts without skills install still get that bar.

## Entry points

- CLI: `archivist remember`, `update`, `retire`, `check`
- MCP: `remember`, `update`, `retire`, `get`, `check`
- Types: `record.Record`, `archive.Service`, `store.UpsertRecord`

---

