# Refresh the archive

After changing records or code structure:

1. `archivist index --plain` — index records and code map (no Ollama).
2. `archivist embed --worker --once` — embed queued records (needs Ollama).
3. `archivist export` — regenerate `docs/archive/`.
