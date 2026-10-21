---
name: refresh-archive
description: Import markdown into SQLite if needed, then rebuild the code map, embeddings, and (when records.write_docs is true) the generated docs/archive tree. Use after changing records or code structure, or when the user asks to reindex or export the archive.
---

# Refresh the archive

Call the MCP tools. Use the CLI only when MCP is unavailable.

1. `import` (or `archivist import`). Upserts typed markdown into SQLite. Optional when no record markdown is on disk. Never deletes DB-only records.
2. `index` (or `archivist index`). Updates the code map and git metadata. No Ollama.
3. `embed` with `once` true (or `archivist embed --once`). Drains the queue. Needs Ollama; skip if it is down.
4. `archivist export`. There is no MCP export tool. Writes `records.export` (default `docs/archive/`) only when `records.write_docs` is true; otherwise it is a no-op. `--bundle` and publish still write. Do not hand-edit that tree.

- After changing how a capability behaves, update its `feature` record (or create one with `record-feature`) before this refresh.
- Skip the whole sequence if you only touched `docs/archive/` or `.archivist/`.
- The always-on current rule calls MCP `index` as source changes. This sequence still runs at the end of the turn.
