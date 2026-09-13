---
id: rec_c8adf16494471b04e19a
type: feature
scope: global
status: accepted
title: Command log
applies_to: [internal/cmdlog/**,internal/cmd/**,internal/mcp/**,internal/config/**]
---

## Purpose

Opt-in audit log of archive CLI commands and MCP tool calls, written as JSONL under `.archivist/commands.log`.

## Behavior

`log_commands` defaults false. When true, each archive command or MCP tool appends two lines: `dir=in` with arguments, then `dir=out` with parsed JSON result (MCP) or ok/error (CLI) plus `duration_ms`. `init`, `version`, `skills`, and `archivist mcp` itself are skipped; MCP tool calls are still logged. Write errors are swallowed. Fields over 64KiB become `{truncated, bytes, preview}`.

Off: no file is created. On: `.archivist/` is created if needed. The path is not configurable.

## Connects to

- Config: `log_commands`.
- Path: `.archivist/commands.log` (`config.CommandsLogPath`).
- CLI wrap in `internal/cmd`; MCP `WithToolHandlerMiddleware`.

## Entry points

- Config: `.archivist.json` `log_commands`
- Types: `cmdlog.Logger`, `cmdlog.FromConfig`
- CLI: wrapped `search`, `status`, `remember`, `update`, `retire`, `check`, `index`, `embed`, `export`, `publish`, `migrate records`
- MCP: all tools when the flag is on
