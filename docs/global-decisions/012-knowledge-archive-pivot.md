---
id: rec_ac188553efd793db61e5
type: decision
scope: global
status: accepted
title: Pivot to knowledge archive with MCP primary surface
---

# Pivot to knowledge archive with MCP primary surface

- Status: accepted
- Date: 2026-09-12
- Scope: global

## Context

Archivist indexed code bodies, comments, and commits. The product goal is a curated archive of design decisions, rules, guides, and code structure — not source contents. Agents should query via MCP; humans and other tools read generated markdown under `docs/archive/`.

## Decision

- Typed records (`decision`, `rule`, `guide`, `map`, `pitfall`) with YAML front matter are the unit of knowledge; markdown files are source of truth.
- Schema v3 SQLite stores records, FTS5, embed queue, and a structural code map (symbols/imports, no bodies).
- MCP server (`archivist mcp`, mcp-go) is the primary query surface; CLI mirrors MCP tools.
- Embedding is asynchronous: writes enqueue; `archivist embed --worker` calls Ollama.
- Hybrid retrieval fuses FTS5 and cosine vectors; keyword-only when Ollama is down.
- `archivist export` generates `docs/archive/` for non-MCP consumers.
- Publish destinations are configured shell commands over a portable bundle.

## Consequences

- Code-body chunks and `dump` are removed; existing indexes require reindex after schema v3.
- Indexing no longer requires Ollama; embedding is optional and deferred.
- Scope is a column (`dev`, `repo`, `global`) with overlay merge at query time.
- Skills install generates host-specific agent instructions from neutral `skills/` templates.
