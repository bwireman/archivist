---
id: rec_8c31cc5708c0f483ffcc
type: decision
scope: global
status: accepted
title: "Archive hygiene: current facts, stubs for superseded, delete non-records"
tags: [hygiene]
---

## Context

The archive still held pre-pivot indexer ADRs (dump, ADR globs, TUIs, code-body chunks), session "gap" notes marked accepted, and empty Untitled export leftovers. Agents treat this archive as the source of truth, so stale accepted records are worse than missing history.

## Decision

- An accepted record must match current code. If the choice still holds, update the body in place. If a later record replaced it, retire it (`superseded` + `superseded_by`) and leave a short stub, not the obsolete spec.
- Delete records that are not decisions: empty/untitled files, one-off session notes, and "gap catalog" stubs.
- Do not duplicate a typed `rule` inside a `decision` when the rule already exists.
- Prefer one current document per topic (config, retrieval, code map) over a chain of overlapping accepted ADRs.

## Consequences

- Search and `docs/archive/INDEX.md` stay small enough to consult.
- Historical TUI/glob/chunking choices remain findable as superseded stubs.
- New session friction belongs in chat or a confirmed pitfall, not as an accepted decision.
