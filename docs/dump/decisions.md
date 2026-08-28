# Archivist context dump

Retrieved chunks from a local code index. Use only facts present here.

- Query: (all matching chunks)
- Type: adr
- Chunks: 4

## `docs/decisions/001-use-qwen3-embedding.md` (adr, lines 1-5)

```md
# Use qwen3-embedding:0.6b for this repository

- Status: accepted
- Date: 2026-08-28
```


## `docs/decisions/001-use-qwen3-embedding.md` (adr, lines 6-8)

```md
## Context
`archivist init` writes `nomic-embed-text` as the default Ollama embed model. This repository needed a model that is actually installed locally so `make index` can run.
```


## `docs/decisions/001-use-qwen3-embedding.md` (adr, lines 9-11)

```md
## Decision
Pin `.archivist.json` to `qwen3-embedding:0.6b` on `http://localhost:11434`. Leave the tool default in `config.Default()` as `nomic-embed-text`.
```


## `docs/decisions/001-use-qwen3-embedding.md` (adr, lines 12-16)

```md
## Consequences
- Indexing this repo works without pulling `nomic-embed-text`.
- Other repos created with `archivist init` still get `nomic-embed-text` unless they override it.
- Changing the embed model later requires a full reindex; existing vectors are not comparable across models.
```

