---
name: refresh-archive
description: Rebuild the Archivist index, embeddings, and generated docs/archive tree. Use after changing records or code structure, or when the user asks to reindex or export the archive.
---

# Refresh the archive

```bash
archivist index
archivist embed --worker --once
archivist export
```

- `index` writes records and the code map. No Ollama.
- `embed --worker --once` drains the queue. Needs Ollama; skip if it is down.
- `export` regenerates `records.export` (default `docs/archive/`). Do not hand-edit that tree.
- After changing how a capability behaves, update its `feature` record (or create one with `record-feature`) before this refresh.
- Skip the whole sequence if you only changed `docs/archive/` or `.archivist/`.
