---
id: rec_config014
type: decision
scope: global
status: accepted
title: Reshape config around records directories
supersedes: [rec_b709c21fa97255bd7629, rec_bbf3e07cb5eeec0bd837, rec_293e0ad9ce7ae2e270dd]
---

## Context

`.archivist.json` still described the old code indexer: ADR globs, skip_dirs overlapping gitignore, and configurable SQLite paths. Record locations were hardcoded. This checkout's `store.global_path` still pointed at `global.db` while the default was `archive.db`.

## Decision

- `records.repo` / `records.global` / `records.dev` / `records.export` are directories.
- Empty `dev` means `~/.archivist/records`. Empty `global` means `~/.archivist` (not `docs/global-decisions`).
- This product sets `records.global` to `docs/global-decisions` so product-wide records stay in git.
- Always skip `.git` and `.archivist`. Always honor `.gitignore`. Extra excludes go in `index.skip_globs`.
- SQLite is always `.archivist/index.db` and `~/.archivist/archive.db`.
- Keep `ollama.base_url`, `embed_model`, `embed_timeout` (default `5m`). Empty URL uses `$OLLAMA_HOST`.
- Keep `publish.destinations`. Drop `adr_paths` / `index.adr` / `store.*` / `honor_gitignore` / `skip_dirs`.

## Consequences

- Existing configs that only set Ollama still load.
- Configs that relied on `store.global_path` or `index.adr` need a one-line edit.
- `vendor/` and `node_modules/` are skipped only if gitignored or listed in `skip_globs`.
