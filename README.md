# Archivist

A local semantic index for a Git repository. Archivist chunks code, docs, comments, git history, and architecture decision records (ADRs), embeds them with [Ollama](https://ollama.com), and stores them in SQLite. `search` and `dump` are the interfaces for you and for agents.

There is no cloud service. The index lives on disk next to the repo (plus one machine-wide ADR database under `~/.archivist/`).

## Requirements

- [Ollama](https://ollama.com) running locally (`ollama serve`, default `http://localhost:11434`)
- An embedding model pulled in Ollama. Init defaults to `qwen3-embedding:0.6b`; pick any model you actually have.
- [Go](https://go.dev) 1.26+ to build or install the CLI from this checkout

## Install the CLI

There is no published binary yet. From this checkout:

```bash
make install
```

That runs `go install ./cmd/archivist` into `$(go env GOPATH)/bin`. Put that directory on your `PATH`, then check:

```bash
archivist version
```

You can also run `./archivist` after `make build` without installing.

## Set up a new repository

1. **Start Ollama** and pull the embed model you will pin in this repo:

   ```bash
   ollama serve   # if it is not already running
   ollama pull qwen3-embedding:0.6b
   ```

   Init can use a different model (for example `nomic-embed-text`). Pull that name instead, and set it in the init form or in `.archivist.json`. Changing the model later needs a full reindex; existing vectors are not comparable across models.

2. **Initialize Archivist in the repo:**

   ```bash
   cd /path/to/your/repo
   archivist init
   ```

   On a TTY, init opens a form for Ollama URL, embed model, skip dirs/globs, gitignore, and ADR globs. `--plain` writes the tool defaults with no prompt:

   ```bash
   archivist init --plain
   ```

   Init writes:

   | Path | Commit? | Purpose |
   | --- | --- | --- |
   | `.archivist.json` | yes | Ollama, skip lists, ADR globs, store paths |
   | `.archivist/` | no | Repo SQLite index (`index.db`). Init appends `.archivist/` to `.gitignore`. |
   | `docs/decisions/` | yes (when you add ADRs) | Repo ADRs for this checkout |
   | `docs/global-decisions/` | only if you author product ADRs | Files here go in the machine-wide global index, not the repo DB |

   `archivist init` does not copy Cursor rules and does not create `~/.archivist/decisions/`.

3. **Index:**

   ```bash
   archivist index
   ```

   On a TTY this shows a progress view (`q` cancels). Scripts and agents should pass `--plain` for a one-line summary:

   ```bash
   archivist index --plain
   ```

   Indexing walks the repo (honoring `.gitignore` by default), skips `docs/dump/`, embeds new or changed chunks, and writes:

   - **Repo index:** `.archivist/index.db` — code, docs, git, and repo ADRs (`docs/decisions/**`)
   - **Global index:** `~/.archivist/global.db` — in-repo `docs/global-decisions/**` plus `~/.archivist/decisions/`

4. **Confirm:**

   ```bash
   archivist status
   ```

5. **Search and dump:**

   ```bash
   archivist search "how auth middleware works"
   archivist dump "how auth middleware works"
   archivist dump "how auth middleware works" -o docs/dump/auth.md
   archivist dump --type adr --adr-scope repo -o docs/dump/decisions.md
   ```

   Query `search`/`dump` need Ollama. Dumping already-indexed ADRs with `--type adr` and no query does not.

From another directory, pass `--path`:

```bash
archivist --path /path/to/your/repo status
```

### Optional Makefile targets

This checkout’s `Makefile` is for building Archivist. In an application repo you only need something like:

```makefile
.PHONY: index dump-docs refresh-docs

index:
	archivist index --plain

dump-docs:
	mkdir -p docs/dump
	archivist dump --type adr --adr-scope repo -o docs/dump/decisions.md

refresh-docs: index dump-docs
```

## Two indexes

Default `search` and `dump` use **this checkout’s** `.archivist/index.db` (code, docs, git, repo ADRs). They do not mix in other projects.

`--adr-scope global` reads `~/.archivist/global.db`: product ADRs from `docs/global-decisions/` in repos you have indexed, and user ADRs under `~/.archivist/decisions/`. Use that only when the question is about those shared decisions, not this repo’s code.

`--scope` is a path prefix or glob inside the index you opened. It is not ADR kind.

## Cursor / agent rules

Init does not install editor rules. Copy the four files under [`.cursor/rules/`](.cursor/rules/) from this repo into the new project, then change two things so they match a repo that has Archivist on `PATH` rather than a local `./archivist` binary:

- Use `archivist` instead of `./archivist`.
- Use `archivist index --plain` (or `make index` if you added the targets above) instead of `make index` from this checkout.

Keep the workflow the rules describe:

1. **Consult** the index (then `docs/decisions/` / `docs/dump/`) before guessing APIs or past decisions.
2. **Record** real design choices as ADRs in `docs/decisions/NNN-slug.md` (user-wide: `~/.archivist/decisions/`). Do not put application decisions in `docs/global-decisions/`; that directory is for how Archivist itself works.
3. **Dump** retrieved context into `docs/dump/` after decisions or architecture work. Do not hand-edit dumps; `docs/dump/` is not indexed.
4. **Re-index** at the end of a turn that changed indexed files. Skip if you only touched `docs/dump/` or `.archivist/`. If Ollama is down, say so and continue.

ADR shape:

```markdown
# Use SQLite for the local index

- Status: accepted
- Date: 2026-08-28
- Scope: repo

## Context
Why the question came up.

## Decision
What we chose, in one or two sentences.

## Consequences
- What becomes easier
- What we are accepting
```

`NNN` is the next number **in that directory**. Status is `proposed`, `accepted`, `deprecated`, or `superseded`. `Scope` is for humans; classification is the path (repo vs global globs).

## Commands

| Command | Needs Ollama | What it does |
| --- | --- | --- |
| `archivist init` | no | Write `.archivist.json`, data dirs, ADR dirs, gitignore |
| `archivist index` | yes | Incremental embed into repo + global SQLite |
| `archivist search <query>` | yes | Semantic search (`--type`, `--adr-scope`, `--top`, `--json`) |
| `archivist dump [query]` | yes if there is a query | Markdown for an LLM (`-o` file or directory) |
| `archivist status` | no (reports embedder health) | Chunk counts, last index time, Ollama reachability |
| `archivist version` | no | CLI version and schema |

`--type` is `code`, `doc`, `commit`, `adr`, or `comment`.

## This checkout

This repository is the Archivist tool. `archivist init` and this checkout both default to `qwen3-embedding:0.6b`.

```bash
make build          # ./archivist
make test
make index          # ./archivist index --plain
make dump-docs      # repo ADRs and (separately) global ADRs into docs/dump/
make refresh-docs   # index, then dump-docs
```

Product ADRs live in `docs/global-decisions/`. Pins that must not follow the binary (embed model, local paths) live in `docs/decisions/`. Regenerated dumps are under `docs/dump/`; do not edit them by hand.
