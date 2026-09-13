---
id: rec_8a68f4096ab3498eeaa3
type: decision
scope: global
status: accepted
title: Distill conversation into the archive without glut
tags: [agents, mcp, hygiene]
---

## Context

Always-on `rules/record.md` told agents to write a record when they chose among alternatives, but not to mine this conversation. Skill descriptions triggered mainly when the user said "remember". Durable choices died in chat. The opposite failure — dumping session notes — is already a rule.

Alternatives: keep recording reactive (explicit remember only); auto-dump every turn into notes; distill lasting facts in-band with a quality bar.

## Decision

Agents must scan this conversation and distill lasting decisions, rules, and features in the same turn they appear. They must not wait for an explicit remember. Quality bar: one current document per topic; search first; update or retire instead of parallel notes; short bodies; skip transcripts, session errors, and code-only detail.

Ship that bar in three places that stay aligned:

- Always-on `rules/record.md` (when).
- On-demand record skills (how, including search-first).
- MCP initialize `instructions` copied from the consult + record templates, plus tool descriptions that prefer search/update over glut.

## Consequences

- Hosts that only configure `archivist mcp` still get consult + distill guidance.
- Editing `rules/consult.md` or `rules/record.md` changes MCP initialize instructions on the next binary build (this checkout's `skills install` already reads the local template tree).
- The session-notes rule remains the checkable constraint against chat glut.
