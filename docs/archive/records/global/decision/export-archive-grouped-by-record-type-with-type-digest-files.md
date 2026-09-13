---
id: rec_126bb406288fdea6a7c8
type: decision
scope: global
status: accepted
title: Export archive grouped by record type with type digest files
tags: [export, records]
---

## Context

Export copied every record under `records/<scope>/` regardless of type. `INDEX.md` already grouped titles by type, and only rules had a compiled digest (`rules.md`). Agents reading the generated tree mixed features, decisions, and rules in one directory and had to open many files to learn one type.

Alternatives: keep mixed copies and only add digest files; nest copies as `records/<type>/<scope>/`; nest copies as `records/<scope>/<type>/`; split source record directories by type.

## Decision

`archivist export` groups generated copies as `records/<scope>/<type>/<slug>.md` (scope first, then type) and writes a full-text digest file per type that has records (`rules.md`, `decisions.md`, `features.md`, `guides.md`, `maps.md`, `pitfalls.md`). `rules.md` is always written so consult instructions can name that path. `map.md` stays the code map; map-type records go to `maps.md`. Source record directories stay organized by scope; type lives in front matter.

The `records/` export directory is replaced each run so the old `records/<scope>/` and `records/<type>/<scope>/` layouts do not linger.

## Consequences

- Bots can read one digest per type or glob `docs/archive/records/<scope>/<type>/`.
- Scope stays the primary folder, matching source dirs and overlay (repo / global / dev).
- `INDEX.md` links include the type segment and point at type digests.
- The consult rule names type digest files next to `INDEX.md`.
- Publish bundles pick up the same layout.
