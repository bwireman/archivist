---
name: record-decision
description: Distill a lasting design choice into an Archivist decision record. Use when this conversation or the code chooses among real alternatives, or when the user asks to remember, ADR, or write a decision. Do not use for how a capability works (record-feature) or for must/must-not constraints (record-rule). Skip chat logs and ephemeral session notes.
---

# Record a decision

A `decision` captures a choice among real alternatives (Context, Decision, Consequences). It is not a `feature` (how it works today) and not a `rule` (must/must-not).

1. Search for the same topic (`search` or `archivist search --type decision`). If a current record exists, `update` it (or `retire` and replace). Do not add a parallel accepted note.
2. Distill a short body. Omit the transcript, options that were never in play, and implementation detail that lives in code.
3. Create only if search shows a gap:

```bash
archivist remember --type decision --scope repo --title "..." --body "..."
```

Or MCP `remember` with the same fields.

4. Scope: `repo` this checkout, `global` the product, `dev` personal. Then `archivist embed --once`. Records live in SQLite; `remember` does not write markdown. Use `archivist export` only when `records.write_docs` is true. `records.repo` / `records.global` / `records.dev` are optional import drop folders.
