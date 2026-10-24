# Configuration

`.archivist.json` keys:

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
  "log_commands": true,
  "archives": ["../other-repo", "~/src/another-repo"]
}
```

- Empty `ollama.base_url` uses `$OLLAMA_HOST` (host:port or a full URL), then `http://localhost:11434`. A value already set in the file wins, including an explicit localhost URL. `config.Load` leaves an empty `base_url` empty so that fallback runs. Omit the key for the same result.
- Optional `ollama.embed_num_ctx` sets Ollama `options.num_ctx` on embed requests (worker and search). Omit it to use the model’s loaded context window.
- Empty `records.dev` is `~/.archivist/records`.
- Empty `records.global` is `~/.archivist` (import walk for top-level `.md`, skip `records/` and `archive.db`). Set a checkout-relative directory (this repo uses `docs/global-decisions`) if you want an in-repo import drop folder for product-wide records. SQLite remains canonical; committing markdown is optional.
- `records.write_docs` (default false) controls whether `archivist export` writes `records.export`. `--bundle` and publish ignore the flag.
- SQLite paths are not configurable: `.archivist/index.db` and `~/.archivist/archive.db`. `archives` lists checkout roots, not database files.
- `log_commands` (default false) appends JSONL lines to `.archivist/commands.log` for archive CLI commands and MCP tools: one `dir=in` line with arguments, one `dir=out` line with the parsed result or error (CLI and MCP, clipped at 64KiB). The two lines share an `id`. Trace matches on that id. Lines with no `id` still pair by command name, oldest unmatched `in` first. `init`, `version`, `skills`, `trace`, and the `mcp` process itself are not logged (MCP tools still are, including `cite`). Logging never fails the command. If `.archivist/` is not writable, the append is skipped. `search`, `map`, `status`, `check`, `cite`, and `trace` open existing checkout and home databases read-only and do not create schema or purge orphans. `link list` opens this checkout read-only. MCP `links` reads. `import`, `index`, `embed`, `export`, `publish`, `remember`, `update`, and `retire` open them writable and purge orphan queue and vector rows. link and unlink open the checkout database writable and create `record_links` when it is missing. `archivist cite` errors when the flag is off. `archivist trace` reads the log and, with `--since`, reports consulted records (get, cite, check, and hybrid or keyword search hits) whose `applies_to` overlaps the diff. Vector-only search hits are listed and left out of that join. `--since` is a git rev or an RFC3339 time. Paths are the work tree versus that rev, including later commits and uncommitted tracked changes, or commits after that time plus uncommitted tracked changes when the value is a time. Untracked files are omitted. `--until` is an inclusive RFC3339 end for log lines (`after <= timestamp <= until`) and does not change paths. Without `--until`, the report warns that later log lines are included. A revision range is rejected. A rev that is not an ancestor of HEAD is still diffed and called out. `--all` can be combined with `--until` and cannot be combined with `--since`.
- `.gitignore` is always honored. `.git` and `.archivist` are always skipped.

## Extra archives

Put `archives` in the `.archivist.json` of the checkout the MCP process starts in (its cwd, or `--path`). Each entry is another checkout's root. Archivist opens `<root>/.archivist/index.db` there.

```json
"archives": ["../other-repo", "/absolute/path/to/another", "~/src/third"]
```

- A relative path is resolved from that checkout, not from a later working directory. `~/…` and absolute paths are accepted.
- Run `archivist init` in the other checkout first. The database file must already exist. Archivist does not create it.
- The list cannot name this checkout, repeat a path, or point at a file. A listed checkout's own `archives` list is not followed.
- `search`, `get`, `cite`, `status`, and `map` read those databases. An extra search hit includes `archive` (that checkout's root). `map` keeps this checkout at the top level and adds each listed checkout under `archives`; that field is omitted when the list is empty. `map` does not run `index` on them. To fill an extra code map, run `archivist --path <that-checkout> index` there.
- Record links are a separate read. `search` and `get` also open a checkout named by a link in this checkout, read-only, even when that checkout is not listed in `archives`. The `archives` list is unchanged. Search does not include source files. Nothing is copied between databases.
- `check`, `index`, `import`, and `embed` stay on this checkout, including the MCP `index` and `embed` tools. Global and dev records stay in `~/.archivist/archive.db`.
- A new repo record about this checkout omits `--archive` (MCP `archive`). A new repo record about a listed checkout passes that same path and is inserted only there. `update` and `retire` change the database that already holds the id. Nothing is copied between archives.

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
