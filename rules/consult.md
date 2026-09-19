# Consult the archive

Before implementing, researching, or changing architecture, look things up in this order. Do not guess APIs, defaults, or past decisions. When the user recalls a past choice or constraint, search the archive; do not trust chat memory.

1. Archivist MCP (preferred) or CLI: `archivist search "..."`, `archivist search --type feature "..."`, `archivist check --paths ...`. SQLite is the live archive.
2. Generated export tree when MCP is unavailable and `records.write_docs` is true: `docs/archive/INDEX.md`, then type digests (`rules.md`, `features.md`, …) and `docs/archive/records/<scope>/<type>/`. To load markdown from disk into SQLite, run `archivist import` (optional on clone; never deletes DB-only records).
3. Code last

When changing a subsystem, read the `feature` record for that capability (how it works) as well as related `decision` and `rule` records. `archivist search` uses hybrid FTS + vectors and degrades to keyword-only when Ollama is down.
