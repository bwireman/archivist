---
name: publish-archive
description: Export an Archivist bundle and push it with a configured destination command. Use when the user asks to publish the archive to git, Confluence, or another destination.
---

# Publish the archive

1. In `.archivist.json`, set a named command. `{{bundle}}` is replaced with the export directory:

```json
"publish": {
  "destinations": {
    "team-wiki": { "command": ["./scripts/push.sh", "{{bundle}}"] }
  }
}
```

2. Run:

```bash
archivist publish team-wiki
```

Core only writes the bundle (records + `archive.json`). The destination command does the rest. Archivist does not speak Confluence or git remotes itself.
