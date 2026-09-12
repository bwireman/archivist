---
name: record-decision
description: Record a design decision in the Archivist archive. Use when choosing between real alternatives, or when the user asks to remember, ADR, or write a decision.
---

# Record a decision

1. Capture Context, Decision, and Consequences in markdown.
2. Create the record:

```bash
archivist remember --type decision --scope repo --title "..." --body "..."
```

Or MCP tool `remember` with the same fields.

3. Scope: `repo` for this checkout, `global` for the product, `dev` for personal notes.
4. Refresh: `archivist index` then `archivist export`.

Directories come from `.archivist.json` `records.repo` / `records.global` / `records.dev`.
