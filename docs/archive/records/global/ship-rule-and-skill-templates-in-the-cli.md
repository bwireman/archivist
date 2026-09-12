---
id: rec_f6175938c024887eacfb
type: decision
scope: global
status: accepted
title: Ship rule and skill templates in the CLI
tags: [skills, embed]
---

## Context

`archivist skills install` read `rules/` and `skills/` from the target repository. That works in this checkout, but a consumer repo like go-over has no template tree, so install failed.

Alternatives: copy templates into every repo; require running install from this source tree; look up files next to the executable; embed the templates in the binary.

## Decision

Embed `rules/*.md` and `skills/*/SKILL.md` in the CLI. `skills install` uses the shipped templates. If the target checkout already has `rules/` or `skills/` with real files, those override the embedded copies so this repo can iterate without a rebuild.

## Consequences

- `go:embed` lives next to the files (`rules/fs.go`, `skills/fs.go`) because embed cannot reach outside its package.
- Consumer repos get Cursor rules and skills from `archivist skills install --target cursor` after `go install`.
- Editing templates in this repo still requires `make install` (or a local `rules/`/`skills/` tree) before host files pick up the change.
