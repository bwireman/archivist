---
id: rec_179dbf51d8a090fef8d7
type: feature
scope: global
status: accepted
title: Typed archive records
applies_to: [internal/record/**, internal/archive/**]
tags: [records]
---

## Purpose

Markdown files with YAML front matter are the source of truth for the archive. SQLite indexes them for search; export mirrors them under `docs/archive/`.

## Behavior

Types: `decision`, `rule`, `feature`, `guide`, `map`, `pitfall`. Scopes: `dev`, `repo`, `global`. Status: `proposed`, `accepted`, `deprecated`, `superseded`. Rules may set `severity` and `applies_to` globs for `archivist check`. Features should set `applies_to` to the packages they document.

`remember` writes a file under the configured records directory and upserts the store (FTS + embed queue). If that default path already exists (on disk or in the index), `remember` reuses the existing record id and overwrites the file — same identity as `update`. Writes are atomic (temp file + rename) then `UpsertRecord`; `archivist index` repairs the store if the file lands and the upsert fails. `update` rewrites the file in place. `retire` sets `status=superseded` and optional `superseded_by`. Query overlay prefers repo over global over dev for the same slug.

`UpsertRecord` keys on record id after adopting any row already stored at `source_path`, so a second write at the same path cannot leave FTS or `embed_queue` pointing at a new id that is not in `records`.

In this product checkout, `records.global` is `docs/global-decisions` so product records stay in git. Empty `records.global` in other repos is `~/.archivist`.

## Connects to

- Config: `records.repo`, `records.global`, `records.dev`, `records.export`.
- Indexer walks those directories (and home global/dev) and prunes missing files.
- Export writes `INDEX.md` by `record.IndexOrder`, a full-text digest per type (`rules.md`, `features.md`, …), and copies under `records/<scope>/<type>/`.
- Check only enforces `rule` records (`applies_to` glob matches); semantic hits are advisory.
- On-demand skills: `record-decision`, `record-rule`, `record-feature`. Always-on `rules/record.md` tells agents to distill lasting facts from this conversation (search first, skip chat glut). MCP initialize `instructions` are the consult + record templates so hosts without skills install still get that bar.

## Entry points

- CLI: `archivist remember`, `update`, `retire`, `check`
- MCP: `remember`, `update`, `retire`, `get`, `check`
- Types: `record.Record`, `archive.Service`, `store.UpsertRecord`
