# Use qwen3-embedding:0.6b for this repository

- Status: superseded
- Date: 2026-08-28
- Scope: repo

## Context
`archivist init` wrote `nomic-embed-text` as the default Ollama embed model. This repository needed a model that is actually installed locally so `make index` can run.

## Decision
Pin `.archivist.json` to `qwen3-embedding:0.6b` on `http://localhost:11434`. Leave the tool default in `config.Default()` as `nomic-embed-text`.

Superseded by `docs/global-decisions/009-qwen3-embed-default.md`: the tool default is now `qwen3-embedding:0.6b`, so this checkout no longer needs a special pin.

## Consequences
- Indexing this repo works without pulling `nomic-embed-text`.
- Other repos created with `archivist init` still get `nomic-embed-text` unless they override it.
- Changing the embed model later requires a full reindex; existing vectors are not comparable across models.
