---
id: rec_9d4061b8f978a752d7f7
type: decision
scope: dev
status: accepted
title: Archivist MCP is configured in .cursor/mcp.json but was not available in this session
tags: [archivist, soon, G-mcp-missing, gap]
---

## Context

Known docs/code gap, dead path, or unverified mismatch. This is recorded so it is not forgotten, not as a completed design choice.

Consult rule prefers MCP. CLI works. Record fallback: archivist search / docs/archive.

## Decision

Until a follow-up changes code or docs, **code behavior is the source of truth**. Do not silently "fix" README, CI comments, or unused helpers without tests.

## Consequences

Treat as follow-up work. Prefer a later `archivist update` or `retire` once resolved.

Catalog id: `G-mcp-missing`.
