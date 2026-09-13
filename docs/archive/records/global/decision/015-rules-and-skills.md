---
id: rec_skills015
type: decision
scope: global
status: accepted
title: Split always-on rules from on-demand skills
---

## Context

`archivist skills install --target cursor` copied `skills/*/SKILL.md` into `.cursor/rules/` with `alwaysApply: false`. Those files were short “always do this” lists, so they behaved like rules, while Cursor skills actually live under `.cursor/skills/`. This repo also had hand-written always-on rules that duplicated the generated files.

## Decision

Keep two template trees:

- `rules/` — always-on constraints (consult, record, refresh).
- `skills/` — on-demand procedures with SKILL.md front matter (`record-decision`, `record-rule`, `record-feature`, `refresh-archive`, `publish-archive`).

Install maps them to the host: Cursor rules + `.cursor/skills/`; Claude skills + `CLAUDE.md`; `agents-md` and Copilot get concatenated rules only.

## Consequences

- Consult is a rule, not a skill.
- `skills install` is the generator; do not hand-edit the host files it writes.
