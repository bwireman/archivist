---
id: rec_68b392b5b1d7fd67e7b3
type: decision
scope: global
status: accepted
title: Record schema and CLI version
---

## Context

The SQLite index has a `meta` table. Without a schema and product version, a later store change would be silent.

## Decision

Keep a product `Version` (`0.1.0`, overridable at build with `-ldflags`) and an integer `Schema` (currently `3`). The CLI prints both via `--version` and `archivist version`. On open, the store migrates toward the current schema and writes `schema_version`. A newer database refuses to open. After a successful index, `archivist_version` is stored next to `last_indexed_at`.

## Consequences

Bump `Schema` and add a case in `applyMigrations` when the store contract changes; bump `Version` for releases. `make build VERSION=x.y.z` injects the product version; the default lives in `internal/version`.
