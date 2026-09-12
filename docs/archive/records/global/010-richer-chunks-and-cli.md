---
id: rec_0ed07ffad59c1d58b9e2
type: decision
scope: global
status: accepted
title: Richer chunks and longer CLI results
---

# Richer chunks and longer CLI results

- Status: accepted
- Date: 2026-08-29
- Scope: global

## Context
Chunks were capped at 2000 bytes and markdown/ADRs split on every heading, so hits were often a title or a five-line section. `archivist search` flattened each hit to 200 bytes on one line and defaulted to 10 results. Agents using search and dump did not get enough surrounding code or decision text.

## Decision
Default max chunk size is 4000 bytes with 400 bytes of overlap. Docs and ADRs pack consecutive sections until that size; they only split at a heading once the current chunk is already substantial. Every file chunk is prefixed with `File` and `Kind` (tree-sitter nodes also get `Node`/`Parent`, the file preamble, and the doc comment above the symbol). Comment chunks include nearby source lines. `search` defaults to 20 hits and prints a multi-line excerpt (up to 4000 bytes). Query `dump` defaults to 40 chunks and prints node metadata. Schema 2 drops stored file chunks so the next index rebuilds them.

## Consequences
- Small ADRs, Cursor rules, and most functions embed as one unit with path and neighbors.
- Existing indexes reopen as schema 2 with files cleared; `archivist index` is required before search is useful again. Commit chunks are left in place.
- Larger embeddings take longer per file. Token-dense files (lockfiles, `go.sum`) still fit the default 4000-byte cap on `qwen3-embedding:0.6b`; a higher cap overflows that model even when the advertised context is 32k.
- Heading-only splits of tiny markdown files no longer happen.
