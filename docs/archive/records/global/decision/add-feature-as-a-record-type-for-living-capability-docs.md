---
id: rec_7c4b40de9fa8f90f2ac8
type: decision
scope: global
status: accepted
title: Add feature as a record type for living capability docs
tags: [records]
---

## Context

Typed records were `decision`, `rule`, `guide`, `map`, and `pitfall`. Decisions capture a choice; rules capture constraints; `guide` was unused. There was no type for how a capability works, which packages and stores it uses, or how to invoke it. That knowledge lived in chat or was stuffed into decisions.

Alternatives: overload `guide`; put capability notes only in `docs/archive/map.md`; add a `feature` type.

## Decision

Add `feature` as a first-class record type. A feature document is living: Purpose, Behavior, Connects to, and Entry points. `applies_to` globs point at the implementing packages. Export lists features after decisions. `guide` remains available for how-to procedures; `map` remains the generated code structure tree.

When behavior changes, update the feature. When the capability is removed, retire it. Do not encode a design choice as a feature or a subsystem description as a decision.

## Consequences

- `archivist remember --type feature` and MCP `remember` accept the type without a schema bump (type is a string column).
- Agents should search `--type feature` before changing a subsystem.
- Skills include `record-feature`; the always-on record rule mentions features next to decisions and rules.
