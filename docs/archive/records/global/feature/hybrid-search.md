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
