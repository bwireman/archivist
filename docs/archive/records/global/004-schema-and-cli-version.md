---
id: rec_68b392b5b1d7fd67e7b3
type: decision
scope: global
status: accepted
title: Record schema and CLI version
---

# Record schema and CLI version

- Status: accepted
- Date: 2026-08-28
- Scope: global

## Context
The SQLite index already has a `meta` table (`last_indexed_at`). There was no record of which Archivist binary or store layout wrote it, so a later schema change would be silent.

## Decision
Keep a product `Version` (`0.1.0`, overridable at build with `-ldflags`) and an integer `Schema` (`1`). The CLI prints both via `--version` and `archivist version`. On open, the store migrates toward the current schema and writes `schema_version`. A newer database refuses to open. After a successful index, `archivist_version` is stored next to `last_indexed_at`. `archivist status` shows both.

## Consequences
- Existing indexes without `schema_version` are treated as schema 0 and migrated to 1 (current CREATE TABLE layout).
- Older CLIs error if they see a higher `schema_version`.
- Bump `Schema` and add a case in `applyMigrations` when the store contract changes; bump `Version` for releases.
- `make build VERSION=x.y.z` injects the product version; the default lives in `internal/version`.
