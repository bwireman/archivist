---
id: rec_a774cf613e4d5dd33694
type: rule
scope: global
status: accepted
title: Never pass unsanitized text to FTS5 MATCH
severity: must
applies_to: [internal/store/**, internal/retrieve/**]
tags: [search, fts]
---

User and agent strings go through `fts5Query` (or equivalent) before `records_fts MATCH`. Punctuation and FTS5 operators in the input must not produce a syntax error. If MATCH still fails with an FTS5 syntax error, return no hits instead of failing the search.
