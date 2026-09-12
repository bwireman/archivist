---
id: rec_bbf3e07cb5eeec0bd837
type: decision
scope: global
status: accepted
title: Separate SQLite index for global ADRs
---

# Separate SQLite index for global ADRs

- Status: accepted
- Date: 2026-08-28
- Scope: global

## Context
In-repo `docs/global-decisions/` and `~/.archivist/decisions/` were stored in each checkout's `.archivist/index.db`. Default search mixed product decisions into project context, and one repo could not consult another checkout's global ADRs without also ingesting that repo's code.

## Decision
Keep two SQLite files. The repo index (`.archivist/index.db`) holds this checkout only: code, docs, git, and repo ADRs. The machine-wide index (`~/.archivist/global.db` by default) holds user ADRs plus files that match `index.adr.global`. `store.global_path` is optional; a relative value resolves under `~/.archivist/`, not the repo. Default search and dump use the repo DB. `--adr-scope global` (and `make dump-docs`'s global half) use the global DB. User ADRs are stored as `user/<file>`. In-repo global chunks stamp `origin=repo` and `origin_root=<abs repo>` so prune only removes this checkout's missing files. Git history is never written to the global DB.

## Consequences
- Agents can search global ADRs from any repo after that machine has indexed them, without pulling in another project's code.
- Default repo search no longer returns global ADRs.
- The same relative path from two products last-writer-wins in `global.db`.
- Global queries use the current embed model; vectors written with a different model will not rank well until reindexed.
- Untagged in-repo global chunks (no `origin_root`) are left in place until overwritten.
- Scoped `--scope` index does not prune in-repo globals outside that prefix.
