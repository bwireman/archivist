# Rules

## Append gitignore entries on their own line

- Severity: must
- Applies to: internal/cmd/**

When appending a line to `.gitignore`, insert a newline first if the file does not already end with one. Otherwise `init` glues `.archivist/` onto the last existing entry and the ignore never matches.

---

## Bump codemap.Version when extractors change

- Severity: must
- Applies to: internal/codemap/**

When adding or changing a code-map extractor (tree-sitter specs, line-based languages, or the generic fallback), increment `codemap.Version`. Index skips unchanged files by content hash; a stale `codemap_version` in meta is what forces a remap of existing trees.

---

## Code map tests must fixture every registered language

- Severity: must
- Applies to: internal/codemap/**

`TestExtractEveryRegisteredLanguage` must include a sample for every key in `languageSpecs` and for `.gleam`. A new tree-sitter or line-based language is not done until that table has a fixture that extracts at least one expected symbol.

---

## Do not store session notes as accepted archive records

- Severity: should-not
- Applies to: docs/decisions/**, docs/global-decisions/**

Ephemeral session observations (MCP down this chat, config currently unset, unverified "gap" catalogs) are not accepted decisions. Record a decision or rule only when choosing between alternatives or encoding a lasting constraint. Prefer `update` or `retire` over adding a parallel accepted note.

---

## Never pass unsanitized text to FTS5 MATCH

- Severity: must
- Applies to: internal/store/**, internal/retrieve/**

User and agent strings go through `fts5Query` (or equivalent) before `records_fts MATCH`. Punctuation and FTS5 operators in the input must not produce a syntax error. If MATCH still fails with an FTS5 syntax error, return no hits instead of failing the search.

---

