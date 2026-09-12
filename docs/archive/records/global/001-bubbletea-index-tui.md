---
id: rec_03b53c31d84603de473c
type: decision
scope: global
status: accepted
title: Use Bubble Tea for the index TUI
---

# Use Bubble Tea for the index TUI

- Status: accepted
- Date: 2026-08-28
- Scope: global

## Context
`archivist index` printed one line when it finished. Embedding is per-chunk and can take a while, so a live view of the current file and counts is useful. `make index` and piped runs still need a log-friendly summary.

## Decision
Show a Bubble Tea TUI when stdout is a TTY. Fall back to a one-line summary when it is not, or when `--plain` is set (`make index` uses `--plain`). The indexer reports progress through a callback so tests do not need a terminal.

## Consequences
- Interactive runs of `archivist index` show scan/file/git progress and can cancel with q.
- `make index` passes `--plain` so agent and CI logs stay a single summary line.
- Charm libraries (bubbletea, bubbles, lipgloss) are dependencies of the CLI.
