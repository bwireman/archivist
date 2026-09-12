# Archivist

A local knowledge archive for design decisions, rules, guides, and code structure. Records live as markdown with front matter; SQLite indexes them for hybrid search; MCP is the primary agent surface. A generated `docs/archive/` tree serves humans and tools without MCP.

Archivist depends only on SQLite and Ollama HTTP — no vendor SDKs.

## Requirements

- [Go](https://go.dev) 1.27+ to build
- [Ollama](https://ollama.com) for embedding (optional at index time; required for `embed --worker`)

## Install

```bash
make install
archivist version
```

## Quick start

```bash
cd /path/to/your/repo
archivist init --plain
archivist index --plain          # no Ollama required
archivist embed --worker --once  # needs Ollama
archivist export                 # writes docs/archive/
archivist mcp                    # MCP server on stdio
```

## Record model

Records are markdown files with YAML front matter:

```markdown
---
id: rec_...
type: rule
scope: repo
status: accepted
title: Never call billing from handlers
severity: must-not
applies_to: ["internal/http/**"]
tags: [billing]
---

## Context
...
```

- **type**: `decision`, `rule`, `guide`, `map`, `pitfall`
- **scope**: `dev` (`~/.archivist/records/`), `repo` (`docs/decisions/`), `global` (`docs/global-decisions/`)
- **severity** (rules): `must`, `must-not`, `should`, `should-not`

## Commands

| Command | Ollama | Purpose |
| --- | --- | --- |
| `archivist init` | no | Config, data dirs, decision dirs |
| `archivist index` | no | Index records + code map |
| `archivist embed --worker` | yes | Drain embed queue |
| `archivist search <query>` | optional | Hybrid FTS + vector search |
| `archivist check` | optional | Match rules to a change |
| `archivist remember` | no | Create a record |
| `archivist update` / `retire` | no | Amend or supersede |
| `archivist export` | no | Generate `docs/archive/` |
| `archivist publish <name>` | no | Bundle + configured shell command |
| `archivist mcp` | optional | MCP server (primary agent API) |
| `archivist migrate records` | no | Convert legacy ADRs |
| `archivist skills install --target cursor` | no | Generate agent skill files |
| `archivist status` | no | Archive + queue status |

## MCP tools

`search`, `get`, `check`, `map`, `remember`, `update`, `retire`, `status`

## Generated archive

`archivist export` writes:

- `docs/archive/INDEX.md` — catalog by type and scope
- `docs/archive/rules.md` — must/must-not rules inline
- `docs/archive/map.md` — code structure overview
- `docs/archive/records/<scope>/<slug>.md` — one file per record
- `docs/archive/archive.json` — machine-readable manifest

Do not hand-edit `docs/archive/`; regenerate with `archivist export`.

## Publish destinations

Configure in `.archivist.json`:

```json
"publish": {
  "destinations": {
    "team-wiki": { "command": ["./scripts/push.sh", "{{bundle}}"] }
  }
}
```

## Makefile (this checkout)

```bash
make build
make test
make index
make embed      # needs Ollama
make export
make refresh-archive
```

## Migration from v2

```bash
archivist migrate records
rm -f .archivist/index.db ~/.archivist/global.db   # schema v3 reset
archivist index --plain
archivist embed --worker --once
archivist export
```
