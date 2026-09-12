# Consult the archive

Before implementing, researching, or changing architecture, look things up in this order. Do not guess APIs, defaults, or past decisions.

1. Archivist MCP (preferred) or CLI: `archivist search "..."`, `archivist check --paths ...`
2. Generated tree when MCP is unavailable: `docs/archive/INDEX.md`, `docs/archive/rules.md`, then the configured `records.repo` directory and `records.global` (default `~/.archivist`)
3. Code last

`archivist search` uses hybrid FTS + vectors and degrades to keyword-only when Ollama is down.
