---
id: rec_plaincli013
type: decision
scope: global
status: accepted
title: Drop interactive TUIs; CLI is always plain
supersedes: ["rec_03b53c31d84603de473c", "rec_487f791d4ced16a80cde"]
---

# Drop interactive TUIs; CLI is always plain

- Status: accepted
- Date: 2026-09-12
- Scope: global

## Context

`archivist index` showed a Bubble Tea progress view on a TTY, and `archivist init` opened a Huh form. Agents and scripts already passed `--plain`. Charm libraries were a large dependency surface for a product whose primary interface is MCP plus one-line CLI.

## Decision

Remove the TUI package and Charm dependencies. `init` always writes `config.Default()`. `index` always prints a one-line summary. `--plain` remains as a hidden no-op so existing scripts keep working.

## Consequences

- No Bubble Tea, Huh, Lip Gloss, or go-isatty in the CLI.
- Interactive setup is editing `.archivist.json`.
- `make index` and leftover `--plain` flags still work; `--plain` is a hidden no-op.
