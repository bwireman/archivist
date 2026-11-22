# Configuration

`~/.archivist/config.json` is the settings file for every checkout. Keys:

```json
{
  "ollama": {
    "base_url": "",
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

- Empty `ollama.base_url` uses `$OLLAMA_HOST` (host:port or a full URL), then `http://localhost:11434`. A value already set in the file wins, including an explicit localhost URL. `config.Load` leaves an empty `base_url` empty so that fallback runs. Omit the key for the same result.
- Optional `ollama.embed_num_ctx` sets Ollama `options.num_ctx` on embed requests (worker and search). Omit it to use the model’s loaded context window.
- Optional `ollama.critic_model` names a small instruct model for an advisory on `remember`. Empty means off. It is not the embed model. The default config example omits it.
- A repo `.archivist.json` or `~/.archivist/checkouts/<sha256>/config.json` is merged once into `~/.archivist/config.json` and removed. A value already set is kept. `commands.log` and plans stay under that checkout directory.
- Relative `records` paths and `index.skip_globs` are shared. A relative path is resolved against the checkout in use.
- Empty `records.dev` is `~/.archivist/records`.
- Empty `records.global` is `~/.archivist` (import walk for top-level `.md`, skip `records/` and `archive.db`). A relative directory, such as `docs/global-decisions`, is resolved inside each checkout. SQLite remains canonical; committing markdown is optional.
- `records.write_docs` (default false) controls whether `archivist export` writes `records.export`. `--bundle` and publish ignore the flag.
- SQLite is `~/.archivist/archive.db` for every checkout. The path is not configurable.
- `log_commands` (default false) appends JSONL lines to `~/.archivist/checkouts/<sha256>/commands.log` for archive CLI commands and MCP tools: one `dir=in` line with arguments, one `dir=out` line with the parsed result or error (CLI and MCP, clipped at 64KiB). The two lines share an `id`. Trace matches on that id. Lines with no `id` still pair by command name, oldest unmatched `in` first. `init`, `version`, `skills`, `trace`, and the `mcp` process itself are not logged (MCP tools still are, including `cite`). `review` is logged so the 7-day clock can see a full review; `archivist review --due` is logged and does not reset that clock. Logging never fails the command. If `.archivist/` is not writable, the append is skipped. `search`, `map`, `status`, `check`, `cite`, `trace`, and `review` open `~/.archivist/archive.db` once, read-only, and do not create schema or purge orphans. `link list` opens that file read-only. MCP `links` reads. `import`, `index`, `embed`, `export`, `publish`, `remember`, `update`, and `retire` open that file writable and do not create it when it is missing. Only `init` creates schema and purges orphan queue and vector rows. link and unlink open that file writable. `archivist cite` errors when the flag is off. `archivist trace` reads the log and, with `--since`, reports consulted records (get, cite, check, and hybrid or keyword search hits) whose `applies_to` overlaps the diff. Vector-only search hits are listed and left out of that join. `--since` is a git rev or an RFC3339 time. Paths are the work tree versus that rev, including later commits and uncommitted tracked changes, or commits after that time plus uncommitted tracked changes when the value is a time. Untracked files are omitted. `--until` is an inclusive RFC3339 end for log lines (`after <= timestamp <= until`) and does not change paths. Without `--until`, the report warns that later log lines are included. A revision range is rejected. A rev that is not an ancestor of HEAD is still diffed and called out. `--all` can be combined with `--until` and cannot be combined with `--since`.
- `.gitignore` is always honored. `.git` and `.archivist` are always skipped.

## One database

SQLite for every checkout, plus global and dev, is `~/.archivist/archive.db`. A repo row has `checkout` set to that repo's canonical path. Global and dev rows use an empty checkout. Settings are `~/.archivist/config.json`. `commands.log` and plans live under `~/.archivist/checkouts/<sha256>/`.

Search, get, and status read that one database. A search hit includes `checkout` when the row has one.

## Publish destinations

`archivist publish <name>` writes a bundle to `.archivist/bundle` and runs the named command. `{{bundle}}` is replaced with that directory. The command pushes the bundle.

```json
"publish": {
  "destinations": {
    "team-wiki": { "command": ["./scripts/push.sh", "{{bundle}}"] }
  }
}
```

An unknown name or an empty command fails. `records.write_docs` does not apply to this bundle.
