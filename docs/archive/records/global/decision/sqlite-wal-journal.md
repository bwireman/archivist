---
id: rec_sqlite_wal_journal
type: decision
scope: global
status: accepted
title: Use WAL journal mode for SQLite stores
---

## Context

Archivist opens the same SQLite files from multiple processes: MCP search while `index` or `embed` writes. Default rollback journal mode serializes readers behind writers.

## Decision

Open repo and home stores with `PRAGMA journal_mode=WAL`. Keep `busy_timeout` and `foreign_keys`. No schema bump; `-wal` and `-shm` files live under `.archivist/` and `~/.archivist/`.

## Consequences

- Concurrent MCP search and CLI index/embed are less likely to block on the same database file.
- WAL is standard for local SQLite with concurrent readers; rollback mode remains the SQLite default for other tools that open the files without WAL.
