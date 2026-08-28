# Use Huh for the init TUI

- Status: accepted
- Date: 2026-08-28

## Context
`archivist init` wrote `config.Default()` with no prompts. The embed model is a real choice (`nomic-embed-text` vs a locally installed model). The index command already uses a Charm TUI on a TTY.

## Decision
Show a Huh form on a TTY so `archivist init` can edit Ollama, index, and store settings before writing `.archivist.json`. `--plain` writes `config.Default()` with no prompt.

## Consequences
- Interactive setup can set the embed model without editing JSON.
- Scripts and `--plain` still get the tool defaults.
- Huh is an additional Charm dependency.
