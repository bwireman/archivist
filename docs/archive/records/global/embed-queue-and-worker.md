---
id: rec_423018b323939206d5cc
type: feature
scope: global
status: accepted
title: Embed queue and worker
applies_to: [internal/embed/**, internal/store/**, internal/cmd/**]
tags: [embed, queue, ollama]
---

## Purpose

Defer Ollama embedding so `index`, `remember`, and `update` never need the embedder. A later worker drains SQLite queues and writes vectors for hybrid search.

## Behavior

Any `UpsertRecord` inserts or replaces a row in `embed_queue` keyed by `record_id` (`text_hash`, `enqueued_at`, `attempts`, `last_error`). Unchanged records skip upsert, so they are not re-queued.

`DequeueEmbed` peeks `ORDER BY enqueued_at LIMIT n`; it does not claim or delete the row. Default batch is 16 per store. Success is `SetRecordVector`, which upserts `record_vectors` and deletes the queue row. Ollama failure calls `FailQueueItem` (increments `attempts`, stores `last_error`) and leaves the row; the worker then returns that error and stops.

`archivist embed --worker` requires a healthy Ollama client. `--concurrency` (default 2) runs in-process goroutines. `--once` processes one concatenated batch from repo + home stores and exits. Without `--once`, the worker peeks again after 500ms until a batch is empty. Two worker processes can embed the same record because dequeue is not a lease. `text_hash` is stored but not used to skip stale in-flight work.

Keyword search works with a full queue. Hybrid ranking only includes records that already have vectors. `archivist status` reports queue depth as the sum of both stores.

## Connects to

- Stores: `.archivist/index.db` (repo) and `~/.archivist/archive.db` (global + dev).
- Config: `ollama.base_url`, `embed_model`, `embed_timeout`.
- Retrieval: vector half of hybrid search.
- Writes: indexer, `remember`, `update`.

## Entry points

- CLI: `archivist embed --worker [--once] [--concurrency N]`
- Types: `embed.Worker`, `store.DequeueEmbed`, `store.SetRecordVector`, `store.FailQueueItem`
