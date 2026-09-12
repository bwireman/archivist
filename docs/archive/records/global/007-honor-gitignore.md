---
id: rec_6e94d1190b59d6365335
type: decision
scope: global
status: accepted
title: Honor .gitignore when indexing
---

# Honor .gitignore when indexing

- Status: accepted
- Date: 2026-08-28
- Scope: global

## Context
`skip_dirs` and `skip_globs` were a second, incomplete copy of what most repos already list in `.gitignore`. Build artifacts, local binaries, and editor dirs were easy to index by accident.

## Decision
The indexer always honors `.gitignore` (root and nested). `.git` and `.archivist` are always skipped. Extra excludes go in `index.skip_globs`. Dev records under `~/.archivist/records/` are not filtered by a checkout's gitignore.

## Consequences
- There is no `honor_gitignore` flag; gitignored paths are never indexed.
- Extra excludes that are not in gitignore belong in `index.skip_globs`.
- Gitignored files that were already in the index are pruned on the next index.
