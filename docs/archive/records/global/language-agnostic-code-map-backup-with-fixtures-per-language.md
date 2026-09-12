---
id: rec_43749304c84ae79978c9
type: decision
scope: global
status: accepted
title: Language-agnostic code map backup with fixtures per language
tags: [codemap]
---

## Context

The generic fallback only matched `func `, `def `, `class `, and `export ` and stored the whole line. Unsupported languages produced empty maps when tree-sitter failed or was absent.

## Decision

- Tree-sitter (Go, Python, JS/JSX/MJS/CJS, TS/TSX/MTS/CTS, Rust, Java) and the Gleam line extractor remain primary.
- Backup is language-agnostic: a declaration keyword plus an identifier, with a short modifier list. Imports use `import`/`use`/`require`/`include`/`from`. Store the identifier, not the whole line.
- Skip docs and data extensions. Skip English stopword names.
- If a primary extractor errors or returns no symbols and no imports, run the backup and keep any package name already found.
- `codemap.Version` is 2. Adding a language also requires a fixture (see the fixture rule).

## Consequences

Keyword languages without a grammar appear in the map. Docs, configs, and lockfiles stay off it. Tree-sitter still wins when it finds symbols.
