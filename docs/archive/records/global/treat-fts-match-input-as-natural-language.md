---
id: rec_d5453789652869540dbb
type: decision
scope: global
status: accepted
title: Treat FTS MATCH input as natural language
tags: [search, fts]
---

## Context

User and agent queries include paths (`docs/foo.md`), config keys (`records.global`), and model names (`qwen3-embedding:0.6b`). Passing them unchanged to FTS5 MATCH treats `/`, `.`, `:`, quotes, and `AND`/`OR`/`NOT` as syntax, so `search` and `check` fail with `fts5: syntax error`.

Alternatives: document FTS5 syntax and require callers to escape; wrap the whole query as one phrase; split into letter/number tokens and quote each as a term.

## Decision

Treat MATCH input as natural language, not FTS5 syntax. Split on non-alphanumeric characters, quote each token (so operators in the input are terms), and combine with implicit AND. If a query has no tokens or MATCH still reports a syntax error, return no FTS hits instead of failing hybrid search. Vector search still uses the raw string.

## Consequences

- Paths and dotted names search as tokens and no longer crash the CLI or MCP.
- Callers cannot use FTS5 operators or prefix `*` as syntax; punctuation is a separator.
- `check` descriptions that happen to be file paths become token queries instead of errors.
