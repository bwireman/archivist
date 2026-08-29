# Default ADR globs are the product directories

- Status: accepted
- Date: 2026-08-28
- Scope: global

## Context
Default `index.adr.repo` also matched `**/adr/**` and `**/ADR*.md` so other layouts would classify as ADRs. This product only uses `docs/decisions/` and `docs/global-decisions/`, which is what the Cursor rules and `archivist init` already create. Extra globs made the config look unrelated to those directories.

## Decision
Default `index.adr.repo` is `docs/decisions/**`. Default `index.adr.global` is `docs/global-decisions/**`. Other layouts stay possible as an explicit override. Empty lists still disable that kind; omitting `adr` still keeps these defaults.

## Consequences
- `.archivist.json` `index.adr` matches the directories agents write to.
- Files like `docs/ADR-001.md` are ordinary docs unless the repo adds a glob.
- `~/.archivist/decisions/` is still implicit, not a glob.
