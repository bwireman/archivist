# Archive Index

Type digests: [Rules](rules.md) · [Decisions](decisions.md) · [Features](features.md)

## rule

- [Append gitignore entries on their own line](records/global/rule/append-gitignore-entries-on-their-own-line.md) (accepted) [must]
- [Bump codemap.Version when extractors change](records/global/rule/bump-codemap-version-when-extractors-change.md) (accepted) [must]
- [Code map tests must fixture every registered language](records/global/rule/code-map-tests-must-fixture-every-registered-language.md) (accepted) [must]
- [Do not store session notes as accepted archive records](records/global/rule/do-not-store-session-notes-as-accepted-archive-records.md) (accepted) [should-not]
- [Embed --once must process every queued item](records/global/rule/embed-once-must-process-every-queued-item.md) (accepted) [must]
- [Never pass unsanitized text to FTS5 MATCH](records/global/rule/never-pass-unsanitized-text-to-fts5-match.md) (accepted) [must]

## decision

- [--once drains the whole embed queue](records/global/decision/once-drains-the-whole-embed-queue.md) (accepted)
- [Add feature as a record type for living capability docs](records/global/decision/add-feature-as-a-record-type-for-living-capability-docs.md) (accepted)
- [Archive hygiene: current facts, stubs for superseded, delete non-records](records/global/decision/archive-hygiene-current-facts-stubs-for-superseded-delete-non-records.md) (accepted)
- [Default ADR globs are the product directories](records/global/decision/008-default-adr-globs.md) (superseded)
- [Default embed model is qwen3-embedding:0.6b](records/global/decision/009-qwen3-embed-default.md) (accepted)
- [Default records.global is ~/.archivist](records/global/decision/default-records-global-is-archivist.md) (accepted)
- [Distill conversation into the archive without glut](records/global/decision/distill-conversation-into-the-archive-without-glut.md) (accepted)
- [Drop interactive TUIs; CLI is always plain](records/global/decision/013-plain-cli.md) (accepted)
- [Export archive grouped by record type with type digest files](records/global/decision/export-archive-grouped-by-record-type-with-type-digest-files.md) (accepted)
- [Honor .gitignore when indexing](records/global/decision/007-honor-gitignore.md) (accepted)
- [Language-agnostic code map backup with fixtures per language](records/global/decision/language-agnostic-code-map-backup-with-fixtures-per-language.md) (accepted)
- [Line-based Gleam map and remapping on extractor version](records/global/decision/line-based-gleam-map-and-remapping-on-extractor-version.md) (accepted)
- [Nest ADR globs under index.adr](records/global/decision/005-nested-adr-config.md) (superseded)
- [Optional JSONL command log at .archivist/commands.log](records/global/decision/optional-jsonl-command-log-at-archivist-commands-log.md) (accepted)
- [Pivot to knowledge archive with MCP primary surface](records/global/decision/012-knowledge-archive-pivot.md) (accepted)
- [Record last search in index meta](records/global/decision/011-last-search-meta.md) (accepted)
- [Record schema and CLI version](records/global/decision/004-schema-and-cli-version.md) (accepted)
- [Reshape config around records directories](records/global/decision/014-config-reshape.md) (accepted)
- [Richer chunks and longer CLI results](records/global/decision/010-richer-chunks-and-cli.md) (superseded)
- [Separate SQLite index for global ADRs](records/global/decision/006-separate-global-index.md) (superseded)
- [Separate global and repo ADRs](records/global/decision/003-global-and-repo-adrs.md) (superseded)
- [Ship rule and skill templates in the CLI](records/global/decision/ship-rule-and-skill-templates-in-the-cli.md) (accepted)
- [Split always-on rules from on-demand skills](records/global/decision/015-rules-and-skills.md) (accepted)
- [Treat FTS MATCH input as natural language](records/global/decision/treat-fts-match-input-as-natural-language.md) (accepted)
- [Use Bubble Tea for the index TUI](records/global/decision/001-bubbletea-index-tui.md) (superseded)
- [Use Huh for the init TUI](records/global/decision/002-huh-init-tui.md) (superseded)
- [Use WAL journal mode for SQLite stores](records/global/decision/sqlite-wal-journal.md) (accepted)
- [Use qwen3-embedding:0.6b for this repository](records/repo/decision/001-use-qwen3-embedding.md) (superseded)

## feature

- [Agent rules and skills](records/global/feature/agent-rules-and-skills.md) (accepted)
- [Archive export](records/global/feature/archive-export.md) (accepted)
- [Command log](records/global/feature/command-log.md) (accepted)
- [Embed queue and worker](records/global/feature/embed-queue-and-worker.md) (accepted)
- [Hybrid search](records/global/feature/hybrid-search.md) (accepted)
- [MCP server](records/global/feature/mcp-server.md) (accepted)
- [Typed archive records](records/global/feature/typed-archive-records.md) (accepted)

