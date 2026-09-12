---
id: rec_293e0ad9ce7ae2e270dd
type: decision
scope: global
status: accepted
title: Default ADR globs are the product directories
---

# Default ADR globs are the product directories

- Status: accepted
- Date: 2026-08-28
- Scope: global

## Context
Default `index.adr.repo` also matched `**/adr/**` and `**/ADR*.md` so other layouts would classify as ADRs. This product only uses `docs/decisions/` and `docs/global-decisions/`, which is what the Cursor rules and `archivist init` already create. Extra globs made the config look unrelated to those directories.

## Decision
Default `records.repo` is `docs/decisions`. Default `records.global` is `docs/global-decisions`. Other layouts stay possible as an explicit directory override. Omitting `records` keeps these defaults.

## Consequences
- `.archivist.json` `records.repo` / `records.global` match the directories agents write to.
- Files like `docs/ADR-001.md` are ordinary docs unless the repo changes those directories.
- `~/.archivist/records/` is still implicit unless `records.dev` is set.
