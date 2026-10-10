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

5. Before remember, search with no scope argument. Search covers repo, global, and dev.
   If a current record covers the topic, update it and keep its scope.
   If search shows a gap, pick one scope:
   - dev: about the person or this machine, including a short-gap answer
   - repo: true only in the checkout the record is about. Write the row to that checkout's database. The process checkout is the context when archive is empty. archive names a listed extra root when the record is about that checkout.
   - global: true for the product in every checkout

   The `--scope global` flag in the example is a product capability. It is not the default for every feature.

   Then `archivist embed --once` if the body changed. Records live in SQLite; `remember` does not write markdown.
