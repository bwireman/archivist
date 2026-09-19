---
id: rec_d5ab1d91d8f643c7f711
type: decision
scope: global
status: accepted
title: Read the code map and commit history instead of dropping them
applies_to: [internal/store/**, internal/index/**, internal/mcp/server.go, internal/cmd/map.go]
---

## Context

A dead-code review found two tables that `archivist index` populated on every run but nothing ever read: `symbol_edges` (import graph from all three extractors) and `commits` (last 500 git commits, read back only as a hash set for insert dedupe). The `map` tool queried `symbols` alone.

Alternatives: delete both and stop writing them; keep writing them but make indexing opt-in; give them readers.

## Decision

Give them readers. `store.ExploreCode` composes symbol search, `ImportsFrom`, `ImportersOf`, and `SearchCommits` into one `CodeSearch` result, exposed as the MCP `map` tool and the `archivist map` CLI command.

## Consequences

- `map` is code exploration, not just symbol lookup; its result shape changed from a bare symbol array to a `CodeSearch` object with `symbols`, `imports`, `importers`, and `commits`.
- Indexed git history is now user-visible, so dropping the git phase of `index` would be a visible regression.
- `SearchSymbols` also matches `file_path`, so a package-path query reaches the files whose imports are worth showing.
