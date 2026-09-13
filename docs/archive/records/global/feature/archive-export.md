---
id: rec_9e0fa8636b4267f0d942
type: feature
scope: global
status: accepted
title: Archive export
applies_to: [internal/export/**, internal/cmd/export.go]
tags: [export]
---

## Purpose

Generate a markdown tree under `records.export` (default `docs/archive/`) so humans and agents can read the archive without MCP.

## Behavior

`archivist export` collects records from the repo and home stores, then writes:

- `INDEX.md` — catalog grouped by `record.IndexOrder`, with links to type digest files.
- One full-text digest per type that has records: `rules.md`, `decisions.md`, `features.md`, `guides.md`, `maps.md`, `pitfalls.md`. `maps.md` is the map-type digest so it does not collide with `map.md` (the generated code map). `rules.md` is always written, even if empty, because consult instructions name that path. Empty types other than rule remove a leftover digest file.
- Individual copies at `records/<scope>/<type>/<slug>.md` (scope first, then type). The `records/` directory is replaced each export so old layout files do not linger.
- `map.md` — code structure (symbols per file).
- `archive.json` — machine-readable manifest.

Do not hand-edit `docs/archive/`; regenerate with `archivist export`. `--bundle` writes the same tree to a chosen directory. Publish uses `WriteBundle` for that tree.

## Connects to

- Config: `records.export`.
- `record.IndexOrder` for section and digest order.
- Typed records and the consult rule (`docs/archive/INDEX.md` then type digests).
- Publish destinations consume the same bundle layout.

## Entry points

- CLI: `archivist export`, `archivist export --bundle <dir>`
- Types: `export.Run`, `export.WriteBundle`
