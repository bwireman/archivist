---
id: rec_423018b323939206d5cc
type: feature
scope: global
status: accepted
title: Embed queue and worker
applies_to: [internal/embed/**, internal/store/**, internal/cmd/**, internal/mcp/**]
tags: [embed, queue, ollama]
---

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
