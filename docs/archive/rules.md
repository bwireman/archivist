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

