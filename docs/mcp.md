# MCP

`archivist mcp` is the primary query surface. It speaks MCP over stdio (JSON-RPC on stdin/stdout).

The server opens the repo and home SQLite files, so the process cwd must be the repo, or you must pass `--path`. Search still works if Ollama is down (keyword-only). Writes (`remember`, `update`, `retire`) never need Ollama.

## Cursor

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

## Claude Desktop and other clients

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

## Several checkouts

Claude Code keeps one MCP process on the checkout it started in. List the other checkouts in that checkout's `.archivist.json` under `archives`. See [Extra archives](configuration.md#extra-archives) for the path rules and which commands read or write them.

## Tools

| Tool | Purpose |
| --- | --- |
| `search` | Hybrid search (`query`, optional `type` such as `feature`, `status`, `top_k`, `full`). Search reads repo, global, and dev and does not take a scope. Default omits deprecated and superseded, and omits the body. Pass `full` for every stored field. Use before implementing or writing a record. |
| `get` | One record by id or slug. Default returns id, slug, type, scope, title, status, severity, body, tags, applies_to, and superseded_by. Pass `full` for every stored field. |
| `check` | Rules for a change (`description`, `paths`, `diff`) |
| `map` | Explore the code map: matching symbols and files, the imports those files declare, the files importing the query, and recent commits mentioning it (`query`, optional `limit`). The top level is this checkout. Listed `archives` are returned under `archives` and are not indexed by this call. |
| `remember` | Create a record in SQLite after search shows a gap (`type` is `decision`, `rule`, `feature`, `guide`, `map`, or `pitfall`). Optional `archive` is the listed checkout a repo record is about; omit it when that context is the process checkout. Distill lasting facts; do not dump chat. No markdown file. |
| `update` | Amend title, body, or status in place (prefer over a parallel `remember`) |
| `retire` | Mark superseded when a later choice replaces it |
| `cite` | Optional. Record that a retrieved record changed the work (`id`, one-line `effect`). Requires `log_commands`. |
| `import` | Upsert typed markdown into SQLite (no prune) |
| `status` | Counts, embed queue, Ollama health |

## Prompts

`prompts/list` walks through adding one record. Each prompt takes `topic` (required) and optional `scope` (`repo`, `global`, or `dev`). The prompt does not write. It tells the client to call `search` without a scope, update a current record and keep its scope, then `remember` only when search shows a gap. The optional `scope` is a suggestion the procedure can override: `dev` is about the person or this machine, `repo` is true only in the checkout the record is about, and `global` is true for the product in every checkout. Write a repo row to that checkout's database. The process checkout is the context when `archive` is empty. `archive` names a listed extra root when the record is about that checkout.

| Prompt | Record |
| --- | --- |
| `add-decision` | Choice among alternatives (Context, Decision, Consequences) |
| `add-rule` | Must or should, with `severity` and `applies_to` |
| `add-feature` | How a capability works (Purpose, Behavior, Connects to, Entry points) |
| `add-guide` | How-to procedure |
| `add-map` | Packages and how they connect |
| `add-pitfall` | Confirmed gotcha |

Initialize `instructions` are the consult, cite, ask, and record rule templates, so MCP-only hosts still look things up, cite a record that changed the work, ask when a choice or preference is unsettled, and write a short record when a later session would look it up.

Tool output is JSON, the same shape as CLI `--json`. The default is a short card: search omits the body, and get keeps the body while dropping hash, timestamps, provenance, and source path. `full` and `--full` restore every stored field. Without MCP, use `archivist search`, `archivist cite`, and `archivist trace`. The generated `docs/archive/` tree is an optional export when `records.write_docs` is true, not the live archive.
