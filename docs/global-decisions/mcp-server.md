---
id: rec_00d3d3d48b988c969105
type: feature
scope: global
status: accepted
title: MCP server
applies_to: [internal/mcp/**, internal/cmd/mcp.go]
tags: [mcp, agents]
---

## Purpose

Primary agent API over stdio. Clients consult and curate the archive without a separate CLI round-trip.

## Behavior

`archivist mcp` serves `search`, `get`, `check`, `map`, `remember`, `update`, `retire`, `import`, and `status` over JSON-RPC stdio. Initialize `instructions` are `rules/consult.md` plus `rules/record.md` so hosts without `skills install` still look things up and distill lasting facts from conversation. Tool descriptions tell agents to search before writing, prefer `update` on an existing record, and skip chat transcripts.

Search works if Ollama is down (keyword-only). Writes never need Ollama. The process cwd must be the repo, or the client must pass `--path`.

`remember` / `update` / `retire` write SQLite only (no markdown file). `remember` splits `applies_to` and `tags` on commas or newlines (same as CLI `--applies-to` / `--tags`). `check` splits `paths` the same way. `import` upserts typed markdown from disk into SQLite (no prune).

`map` takes `query` and optional `limit` (default 30 rows per section) and returns `store.CodeSearch`: matching symbols, the imports their files declare, the files importing the query, and recent commits mentioning it. It needs `archivist index` to have run.

When `log_commands` is true, each tool call is appended to `.archivist/commands.log` (`dir=in` then `dir=out`) via tool-handler middleware. The `mcp` process itself is not logged as a CLI command. Stdio JSON-RPC is never written to the log.

## Connects to

- `retrieve.Engine`, `archive.Service`, `check`, `store.ExploreCode`, embed health/`status`.
- Embedded rule templates in `rules/` (`consult.md`, `record.md`).
- On-demand skills for the per-type write procedure.
- Command log: `log_commands`, `cmdlog.Logger`.

## Entry points

- CLI: `archivist mcp`
- Types: `mcp.Server`, `mcp.ServeStdio`
