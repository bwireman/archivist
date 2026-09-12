# Archivist

A local knowledge archive for design decisions, rules, guides, and code structure. Records live as markdown with front matter; SQLite indexes them for hybrid search; MCP is the primary agent surface. A generated `docs/archive/` tree serves humans and tools without MCP.

Archivist depends only on SQLite and Ollama HTTP — no vendor SDKs.

## Requirements

- [Go](https://go.dev) 1.27+ to build or install the CLI
- [Ollama](https://ollama.com) for embedding (optional for `index` / `search`; required for `embed --worker`)

## Install the CLI

There is no published binary yet. From this checkout:

```bash
git clone https://github.com/bwireman/archivist.git
cd archivist
make install
archivist version
```

`make install` runs `go install ./cmd/archivist` into `$(go env GOPATH)/bin` (or `GOBIN` if set). Put that directory on your `PATH`.

To run without installing:

```bash
make build          # ./archivist
./archivist version
```

## Set up a repository

1. **Optional — start Ollama** and pull the embed model (needed later for `embed` and hybrid search):

   ```bash
   ollama serve
   ollama pull qwen3-embedding:0.6b
   ```

   Empty `ollama.base_url` in config uses `$OLLAMA_HOST` (host:port or a full URL) then `http://localhost:11434`.

2. **Initialize** in the repo you want to archive:

   ```bash
   cd /path/to/your/repo
   archivist init
   ```

   That writes `.archivist.json`, creates record dirs (`docs/decisions/`, `docs/global-decisions/`, `docs/archive/`), appends `.archivist/` to `.gitignore`, and creates `~/.archivist/records/` for dev-scoped notes.

3. **Index, embed, export:**

   ```bash
   archivist index                  # records + code map; no Ollama
   archivist embed --worker --once  # skip if Ollama is down
   archivist export                 # writes docs/archive/
   ```

4. **Install agent rules/skills** (optional):

   ```bash
   archivist skills install --target cursor    # or claude | agents-md | copilot
   ```

5. **Point an MCP client at this repo** (next section), then `archivist status` to confirm.

From another directory, pass `--path`:

```bash
archivist --path /path/to/your/repo status
```

## MCP

`archivist mcp` is the primary query surface. It speaks MCP over **stdio** (JSON-RPC on stdin/stdout). It does not take a port; `--http` is not implemented.

The server opens the repo and home SQLite files, so the process cwd must be the repo, or you must pass `--path`. Search still works if Ollama is down (keyword-only). Writes (`remember`, `update`, `retire`) never need Ollama.

### Cursor

Project file `.cursor/mcp.json` (or a user-level MCP config):

```json
{
  "mcpServers": {
    "archivist": {
      "command": "archivist",
      "args": ["mcp"]
    }
  }
}
```

If the client does not start the process in the repo root:

```json
{
  "mcpServers": {
    "archivist": {
      "command": "archivist",
      "args": ["--path", "/absolute/path/to/your/repo", "mcp"]
    }
  }
}
```

`archivist` must be on the `PATH` the GUI app sees (not only your interactive shell). `--path` is a persistent flag, so `archivist mcp --path /abs/repo` works too.

### Claude Desktop and other clients

Same stdio command. In Claude Desktop (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "archivist": {
      "command": "archivist",
      "args": ["--path", "/absolute/path/to/your/repo", "mcp"]
    }
  }
}
```

Any client that can spawn a process can use `archivist mcp` the same way.

### Tools

| Tool | Purpose |
| --- | --- |
| `search` | Hybrid search (`query`, optional `type`, `scope`, `top_k`) |
| `get` | One record by id or slug |
| `check` | Rules for a change (`description`, `paths`, `diff`) |
| `map` | Where code lives (symbols / files) |
| `remember` | Create a record (`type`, `scope`, `title`, `body`, …) |
| `update` | Amend title, body, or status |
| `retire` | Mark superseded |
| `status` | Counts, embed queue, Ollama health |

Tool output is JSON, the same shape as CLI `--json`. Agents without MCP should read `docs/archive/` instead.

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
| `archivist skills install --target cursor` | no | Always-on rules + on-demand skills |
| `archivist status` | no | Archive + queue status |

## Generated archive

`archivist export` writes:

- `docs/archive/INDEX.md` — catalog by type and scope
- `docs/archive/rules.md` — must/must-not rules inline
- `docs/archive/map.md` — code structure overview
- `docs/archive/records/<scope>/<slug>.md` — one file per record
- `docs/archive/archive.json` — machine-readable manifest

Do not hand-edit `docs/archive/`; regenerate with `archivist export`.

## Configuration

`.archivist.json` keys:

```json
{
  "ollama": {
    "base_url": "http://localhost:11434",
    "embed_model": "qwen3-embedding:0.6b",
    "embed_timeout": "5m"
  },
  "index": {
    "skip_globs": ["*.pb.go"]
  },
  "records": {
    "repo": "docs/decisions",
    "global": "docs/global-decisions",
    "dev": "",
    "export": "docs/archive"
  },
  "publish": {
    "destinations": {
      "team-wiki": { "command": ["./scripts/push.sh", "{{bundle}}"] }
    }
  }
}
```

- Empty `ollama.base_url` uses `$OLLAMA_HOST` (scheme optional) or `http://localhost:11434`.
- Empty `records.dev` is `~/.archivist/records`.
- SQLite paths are not configurable: `.archivist/index.db` and `~/.archivist/archive.db`.
- `.gitignore` is always honored. `.git` and `.archivist` are always skipped.

## Agent rules and skills

`archivist skills install --target cursor|claude|agents-md|copilot` writes:

- **Rules** (always on): consult the archive, record decisions, refresh after changes. Cursor: `.cursor/rules/archivist-*.mdc`. `agents-md` / `copilot` get a single concatenated file only.
- **Skills** (on demand): `record-decision`, `record-rule`, `refresh-archive`, `publish-archive`. Cursor: `.cursor/skills/<name>/SKILL.md`. Claude: `.claude/skills/`.

Templates live in `rules/` and `skills/` in this repo.

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
make install    # go install into GOPATH/bin
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
archivist index
archivist embed --worker --once
archivist export
```
