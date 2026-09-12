---
id: rec_b709c21fa97255bd7629
type: decision
scope: global
status: superseded
title: Nest ADR globs under index.adr
superseded_by: rec_config014
---

# Nest ADR globs under index.adr

- Status: superseded
- Date: 2026-08-28
- Scope: global

## Context
Repo vs global ADRs were configured as sibling lists `adr_paths` and `global_adr_paths`. The names did not say what the lists were for, or how they related.

## Decision
Put both under `index.adr`, with `repo` and `global` glob lists. Global wins if a path matches both. `~/.archivist/decisions/` stays implicit (not a repo glob) and is indexed as `global/<file>`. Old `adr_paths` / `global_adr_paths` keys still load.

## Consequences
- `.archivist.json` reads as "these paths are repo ADRs, these are global ADRs".
- Save writes the nested shape only.
- Empty lists disable that kind; omitting `adr` keeps the defaults.
