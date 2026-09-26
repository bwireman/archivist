---
name: init-archive
description: Scan a repository for existing decisions, rules, features, guides, maps, and pitfalls, write the gaps into the Archivist archive, and embed them. Use when the archive is empty or thin, after archivist init, or when the user asks to seed or initialize the archive from the repo. Do not use for a single new choice (record-decision, record-rule, record-feature).
---

# Initialize the archive from the repo

`archivist init` creates config and an empty store. This skill fills that store from material already in the checkout so search works before new work starts.

## 1. Prepare

1. If `.archivist.json` is missing, run `archivist init`.
2. `archivist import` so typed markdown already on disk is in SQLite. Import never deletes DB-only rows.
3. `archivist index` so the code map exists. No Ollama.
4. `status` (MCP or `archivist status`). Continue when records already exist; skip topics search already covers.

## 2. Scan

Look for durable knowledge already written down. Prefer, in order:

- ADR or decision docs, `docs/decisions`, architecture notes, RFCs
- README and contributor docs that state how a capability works or a constraint
- In-repo rules and policy docs that are product constraints
- A short map of the main packages from `archivist map` or MCP `map`

Types: `decision` (a real choice among alternatives), `rule` (must/should), `feature` (how a capability works), `guide`, `map`, `pitfall`.

Skip source bodies, chat logs, generated `docs/archive/`, `.archivist/`, secrets, and one-off TODOs. Do not invent a record because code exists. One current document per topic.

## 3. Write

For each candidate, `search` the topic (filter `--type` when it is clear). If a current record exists, `update` it or leave it. Otherwise follow `record-decision`, `record-rule`, or `record-feature` and `remember` a short body.

Scope: `repo` for this checkout. `global` only when the text is product-wide. Do not use `dev` for a repo scan.

## 4. Embed

```bash
archivist embed --once
archivist export
```

`embed --once` drains the queue so hybrid search can use the new records. Skip embed if Ollama is down; keyword search still works. `export` is a no-op unless `records.write_docs` is true.

Report what was added, what was already present, and whether embed ran.
