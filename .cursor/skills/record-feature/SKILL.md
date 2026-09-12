---
name: record-feature
description: Record how a product capability works in the Archivist archive. Use when documenting a feature, subsystem, queue, CLI/MCP surface, or what it connects to; or when the user asks to remember a feature.
---

# Record a feature

A `feature` record is living documentation of a capability: how it behaves, which packages and stores it uses, and how agents or humans invoke it. It is not a design decision (why we chose it) and not a rule (must/must-not).

1. Set `applies_to` path globs for the implementing packages so agents can tie the feature to the code. `archivist check` only matches `rule` records; look features up with `archivist search --type feature`.
2. Capture Purpose, Behavior, Connects to, and Entry points:

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

3. Create the record:

```bash
archivist remember --type feature --scope global --title "..." --applies-to "internal/embed/**" --tags queue --body "..."
```

Or MCP tool `remember` with `type=feature`.

4. Scope: `global` for product capabilities, `repo` for this checkout only, `dev` for personal notes.
5. When behavior changes, `archivist update` the feature. When the capability is removed, `archivist retire` it.
6. Refresh: `archivist index` then `archivist export`.
