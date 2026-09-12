---
id: rec_d5b84579e761adee897e
type: decision
scope: dev
status: accepted
title: records.dev is unset (defaults to ~/.archivist/records)
tags: [archivist, later, G-records-dev-unset, gap]
---

## Context

Known docs/code gap, dead path, or unverified mismatch. This is recorded so it is not forgotten, not as a completed design choice.

Fine if personal notes stay out of the repo. Record that split so agents do not put workflow prefs in docs/decisions.

## Decision

Until a follow-up changes code or docs, **code behavior is the source of truth**. Do not silently "fix" README, CI comments, or unused helpers without tests.

## Consequences

Treat as follow-up work. Prefer a later `archivist update` or `retire` once resolved.

Catalog id: `G-records-dev-unset`.
