---
id: rec_2a6281f6b3917c11dc0b
type: decision
scope: global
status: accepted
title: Separate global and repo ADRs
---

# Separate global and repo ADRs

- Status: accepted
- Date: 2026-08-28
- Scope: global

## Context
Product decisions (index TUI, init TUI) were stored next to this checkout's repo-specific choices (embed model). Agents need to tell tool-wide decisions from project-local ones, and a user may keep ADRs that apply to every repo.

`--scope` already means a path prefix or glob. Reusing it for ADR kind would collide with dump and index flags.

## Decision
Classify ADRs as `repo` or `global`. Repo ADRs live in `docs/decisions/`. In-repo global ADRs live in `docs/global-decisions/` (a sibling directory, so `docs/decisions/**` does not match them). User-global ADRs live in `~/.archivist/decisions/`. Filter with `--adr-scope repo|global`. `make dump-docs` writes `docs/dump/decisions.md` and `docs/dump/global-decisions.md`. Storage is two SQLite files; see `006-separate-global-index.md`.

## Consequences
- Search and dump can list only repo or only global ADRs.
- Path globs are the source of truth for classification; a `- Scope:` line in the markdown is for humans.
- Existing ADR chunks without `adr_scope` metadata are treated as repo.
- `archivist init` creates the two in-repo directories. It does not create `~/.archivist/decisions/`.
