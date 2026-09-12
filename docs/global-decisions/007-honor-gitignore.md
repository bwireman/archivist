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
`index.honor_gitignore` defaults to true. When set, the indexer skips files and directories that match the repo `.gitignore` (root and nested), including directory patterns so ignored trees are not walked. `skip_dirs` and `skip_globs` still apply. User-global ADRs under `~/.archivist/decisions/` are not filtered by a checkout's gitignore. Set the flag false to index gitignored paths.

## Consequences
- `archivist init` writes `honor_gitignore: true`. Set the flag false in `.archivist.json` to index gitignored paths.
- Omitting the key in an existing `.archivist.json` still means true.
- Gitignored files that were already in the index are pruned on the next index.
