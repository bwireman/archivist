---
id: rec_330b72a24aff842edf5a
type: decision
scope: global
status: accepted
title: "Default embed model is qwen3-embedding:0.6b"
supersedes: ["rec_d517cd894e443ee0d4f2"]
---

## Context

`archivist init` used to write `nomic-embed-text`. New repos should match the model this checkout already uses.

## Decision

`config.Default()` and `archivist init` use `qwen3-embedding:0.6b`. A repo may override `ollama.embed_model` in `.archivist.json`.

## Consequences

`ollama pull qwen3-embedding:0.6b` is the documented setup step. Changing the embed model later requires re-embedding; existing vectors are not comparable across models.
