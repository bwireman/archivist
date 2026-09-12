---
name: record-decision
description: Record a design decision in the Archivist archive. Use when choosing between real alternatives, or when the user asks to remember, ADR, or write a decision. Do not use this for how a capability works — that is record-feature.
---

# Record a decision

A `decision` captures a choice among real alternatives (Context, Decision, Consequences). It is not a `feature` (how a capability works today) and not a `rule` (must/must-not).

1. Capture Context, Decision, and Consequences in markdown.
2. Create the record:

```bash
archivist remember --type decision --scope repo --title "..." --body "..."
```

Or MCP tool `remember` with the same fields.

3. Scope: `repo` for this checkout, `global` for the product, `dev` for personal notes.
4. Refresh: `archivist index` then `archivist export`.

If you are documenting behavior, connections, or entry points, use the `record-feature` skill (`--type feature`) instead.

Directories come from `.archivist.json` `records.repo` / `records.global` (default `~/.archivist`) / `records.dev` (default `~/.archivist/records`).
