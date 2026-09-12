---
id: rec_daaae5decc593a5921ad
type: decision
scope: global
status: accepted
title: Line-based Gleam map and remapping on extractor version
tags: [codemap, gleam]
---

## Context

The code map used tree-sitter for Go, Python, JavaScript, TypeScript, Rust, and Java. Everything else fell back to a line matcher that only recognized `func `, `def `, `class `, and `export `. Gleam (`pub fn`, `fn`, `pub type`, `const`) produced no symbols. `docs/archive/map.md` skips files with an empty symbol list, so a Gleam project such as go-over only showed its two `.mjs` FFI files. Those `.mjs` files were themselves unmatched as JavaScript (only `.js`/`.jsx` were wired). Reindexing after an extractor fix still skipped files whose content hashes had not changed.

## Decision

- Keep tree-sitter for `.go`, `.py`, `.js`/`.jsx`/`.mjs`/`.cjs`, `.ts`/`.tsx`/`.mts`/`.cts`, `.rs`, and `.java`.
- Add a line-based Gleam extractor (`.gleam`) for top-level `import`, `fn`, `type` (including constructors), and `const`. Do not add tree-sitter-gleam; `go-tree-sitter` does not ship that grammar.
- Leave the generic fallback conservative so markdown and config files do not pollute the map.
- Store `codemap_version` in index meta. Bump `codemap.Version` when extractors change. A full `archivist index` remaps files even when content hashes match if that version is missing or stale. Do not stamp the version after a scoped index.

## Consequences

- Gleam sources appear in `map.md` and `archivist map` after the next full index.
- Existing indexes remap once without deleting `.archivist/index.db`.
- Other languages without a tree-sitter spec still need a dedicated extractor or they stay off the map.
- `.mjs` symbols are names from the JavaScript grammar (`parse_adv`) rather than the whole `export function` line.
