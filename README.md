# Archivist

A local knowledge archive for design decisions, rules, features, guides, and code structure. **SQLite is the source of truth** for records (`.archivist/index.db` for this checkout, `~/.archivist/archive.db` for global and personal notes). MCP is the primary agent surface. Markdown is optional: `archivist import` upserts typed files into SQLite; `archivist export` writes `docs/archive/` only when `records.write_docs` is true.

`remember` / `update` / `retire` write the databases only. `archivist index` updates the code map and git metadata; it does not ingest or delete records. Missing markdown does not wipe the archive.

Archivist depends only on SQLite and Ollama HTTP — no vendor SDKs.

## Requirements

- [Go](https://go.dev) 1.27+ to build or install the CLI
- [Ollama](https://ollama.com) for embedding (optional for `index` / `search`; required for `embed`)

## Install the CLI

Each successful commit to `main` publishes a GitHub pre-release. This downloads
the newest one, checks its SHA-256, and installs `archivist` (Linux amd64 or
Apple Silicon):

```bash
curl -fsSL https://raw.githubusercontent.com/bwireman/archivist/main/scripts/install.sh | bash
archivist version
```

The binary is written to `~/.local/bin`. Override the directory with
`ARCHIVIST_INSTALL_DIR`, or pin a tag with `ARCHIVIST_TAG=v0.1.0-<sha>`.

From this checkout, with Go 1.27+:

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

   That writes `.archivist.json`, creates `docs/decisions/` as an optional **import drop folder** (not required for `remember`), appends `.archivist/` to `.gitignore`, and uses `~/.archivist/archive.db` for global and dev records (plus `~/.archivist/records/` as a drop folder for dev-scoped markdown). It does not create `docs/archive/` unless `records.write_docs` is true.

3. **Import (if you have markdown), index, embed, export:**

   ```bash
   archivist import               # optional; upsert markdown into SQLite
   archivist index                # code map + git history; no Ollama
   archivist embed --once         # skip if Ollama is down
   archivist export               # no-op unless records.write_docs is true
   ```

4. **Install agent rules/skills** (optional):

   ```bash
   archivist skills install --target cursor    # or claude | agents-md | copilot
   ```

5. **Seed the archive** with the `init-archive` skill after rules are installed. It scans the repo for decisions, rules, features, and the other record types, writes the gaps, and embeds them so search is useful before new work. `archivist init` only creates the empty store.

6. **Point an MCP client at this repo** (next section), then `archivist status` to confirm.

From another directory, pass `--path`:

```bash
archivist --path /path/to/your/repo status
```

## MCP

`archivist mcp` is the primary query surface. It speaks MCP over **stdio** (JSON-RPC on stdin/stdout).

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
| `search` | Hybrid search (`query`, optional `type` such as `feature`, `scope`, `status`, `top_k`, `full`). Default omits deprecated and superseded, and omits the body. Pass `full` for every stored field. Use before implementing or writing a record. |
| `get` | One record by id or slug. Default returns id, slug, type, scope, title, status, severity, body, tags, applies_to, and superseded_by. Pass `full` for every stored field. |
| `check` | Rules for a change (`description`, `paths`, `diff`) |
| `map` | Explore the code map: matching symbols and files, the imports those files declare, the files importing the query, and recent commits mentioning it (`query`, optional `limit`) |
| `remember` | Create a record in SQLite after search shows a gap (`type` is `decision`, `rule`, `feature`, `guide`, `map`, or `pitfall`). Distill lasting facts; do not dump chat. No markdown file. |
| `update` | Amend title, body, or status in place (prefer over a parallel `remember`) |
| `retire` | Mark superseded when a later choice replaces it |
| `cite` | Optional. Record that a retrieved record changed the work (`id`, one-line `effect`). Requires `log_commands`. |
| `import` | Upsert typed markdown into SQLite (no prune) |
| `status` | Counts, embed queue, Ollama health |

### Prompts

`prompts/list` walks through adding one record. Each prompt takes `topic` (required) and optional `scope` (`repo`, `global`, or `dev`). The prompt does not write. It tells the client to call `search` without a scope, update a current record and keep its scope, then `remember` only when search shows a gap. The optional `scope` is a suggestion the procedure can override: `dev` is about the person or this machine, `repo` is true only in this checkout, and `global` is true for the product in every checkout.

| Prompt | Record |
| --- | --- |
| `add-decision` | Choice among alternatives (Context, Decision, Consequences) |
| `add-rule` | Must or should, with `severity` and `applies_to` |
| `add-feature` | How a capability works (Purpose, Behavior, Connects to, Entry points) |
| `add-guide` | How-to procedure |
| `add-map` | Packages and how they connect |
| `add-pitfall` | Confirmed gotcha |

Initialize `instructions` are the consult, ask, and record rule templates, so MCP-only hosts still look things up, ask when a choice or preference is unsettled, and distill from conversation.

Tool output is JSON, the same shape as CLI `--json`. The default is a short card: search omits the body, and get keeps the body while dropping hash, timestamps, provenance, and source path. `full` and `--full` restore every stored field. Without MCP, use `archivist search`, `archivist cite`, and `archivist trace`. The generated `docs/archive/` tree is an optional export when `records.write_docs` is true, not the live archive.

## Record model

Records are typed rows in SQLite (FTS + optional vectors). `remember` / `update` / `retire` write the DB only. `source_path` is a unique logical key that looks like a markdown path; the file does not have to exist. Markdown uses the same YAML front matter when you `import` or `export`:

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

- **type**:
  - `decision` — a choice among alternatives (Context, Decision, Consequences)
  - `rule` — a must/must-not or should/should-not constraint; set `applies_to` so `archivist check` can match touched files
  - `feature` — living docs of a capability: how it works, what it connects to, how to invoke it (Purpose, Behavior, Connects to, Entry points)
  - `guide` — a how-to procedure
  - `map` — structural notes (`docs/archive/map.md` is the generated code map)
  - `pitfall` — a confirmed gotcha
- **scope**: `repo` lives in `.archivist/index.db`; `global` and `dev` live in `~/.archivist/archive.db`. Logical `source_path` prefixes follow `records.repo` (default `docs/decisions`), `records.global` (default `~/.archivist`), and `records.dev` (default `~/.archivist/records`). Those directories are import drop folders, not required files.
- **severity** (rules): `must`, `must-not`, `should`, `should-not`

Use `--type feature` (or MCP `search` with `type=feature`) when looking up how a subsystem behaves. Encode a design choice as a `decision`, a constraint as a `rule`.

## Commands

| Command | Ollama | Purpose |
| --- | --- | --- |
| `archivist init` | no | Config, SQLite dirs, optional import drop folders |
| `archivist import` | no | Upsert typed markdown into SQLite (no prune) |
| `archivist index` | no | Index code map + git history (feeds `map`) |
| `archivist embed` | yes | Drain embed queue (`--once` processes every item once, then exits) |
| `archivist search <query>` | optional | Hybrid FTS + vector search (`--type feature` for capability docs; `--status` to include a retired status; `--json` for the search card; `--full` with `--json` for every stored field) |
| `archivist map <query>` | no | Explore the code map: symbols, imports, importers, commits |
| `archivist check` | optional | Match rules to a change |
| `archivist cite <id>` | no | Record that a retrieved record changed the work (requires `log_commands`) |
| `archivist trace` | no | Digest `commands.log`; `--since` joins consulted records to a git diff |
| `archivist remember` | no | Create a record in SQLite (no markdown file) |
| `archivist update` / `retire` | no | Amend or supersede |
| `archivist export` | no | Generate `docs/archive/` (no-op unless `records.write_docs`) |
| `archivist publish <name>` | no | Bundle + configured shell command |
| `archivist mcp` | optional | MCP server (primary agent API) |
| `archivist skills install --target cursor` | no | Always-on rules + on-demand skills |
| `archivist status` | no | Archive + queue status |

## Optional markdown export

`archivist export` writes `records.export` (default `docs/archive/`) only when `records.write_docs` is true. `--bundle` and `publish` still write a portable tree when the flag is off. Sharing a checkout’s archive via git means committing markdown or a bundle, then running `archivist import` on clone — not committing SQLite.

When enabled:

- `docs/archive/INDEX.md` — catalog by type, with links to type digests
- `docs/archive/rules.md`, `decisions.md`, `features.md`, … — full-text digest per type (`maps.md` for map-type records so it does not collide with the code map)
- `docs/archive/map.md` — code structure overview
- `docs/archive/records/<scope>/<type>/<slug>.md` — one file per record
- `docs/archive/archive.json` — machine-readable manifest

Do not hand-edit `docs/archive/`; regenerate with `archivist export`. Run `archivist import` after cloning markdown or an export tree to hydrate SQLite.

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
    "dev": "",
    "export": "docs/archive",
    "write_docs": false
  },
  "publish": {
    "destinations": {
      "team-wiki": { "command": ["./scripts/push.sh", "{{bundle}}"] }
    }
  },
  "log_commands": true
}
```

- Empty `ollama.base_url` uses `$OLLAMA_HOST` (scheme optional) or `http://localhost:11434`.
- Optional `ollama.embed_num_ctx` sets Ollama `options.num_ctx` on embed requests (worker and search). Omit it to use the model’s loaded context window.
- Empty `records.dev` is `~/.archivist/records`.
- Empty `records.global` is `~/.archivist` (import walk for top-level `.md`, skip `records/` and `archive.db`). Set a checkout-relative directory (this repo uses `docs/global-decisions`) if you want an in-repo import drop folder for product-wide records. SQLite remains canonical; committing markdown is optional.
- `records.write_docs` (default false) controls whether `archivist export` writes `records.export`. `--bundle` and publish ignore the flag.
- SQLite paths are not configurable: `.archivist/index.db` and `~/.archivist/archive.db`.
- `log_commands` (default false) appends JSONL lines to `.archivist/commands.log` for archive CLI commands and MCP tools: one `dir=in` line with arguments, one `dir=out` line with the parsed result or error (CLI and MCP, clipped at 64KiB). `init`, `version`, `skills`, `trace`, and the `mcp` process itself are not logged (MCP tools still are, including `cite`). Logging never fails the command. `archivist cite` errors when the flag is off. `archivist trace` reads the log and, with `--since`, reports consulted records (get, cite, check, and hybrid or keyword search hits) whose `applies_to` overlaps the diff. Vector-only search hits are listed and left out of that join.
- `.gitignore` is always honored. `.git` and `.archivist` are always skipped.

## Agent rules and skills

`archivist skills install --target cursor|claude|agents-md|copilot` (`codex` is an alias for `agents-md`) writes:

- **Rules** (always on): consult the archive, cite a retrieved record that changed the work before the turn ends, ask when a user-facing choice or preference is unsettled, distill lasting decisions/rules/features from the conversation (skip chat glut), keep the code map current as source changes (`archivist index`), refresh after changes. Each rule is its own file (`rules/consult.md`, `rules/cite.md`, `rules/ask.md`, `rules/record.md`, `rules/current.md`, `rules/refresh.md`). Cursor: `.cursor/rules/archivist-*.mdc`. `agents-md` / `copilot` get a single concatenated file only.
- **Skills** (on demand): `init-archive`, `plan-changes`, `record-decision`, `record-rule`, `record-feature`, `refresh-archive`, `publish-archive`. Write skills are the per-type procedure (search first, short body); the record rule is when to write. `init-archive` scans the repo and embeds the gaps so a new archive is searchable. `plan-changes` searches the archive, writes `.archivist/plans/<slug>.md`, runs an adversarial review, and stops before phase 1. A later accept only sets the plan accepted. A later continue implements the first pending phase. Cursor files come from `archivist skills install --target cursor`. Claude files come from `archivist skills install --target claude`. `agents-md` and Copilot stay rules-only.

Templates live in `rules/` and `skills/` in this repo and are embedded in the CLI. `skills install` uses those shipped templates, so it works in any repo; if the target checkout has its own `rules/` or `skills/`, those override the embedded copies.

Every install also writes `.archivist-install.json` with the CLI version, the binary’s git commit (ldflags `Commit`, otherwise Go’s `vcs.revision`), the host this run installed, and SHA-256 hashes of the rules and skills that run wrote. Cursor hashes the wrapped `.mdc`; other hosts hash the trimmed rule body. `agents-md` and `copilot` record no skill hashes. `--target codex` is an alias for `agents-md`.

Every command compares those installed rule files with the templates this binary would write (a checkout-local `rules/` directory still overrides the embedded copies). When they differ, or the manifest cannot be read, archivist prints a reminder to run `archivist skills install --target <host>` on stderr and continues. No manifest means rules were never installed, so it stays quiet.

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
make import           # upsert typed markdown into SQLite
make index            # code map + git
make embed            # needs Ollama
make export           # no-op unless records.write_docs
make refresh-archive  # import, index, embed, export
```

## CI archives

Pull requests and pushes to `main` run the Go test suite and `go vet`. Each
successful commit to `main` publishes a GitHub **pre-release** tagged
`v<version>-<short-sha>` with a source archive plus compiled CLI binaries for
Linux amd64 and Apple Silicon macOS. `scripts/install.sh` installs the newest
of those releases. Every file includes a SHA-256 checksum;
`build-info.txt` (and per-binary `*.build-info.txt`) record the CLI version and
full commit SHA.
