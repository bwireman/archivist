---
id: rec_daaae5decc593a5921ad
type: decision
scope: global
status: accepted
title: Line-based Gleam map and remapping on extractor version
tags: [codemap, gleam]
---

## Context

Tree-sitter covered Go, Python, JS, TS, Rust, and Java. Gleam (`pub fn`, `fn`, `pub type`, `const`) produced no symbols, and `go-tree-sitter` has no Gleam grammar. Reindexing after an extractor fix skipped files whose content hashes had not changed.

## Decision

- Use a line-based Gleam extractor (`.gleam`) for top-level `import`, `fn`, `type` (including constructors), and `const`. Do not add tree-sitter-gleam.
- Store `codemap_version` in index meta. Bump `codemap.Version` when extractors change so a full index remaps unchanged files. Do not stamp the version after a scoped index.

## Consequences

Gleam sources appear in `map.md` after a full index. Existing indexes remap once without deleting `.archivist/index.db`. The generic fallback for other languages is a later decision (language-agnostic backup).
