---
id: rec_be2f7f9e6ad99ea65edb
type: feature
scope: global
status: accepted
title: Code map exploration
applies_to: [internal/codemap/**, internal/index/**, internal/gitindex/**, internal/cmd/map.go]
---

## Purpose

Answer "where does this live and how did it get here" from indexed code structure, without reading source bodies.

## Behavior

`archivist index` walks the checkout (honoring `.gitignore`, `index.skip_globs`, and the export dir), extracts symbols and import edges per file, and records the last 500 git commits. Record files under `records.repo` / in-repo `records.global` are skipped before being read.

`store.ExploreCode(query, limit)` composes four reads and is the single backing call for both surfaces:

- `SearchSymbols` — `name`, `doc_line`, or `file_path` contains the query
- `ImportsFrom` — import edges declared by the files holding those symbols
- `ImportersOf` — edges whose `to_path` contains the query (who imports this)
- `SearchCommits` — commit subject or body contains the query, newest first

All four use `likeContains`, which escapes `%`, `_`, and `\` so a query of `%` matches nothing rather than everything.

Extractors: tree-sitter for Go, Python, JS/TS, Rust, Java; a hand-rolled Gleam matcher; a language-agnostic line matcher as backup when the others error or find nothing. `codemap.Version` gates remapping — bump it when extractor output changes and the next full index re-extracts files whose content hash is unchanged.

## Connects to

- Tables `files`, `symbols`, `symbol_edges`, `commits`; meta key `codemap_version`.
- `docs/archive/map.md` lists symbols only (export uses `AllSymbols`).

## Entry points

- CLI: `archivist index`, `archivist map <query>` (`--limit`, `--json`)
- MCP: `map` (`query`, `limit`)
- Types: `store.ExploreCode`, `store.CodeSearch`, `codemap.Extract`, `index.Indexer`
