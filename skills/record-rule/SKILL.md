---
name: record-rule
description: Distill a lasting constraint into an Archivist rule record. Use when this conversation states a must/must-not or should/should-not that future work should follow, or when the user asks to add a rule. Do not use for how a capability works (record-feature) or for a choice among alternatives (record-decision). Skip one-off session friction.
---

# Record a rule

A `rule` is an enforceable or advisory constraint. It is not a `feature` (how a capability works) and not a `decision` (why we chose it).

1. Search for an existing rule on the same constraint. Prefer `update` over a duplicate.
2. Pick severity: `must`, `must-not`, `should`, `should-not`.
3. Set `applies_to` path globs so `archivist check` can match touched files.
4. Keep the body to the constraint and why it matters. No transcript.
5. Create only if search shows a gap.

   Before remember, search with no scope argument. Search covers repo, global, and dev.
   If a current record covers the topic, update it and keep its scope.
   If search shows a gap, pick one scope:
   - dev: about the person or this machine, including a short-gap answer
   - repo: true only in this checkout
   - global: true for the product in every checkout

   The `--scope repo` flag in the example is a checkout-only rule. It is not the default for every rule.

```bash
archivist remember --type rule --scope repo --severity must-not --title "..." --applies-to "internal/**" --body "..."
```

Or MCP `remember`.

6. Verify with `archivist check --paths <touched files>` (add `--strict` in CI). Then `archivist embed --once` if the body changed. Records live in SQLite; `remember` does not write markdown.
