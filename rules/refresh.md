# Refresh the archive

After a turn that changed records, Go sources, or docs under the record directories, run:

```bash
archivist index
archivist embed --worker --once
archivist export
```

Index does not need Ollama. Skip embed if Ollama is down. Skip all of this if you only touched `docs/archive/` or `.archivist/`.
