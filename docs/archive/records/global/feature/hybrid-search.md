---
id: rec_393fc5d328e835dae38b
type: feature
scope: global
status: accepted
title: Hybrid search
applies_to: [internal/retrieve/**, internal/store/**]
tags: [search, fts, embed]
---

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
