---
id: rec_e62e2384d8f10e17a473
type: decision
scope: global
status: accepted
title: Default records.global is ~/.archivist
tags: [config, records]
---

## Context

`records.global` defaulted to in-repo `docs/global-decisions`. Global records were already stored in `~/.archivist/archive.db`, but the markdown lived in every checkout. `records.dev` already defaulted to `~/.archivist/records`. Putting machine-wide docs beside the home store matches that split.

## Decision

- Empty `records.global` means `~/.archivist`. Absolute and `~/…` values are also outside the checkout.
- `archivist init` creates `~/.archivist` for global docs and does not create `docs/global-decisions`.
- Scope `global` writes `~/.archivist/<slug>.md` (virtual path `global/…`). The indexer walks that directory for `.md` files and skips `records.dev` (`~/.archivist/records/`).
- A checkout-relative `records.global` (this repository uses `docs/global-decisions`) keeps product-wide records in git.

## Consequences

- New repos share machine-wide global docs without copying `docs/global-decisions`.
- This product archive stays in-repo because `.archivist.json` sets `records.global` explicitly.
- Markdown files at the top of `~/.archivist` are records; `archive.db` and `records/` stay out of that walk.
