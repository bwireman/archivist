---
id: rec_43749304c84ae79978c9
type: decision
scope: global
status: accepted
title: Language-agnostic code map backup with fixtures per language
tags: [codemap]
---

## Context

The generic fallback only matched `func `, `def `, `class `, and `export ` and stored the whole line. It stayed that narrow so markdown and config files would not pollute the map, which also meant unsupported languages produced empty maps when tree-sitter failed or was absent. There were no fixtures that proved every registered extension actually extracted symbols.

## Decision

- Tree-sitter (Go, Python, JS/JSX/MJS/CJS, TS/TSX/MTS/CTS, Rust, Java) and the Gleam line extractor remain primary.
- The backup is language-agnostic: a declaration keyword (`function`, `fn`, `fun`, `func`, `def`/`defp`/`defmodule`, `class`, `struct`, `type`, …) plus an identifier, with a short list of modifiers (`pub`, `export`, `public`, …). Imports use `import`/`use`/`require`/`include`/`from`. Store the identifier, not the whole line.
- Skip docs and data extensions (markdown, json, yaml, toml, html, lockfiles, `go.mod`, …). Skip English stopword names so `class of` / `type of` do not become symbols.
- If a primary extractor errors or returns no symbols and no imports, run the backup and keep any package name already found.
- `TestExtractEveryRegisteredLanguage` must include a fixture for every `languageSpecs` key plus `.gleam`. Adding an extension without a fixture fails the test.
- `codemap.Version` is 2.

## Consequences

- Keyword languages without a grammar (Elixir, Ruby, PHP, Kotlin, Swift, Zig, Lua, …) appear in the map.
- Docs, configs, and lockfiles stay off the map without keeping the fallback to four prefixes.
- Tree-sitter still wins when it finds symbols, so maps are not duplicated with generic line hits.
