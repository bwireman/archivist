---
id: rec_c146ba784a29a4c1aa82
type: decision
scope: global
status: accepted
title: Optional JSONL command log at .archivist/commands.log
applies_to: [internal/config/**,internal/cmdlog/**,internal/cmd/**,internal/mcp/**]
---

## Context

Need an opt-in trace of archive reads and writes (CLI and MCP) without a configurable extra path, and without mixing MCP JSON-RPC stdio into the log.

## Decision

- Top-level bool `log_commands` in `.archivist.json` (default false, omitempty).
- When true, append JSONL to `.archivist/commands.log` (path fixed, like SQLite).
- Each archive operation writes `dir=in` (command + args) then `dir=out` (result or error). MCP tools and archive CLI commands are logged; `init`, `version`, `skills`, and the `mcp` process itself are not.
- Logging failures are ignored so they never fail the command. Payloads larger than 64KiB are truncated.

## Consequences

- Existing configs stay quiet. This checkout sets `log_commands` true.
- The log lives under `.archivist/` so gitignore already covers it.
- Restart `archivist mcp` after flipping the flag.
