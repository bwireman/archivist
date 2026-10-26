# Records, commands, and export

SQLite is the source of truth. `remember`, `update`, and `retire` write the databases only. `archivist index` updates the code map and git metadata; it does not ingest or delete records. Missing markdown does not wipe the archive.

`archivist import` upserts typed markdown into SQLite and never deletes DB-only rows. `archivist export` writes `docs/archive/` only when `records.write_docs` is true.

## Record model

Records are typed rows in SQLite (FTS + optional vectors). `source_path` is a unique logical key that looks like a markdown path; the file does not have to exist. Markdown uses the same YAML front matter when you `import` or `export`:

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
- **scope**: `repo` lives in the checkout's `.archivist/index.db` (this checkout, or a listed extra when the record is about that checkout); `global` and `dev` live in `~/.archivist/archive.db`. Logical `source_path` prefixes follow `records.repo` (default `docs/decisions`), `records.global` (default `~/.archivist`), and `records.dev` (default `~/.archivist/records`). Those directories are import drop folders, not required files.
- **severity** (rules): `must`, `must-not`, `should`, `should-not`

Use `--type feature` (or MCP `search` with `type=feature`) when looking up how a subsystem behaves. Encode a design choice as a `decision`, a constraint as a `rule`.

## Commands

| Command | Ollama | Purpose |
| --- | --- | --- |
| `archivist init` | no | Config, checkout and home SQLite databases, optional import drop folders. Query commands do not create a database; a missing one says to run init. |
| `archivist import` | no | Upsert typed markdown into SQLite (no prune) |
| `archivist index` | no | Index this checkout's code map + git history (feeds `map`). Does not walk `archives`. |
| `archivist embed` | yes | Drain embed queue (`--once` processes every item once, then exits) |
| `archivist search <query>` | optional | Hybrid FTS + vector search, including listed `archives` (`--type feature` for capability docs; `--status` to include a retired status; `--json` for `{results, links}`; `--full` with `--json` for every stored field of each ranked hit). `links` follows `record`, `pin`, and `repo` edges. A linked checkout does not need to be listed in `archives`. Source files named by `applies_to` are included, at most 4 files and 2KiB each, and 24KiB for the response. Opens existing databases read-only. |
| `archivist link` | no | Store a `record`, `pin`, or `repo` link in this checkout (`--kind`, `--from`, `--to`). `--to` is `checkout#id` for `record` and `pin`, and `checkout` for `repo`. |
| `archivist unlink` | no | Remove one link. Same flags as `archivist link`. A missing link succeeds. |
| `archivist link list` | no | List this checkout's record links (`--json`). |
| `archivist map <query>` | no | Explore the code map: symbols, imports, importers, commits. Also reads listed `archives`; does not index them. |
| `archivist check` | optional | Match rules to a change |
| `archivist cite <id>` | no | Record that a retrieved record changed the work (requires `log_commands`) |
| `archivist trace` | no | Digest `commands.log`. `--since` joins consulted records to `git diff` of the work tree versus that rev (staged and unstaged tracked changes included; untracked files omitted). `--until` is an inclusive RFC3339 end for log lines and does not change those paths. Without `--until`, later log lines stay in the digest. |
| `archivist review` | no | List missed rules, close records, stale `applies_to`, and features to compare. Does not update, retire, or rank. `--due` reports whether a full review is due (every 7 days) and does not count as a review. |
| `archivist remember` | no | Create a record in SQLite (no markdown file). `--archive` names a listed checkout when the repo record is about that checkout. |
| `archivist update` / `retire` | no | Amend or supersede |
| `archivist export` | no | Generate `docs/archive/` (no-op unless `records.write_docs`) |
| `archivist publish <name>` | no | Bundle + configured shell command |
| `archivist mcp` | optional | MCP server (primary agent API) |
| `archivist skills install --target cursor` | no | Always-on rules + on-demand skills |
| `archivist status` | no | Archive + queue status. Listed archives are counted separately from this checkout. |

## Optional markdown export

`archivist export` writes `records.export` (default `docs/archive/`) only when `records.write_docs` is true. `--bundle` and `publish` still write a portable tree when the flag is off. Sharing a checkout’s archive via git means committing markdown or a bundle, then running `archivist import` on clone — not committing SQLite.

When enabled:

- `docs/archive/INDEX.md` — catalog by type, with links to type digests
- `docs/archive/rules.md`, `decisions.md`, `features.md`, … — full-text digest per type (`maps.md` for map-type records so it does not collide with the code map)
- `docs/archive/map.md` — code structure overview
- `docs/archive/records/<scope>/<type>/<slug>.md` — one file per record
- `docs/archive/archive.json` — machine-readable manifest

Do not hand-edit `docs/archive/`; regenerate with `archivist export`. Run `archivist import` after cloning markdown or an export tree to hydrate SQLite.
