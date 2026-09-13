---
id: rec_6b33fa3dd9c4fec4fd8e
type: rule
scope: global
status: accepted
title: Code map tests must fixture every registered language
severity: must
applies_to: [internal/codemap/**]
tags: [codemap]
---

`TestExtractEveryRegisteredLanguage` must include a sample for every key in `languageSpecs` and for `.gleam`. A new tree-sitter or line-based language is not done until that table has a fixture that extracts at least one expected symbol.
