# Record decisions, rules, and features

Scan this conversation for durable knowledge. When a lasting choice, constraint, or capability description appears, write it to the archive in this turn. Do not wait for "remember this." Do not leave it only in chat.

- `decision` — a real choice among alternatives (Context, Decision, Consequences).
- `rule` — must/must-not or should/should-not that future work should follow.
- `feature` — how a capability works, what it connects to, and how to invoke it.

Search first. Update or retire an existing record instead of adding a parallel note. One current document per topic. Keep bodies short: facts, not narration.

Skip chat transcripts, restatements of records already on file, ephemeral session state (this chat's errors, "MCP is down"), unverified catalogs, and details that live only in code.

Use MCP `remember` / `update` / `retire` or `archivist remember`. These write SQLite only. Scope: `repo` this checkout, `global` the product, `dev` personal. Markdown under `records.repo` / `records.global` (default `~/.archivist`) / `records.dev` is optional import input, not the live archive.
