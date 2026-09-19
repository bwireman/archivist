---
name: refresh-archive
description: Import markdown into SQLite if needed, then rebuild the code map, embeddings, and (when records.write_docs is true) the generated docs/archive tree. Use after changing records or code structure, or when the user asks to reindex or export the archive.
---

# Refresh the archive

```bash
archivist import
archivist index
archivist embed --worker --once
archivist export
```

- `import` upserts typed markdown into SQLite (optional when no record markdown on disk; never deletes DB-only records).
- `index` updates the code map and git metadata. No Ollama.
- `embed --worker --once` drains the queue. Needs Ollama; skip if it is down.
- `export` writes `records.export` (default `docs/archive/`) only when `records.write_docs` is true; otherwise it is a no-op. `--bundle` and publish still write. Do not hand-edit that tree.
- After changing how a capability behaves, update its `feature` record (or create one with `record-feature`) before this refresh.
- Skip the whole sequence if you only touched `docs/archive/` or `.archivist/`.
