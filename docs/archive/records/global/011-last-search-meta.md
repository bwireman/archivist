---
id: rec_2af87c2692626de3ecd7
type: decision
scope: global
status: accepted
title: Record last search in index meta
---

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
