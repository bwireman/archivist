---
id: rec_7d90d1d2186279928bb4
type: feature
scope: global
status: accepted
title: Agent rules and skills
applies_to: [rules/**, skills/**, internal/skills/**]
tags: [agents, skills]
---

## Purpose

Generate host-specific always-on rules and on-demand skills from templates shipped in the CLI.

## Behavior

`archivist skills install --target cursor|claude|agents-md|copilot` writes consult, record, and refresh rules, and (for Cursor/Claude) the record, refresh, and publish skills. If the target checkout already has `rules/` or `skills/` with real files, those override the embedded copies.

The record rule tells agents to scan this conversation and distill lasting decisions, rules, and features without dumping chat. Skills are the per-type procedure (search first, short body). Cursor rule front matter is `alwaysApply: true` plus a one-line description (`cursorRuleDescription`). `agents-md` and Copilot get concatenated rules only.

## Connects to

- Template trees `rules/` and `skills/` (`go:embed` via `rules/fs.go`, `skills/fs.go`).
- MCP initialize instructions reuse consult + record.
- Decision: split always-on rules from on-demand skills; ship templates in the CLI.

## Entry points

- CLI: `archivist skills install --target cursor`
- Types: `skills.Install`
