# Archivist

A local archive of design decisions, rules, features, guides, and code structure. Records live in SQLite (`.archivist/index.db` in the checkout, `~/.archivist/archive.db` for global and personal notes). Agents use MCP. Ollama embeds records so search can mix keywords and vectors.

## Requirements

- [Go](https://go.dev) 1.27+ to build the CLI
- [Ollama](https://ollama.com) for `embed` and hybrid search. `index` and keyword search work without it.

## Install

Each commit to `main` publishes a GitHub pre-release. This installs the newest Linux amd64 or Apple Silicon binary and checks its SHA-256:

```bash
curl -fsSL https://raw.githubusercontent.com/bwireman/archivist/main/scripts/install.sh | bash
archivist version
```

The binary is written to `~/.local/bin`. `ARCHIVIST_INSTALL_DIR` changes that directory. `ARCHIVIST_TAG` pins a release.

From this checkout:

```bash
make install    # go install ./cmd/archivist
make build      # ./archivist
```

## Set up a repository

```bash
ollama serve
ollama pull qwen3-embedding:0.6b

archivist skills install --user
cd /path/to/your/repo
archivist init
```

`archivist skills install --user` is the setup path: skills under `~/.agents/skills` and one `archivist` entry in `~/.cursor/mcp.json`. `archivist init` in a repo registers that checkout. `archivist import` is optional. `archivist skills install --target cursor` still installs into one checkout and is not the setup path.

From another directory: `archivist --path /path/to/your/repo status`.

## MCP

`archivist mcp` speaks stdio. In the repo, `.cursor/mcp.json`:

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

Run it from the repo root, or pass an absolute `--path`. `archivist` has to be on the PATH the GUI app sees.

## More

- [MCP tools, prompts, and other clients](docs/mcp.md)
- [Records, commands, and export](docs/records.md)
- [Configuration and publish](docs/configuration.md)
- [Agent rules, skills, and this checkout](docs/agents.md)
