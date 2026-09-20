# Refresh the archive

After a turn that changed records (via MCP/CLI), Go sources, or markdown you want in SQLite, run:

```bash
archivist import
archivist index
archivist embed --once
archivist export
```

SQLite is the source of truth for records; `remember`/`update`/`retire` write the DB only. `import` upserts typed markdown from `records.repo`, `records.global`, dev records, and export copies — it never prunes DB-only rows. Skip `import` when no markdown record dirs exist. `index` is code map + git only (no Ollama). Skip embed if Ollama is down. `export` is a no-op unless `records.write_docs` is true (`--bundle` and publish still write). Skip the whole sequence if you only touched `docs/archive/` or `.archivist/`.

If the turn changed how a capability behaves, update (or create) its `feature` record first.

## Reindex on commit (optional)

`archivist init` and `archivist skills install` write `.githooks/post-commit`, which runs `archivist index` after each commit (code map + git metadata only; no Ollama). Enable it once per clone:

```bash
git config core.hooksPath .githooks
```

The hook is a no-op when `.archivist.json` is missing or `archivist` is not on `PATH`. Index failures do not affect the commit. Agents still run the full refresh sequence above when they change records or code during a turn; the hook keeps the map current between commits.
