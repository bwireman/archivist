# Consult the archive

Before implementing, researching, or changing architecture, look things up in this order. Do not guess APIs, defaults, or past decisions. When the user recalls a past choice or constraint, search the archive; do not trust chat memory.

1. Archivist MCP (preferred) or CLI: `archivist search "..."`, `archivist search --type feature "..."`, `archivist check --paths ...`
2. Generated tree when MCP is unavailable: `docs/archive/INDEX.md`, then the type digest files (`docs/archive/rules.md`, `features.md`, `decisions.md`, …) and copies under `docs/archive/records/<scope>/<type>/`, then the configured `records.repo` directory and `records.global` (default `~/.archivist`)
3. Code last

When changing a subsystem, read the `feature` record for that capability (how it works) as well as related `decision` and `rule` records. `archivist search` uses hybrid FTS + vectors and degrades to keyword-only when Ollama is down.
