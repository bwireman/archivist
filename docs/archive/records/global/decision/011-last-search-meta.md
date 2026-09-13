---
id: rec_2af87c2692626de3ecd7
type: decision
scope: global
status: accepted
title: Record last search in index meta
---

## Context

`archivist status` shows `last_indexed_at`. Without a stored last query, it is hard to tell whether the index had been searched.

## Decision

After a successful search, write `last_search` (trimmed query) and `last_search_at` (RFC3339 UTC) on the repo store. Empty queries are not stored. No schema bump: these are extra `meta` rows.

## Consequences

Status currently prints last indexed time, not last search; the keys remain on the store. Search stamps even on keyword-only retrieval.
