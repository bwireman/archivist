---
name: record-rule
description: Record a must/must-not or should/should-not rule in the Archivist archive. Use when encoding a constraint, lint-like guidance, or when the user asks to add a rule. Do not use this for how a capability works — that is record-feature.
---

# Record a rule

A `rule` is an enforceable or advisory constraint. It is not a `feature` (how a capability works) and not a `decision` (why we chose it).

1. Pick severity: `must`, `must-not`, `should`, `should-not`.
2. Set `applies_to` path globs so `archivist check` can match touched files.
3. Create the record:

```bash
archivist remember --type rule --scope repo --severity must-not --title "..." --applies-to "internal/**" --body "..."
```

Or MCP tool `remember`.

4. Verify with `archivist check --paths <touched files>` (add `--strict` in CI).
5. Refresh: `archivist index` then `archivist export`.

If you are documenting behavior, connections, or entry points rather than a constraint, use the `record-feature` skill (`--type feature`) instead.
