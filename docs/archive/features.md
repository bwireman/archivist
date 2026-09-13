# Features

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

## Embed queue and worker

- Status: accepted
- Scope: global
- Applies to: internal/embed/**, internal/store/**, internal/cmd/**, internal/mcp/**
- Tags: embed, queue, ollama

## Purpose

Defer Ollama embedding so `index`, `remember`, and `update` never need the embedder. A later worker drains SQLite queues and writes vectors for hybrid search.

## Behavior

Any `UpsertRecord` inserts or replaces a row in `embed_queue` keyed by `record_id` (`text_hash`, `enqueued_at`, `attempts`, `last_error`). Unchanged records skip upsert, so they are not re-queued. Deleting a record also deletes its queue row and vector.

The worker lists **all** queued rows on each store (not a 16-item peek) and embeds them with `--concurrency` goroutines (default 2). Each item is looked up on the store it was dequeued from, then on the other worker stores, so a home record is still embedded if the row was dequeued from the repo connection. Success is `SetRecordVector` on the store that holds the record, which upserts `record_vectors` and deletes the queue row. Opening a store deletes queue rows and vectors whose `record_id` is gone. A queue row with no matching record anywhere is dropped and logged. Ollama failure calls `FailQueueItem` and **does not** stop siblings in the same pass.

`archivist embed --worker --once` runs one full pass over the current queue and exits. Without `--once`, it repeats until the queue is empty (or a pass embeds nothing and items remain). `make embed` and the refresh skill use `--once`.

Keyword search works with a full queue. Hybrid ranking only includes records that already have vectors. `archivist status` and the MCP `status` tool report record count and queue depth as the sum of both stores.

## Connects to

- Stores: `.archivist/index.db` (repo) and `~/.archivist/archive.db` (global + dev).
- Config: `ollama.base_url`, `embed_model`, `embed_timeout`.
- Retrieval: vector half of hybrid search.
- Writes: indexer, `remember`, `update`.

## Entry points

- CLI: `archivist embed --worker [--once] [--concurrency N]`
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

User text is not FTS5 syntax. `fts5Query` splits on non-alphanumeric characters, quotes each token, and ANDs them, so paths (`docs/foo.md`) and dotted names (`records.global`) cannot produce `fts5: syntax error`. An empty token list or a leftover MATCH syntax error yields no FTS hits; vector search still uses the raw string. If Ollama is down, search is keyword-only.

After a successful search, the repo store stamps `last_search` and `last_search_at`. Status currently prints last indexed time, not last search.

## Connects to

- FTS table `records_fts` (title, body, tags) updated on upsert.
- `record_vectors` filled by the embed worker.
- Record types including `feature`, `decision`, and `rule`.

## Entry points

- CLI: `archivist search <query> [--type] [--scope] [--top]`
- MCP: `search`
- Types: `retrieve.Engine`, `store.SearchFTS`, `store.fts5Query`

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

`remember` writes a file under the configured records directory and upserts the store (FTS + embed queue). `update` rewrites the file in place. `retire` sets `status=superseded` and optional `superseded_by`. Query overlay prefers repo over global over dev for the same slug.

In this product checkout, `records.global` is `docs/global-decisions` so product records stay in git. Empty `records.global` in other repos is `~/.archivist`.

## Connects to

- Config: `records.repo`, `records.global`, `records.dev`, `records.export`.
- Indexer walks those directories (and home global/dev) and prunes missing files.
- Export writes `INDEX.md` by `record.IndexOrder`, a full-text digest per type (`rules.md`, `features.md`, …), and copies under `records/<scope>/<type>/`.
- Check only enforces `rule` records.
- On-demand skills: `record-decision`, `record-rule`, `record-feature`. Always-on `rules/record.md` tells agents when to write each type.

## Entry points

- CLI: `archivist remember`, `update`, `retire`, `check`
- MCP: `remember`, `update`, `retire`, `get`, `check`
- Types: `record.Record`, `archive.Service`

---

