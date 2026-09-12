---
id: rec_330b72a24aff842edf5a
type: decision
scope: global
status: accepted
title: "Default embed model is qwen3-embedding:0.6b"
---

# Default embed model is qwen3-embedding:0.6b

- Status: accepted
- Date: 2026-08-28
- Scope: global

## Context
`archivist init` wrote `nomic-embed-text` via `config.Default()`. This checkout already indexes with `qwen3-embedding:0.6b` because that model is installed locally. New repos should get the same default so they can index without a per-repo pin.

## Decision
`config.Default()` and `archivist init` use `qwen3-embedding:0.6b`. A repo may override `ollama.embed_model` in `.archivist.json`.

## Consequences
- `ollama pull qwen3-embedding:0.6b` is the documented setup step.
- Existing configs that set `nomic-embed-text` keep that model.
- Changing the embed model later requires a full reindex; existing vectors are not comparable across models.
- The repo-only pin in `docs/decisions/001-use-qwen3-embedding.md` is superseded.
