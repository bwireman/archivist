---
name: record-feature
description: Distill how a capability works into an Archivist feature record. Use when this conversation or a code change establishes behavior, connections, or entry points, or when the user asks to remember a feature. Do not use for why we chose it (record-decision) or for must/must-not constraints (record-rule).
---

# Record a feature

A `feature` is living documentation of a capability. It is not a design decision and not a rule. When behavior changes, `update` the feature; when the capability is removed, `retire` it. Search `--type feature` before creating a second document.

1. Search for an existing feature on this capability. Prefer `update`.
2. Set `applies_to` path globs for the implementing packages. `archivist check` only matches `rule` records.
3. Capture only what a later agent needs. Skip chat narration.

```markdown
## Purpose

One or two sentences.

## Behavior

How it works today, including failure and empty cases.

## Connects to

Stores, other features, config keys, and record types.

## Entry points

CLI, MCP, and the main Go types.
```

4. Create only if search shows a gap:

```bash
archivist remember --type feature --scope global --title "..." --applies-to "internal/embed/**" --tags queue --body "..."
```

Or MCP `remember` with `type=feature`.

5. Scope: `global` for product capabilities, `repo` for this checkout only, `dev` for personal notes. Then `archivist index` and `archivist export`.
