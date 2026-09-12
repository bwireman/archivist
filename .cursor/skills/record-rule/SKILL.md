---
name: record-rule
description: Record a must/must-not or should/should-not rule in the Archivist archive. Use when encoding a constraint, lint-like guidance, or when the user asks to add a rule.
---

# Record a rule

1. Pick severity: `must`, `must-not`, `should`, `should-not`.
2. Set `applies_to` path globs so `archivist check` can match touched files.
3. Create the record:

```bash
archivist remember --type rule --scope repo --severity must-not --title "..." --applies-to "internal/**" --body "..."
```

Or MCP tool `remember`.

4. Verify with `archivist check --paths <touched files>` (add `--strict` in CI).
5. Refresh: `archivist index` then `archivist export`.
