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

4. Before remember, search with no scope argument. Search covers repo, global, and dev.
   If a current record covers the topic, update it and keep its scope.
   If search shows a gap, pick one scope:
   - dev: about the person or this machine, including a short-gap answer
   - repo: true only in this checkout
   - global: true for the product in every checkout

   The `--scope repo` flag in the example is a checkout-only decision. It is not the default for every decision.

   Then `archivist embed --once`. Records live in SQLite; `remember` does not write markdown. Use `archivist export` only when `records.write_docs` is true. `records.repo` / `records.global` / `records.dev` are optional import drop folders.
