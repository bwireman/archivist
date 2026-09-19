---
id: rec_6551943a15f76eed1f64
type: rule
scope: global
status: accepted
title: Check --strict only fails on applies_to glob matches
severity: must
applies_to: [internal/check/**, internal/cmd/remember.go]
tags: [check, strict]
---

`archivist check --strict` and `Result.HasViolation` must be true only when a must/must-not rule's `applies_to` glob matches the supplied paths or diff paths. Semantic search hits are advisory and must not fail CI.
