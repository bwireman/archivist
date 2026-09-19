# Rules

## Append gitignore entries on their own line

- Status: accepted
- Scope: global
- Severity: must
- Applies to: internal/cmd/**
- Tags: init, gitignore

When appending a line to `.gitignore`, insert a newline first if the file does not already end with one. Otherwise `init` glues `.archivist/` onto the last existing entry and the ignore never matches.

---

## Bump codemap.Version when extractors change

- Status: accepted
- Scope: global
- Severity: must
- Applies to: internal/codemap/**
- Tags: codemap

When adding or changing a code-map extractor (tree-sitter specs, line-based languages, or the generic fallback), increment `codemap.Version`. Index skips unchanged files by content hash; a stale `codemap_version` in meta is what forces a remap of existing trees.

---

## Check --strict only fails on applies_to glob matches

- Status: accepted
- Scope: global
- Severity: must
- Applies to: internal/check/**, internal/cmd/remember.go
- Tags: check, strict

`archivist check --strict` and `Result.HasViolation` must be true only when a must/must-not rule's `applies_to` glob matches the supplied paths or diff paths. Semantic search hits are advisory and must not fail CI.

---

## Code map tests must fixture every registered language

- Status: accepted
- Scope: global
- Severity: must
- Applies to: internal/codemap/**
- Tags: codemap

`TestExtractEveryRegisteredLanguage` must include a sample for every key in `languageSpecs` and for `.gleam`. A new tree-sitter or line-based language is not done until that table has a fixture that extracts at least one expected symbol.

---

## Do not store session notes as accepted archive records

- Status: accepted
- Scope: global
- Severity: should-not
- Applies to: docs/decisions/**, docs/global-decisions/**
- Tags: hygiene

Ephemeral session observations (MCP down this chat, config currently unset, unverified "gap" catalogs), conversation transcripts, and restatements of records already on file are not accepted archive records. Distill a lasting choice, constraint, or capability. Prefer `update` or `retire` over adding a parallel accepted note.

---

## Embed --once must process every queued item

- Status: accepted
- Scope: global
- Severity: must
- Applies to: internal/embed/**, internal/store/**
- Tags: embed, queue

`archivist embed --worker --once` must process every row currently in `embed_queue` (repo and home). It must not stop after a batch size, and it must not skip remaining items because one sibling failed. Queue rows with no matching record must be deleted, not counted as successfully embedded.

---

## Never pass unsanitized text to FTS5 MATCH

- Status: accepted
- Scope: global
- Severity: must
- Applies to: internal/store/**, internal/retrieve/**
- Tags: search, fts

User and agent strings go through `fts5Query` (or equivalent) before `records_fts MATCH`. Punctuation and FTS5 operators in the input must not produce a syntax error. If MATCH still fails with an FTS5 syntax error, return no hits instead of failing the search.

---

