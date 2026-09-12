---
id: rec_0d9749973b9e9660b83a
type: rule
scope: global
status: accepted
title: Do not store session notes as accepted archive records
severity: should-not
applies_to: [docs/decisions/**, docs/global-decisions/**]
tags: [hygiene]
---

Ephemeral session observations (MCP down this chat, config currently unset, unverified "gap" catalogs) are not accepted decisions. Record a decision or rule only when choosing between alternatives or encoding a lasting constraint. Prefer `update` or `retire` over adding a parallel accepted note.
