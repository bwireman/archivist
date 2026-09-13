---
id: rec_cd668042d22370100627
type: decision
scope: global
status: accepted
title: --once drains the whole embed queue
applies_to: [internal/embed/**, internal/store/**]
tags: [embed, queue, worker]
---

## Context

`archivist embed --worker --once` (used by `make embed` and the refresh skill) peeked `LIMIT 16` per store and returned on the first Ollama error. A missing record counted as success without deleting its queue row. Any home or repo queue larger than 16 never emptied, so the queue looked stuck.

Alternatives: keep `--once` as a single batch and tell callers to loop; raise the batch size; make `--once` drain until empty (which would retry failed rows forever).

## Decision

`--once` snapshots the entire queue on each store and processes every item once. A sibling failure does not abort the rest of the pass. Queue rows whose records are gone are dropped. Without `--once`, the worker repeats until the queue is empty or a pass makes no progress.

## Consequences

A healthy Ollama run with `--once` leaves an empty queue. Failed items stay for a later retry. CLI and MCP `status` count records and queue depth across repo and home stores.
