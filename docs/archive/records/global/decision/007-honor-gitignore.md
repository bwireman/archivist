---
id: rec_6e94d1190b59d6365335
type: decision
scope: global
status: accepted
title: Honor .gitignore when indexing
---

## Context

`skip_dirs` duplicated `.gitignore`. Build artifacts were easy to index by accident.

## Decision

The indexer always honors `.gitignore` (root and nested). `.git` and `.archivist` are always skipped. Extra excludes go in `index.skip_globs`. There is no `honor_gitignore` flag. Dev records under `~/.archivist/records/` are not filtered by a checkout's gitignore.

## Consequences

Gitignored files already in the index are pruned on the next index.
