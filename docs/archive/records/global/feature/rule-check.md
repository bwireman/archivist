---
id: rec_4ffd6c7ec10e0f187739
type: feature
scope: global
status: accepted
title: Rule check
applies_to: [internal/check/**, internal/cmd/remember.go, internal/mcp/**]
tags: [check, rules]
---

## Purpose

Match archive rules against a proposed change so agents and CI can see which constraints apply.

## Behavior

`check.Run` loads every `rule` record from the repo and home stores. Rules whose `applies_to` globs match the supplied paths are listed first. An optional description runs hybrid search over rules; those hits are advisory. Diff text adds paths from `diff --git`, `---` and `+++` lines. `/dev/null` is ignored; a rename contributes both old and new paths. MCP `paths` may be a comma- or newline-separated list.

`HasViolation` is true only when a must/must-not rule matched via `applies_to`. Semantic-only hits never set it. CLI `--strict` exits nonzero on `HasViolation`.

## Connects to

- `retrieve.Engine` for semantic rule search.
- `record.MatchesPaths` / `IsEnforceable`.
- Typed records and the record-rule skill.

## Entry points

- CLI: `archivist check [description] [--paths] [--diff] [--strict]`
- MCP: `check`
- Types: `check.Run`, `check.Result`
