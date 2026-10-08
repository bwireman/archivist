# Record decisions, rules, and features

Scan this conversation for durable knowledge. When a lasting choice, constraint, or capability description appears, write it to the archive in this turn. Do not wait for "remember this." Do not leave it only in chat.

On a plan-changes planning turn, do not remember, update, or retire, including interview answers. Those writes happen only in the accepted phase the plan names. This ban wins over the same-turn write above.

- `decision` — a real choice among alternatives (Context, Decision, Consequences).
- `rule` — must/must-not or should/should-not that future work should follow.
- `feature` — how a capability works, what it connects to, and how to invoke it.

Search first. Update or retire an existing record instead of adding a parallel note. One current document per topic. Keep bodies short: facts, not narration.

Musts that an accepted plan-changes phase assigns to a named decision and the skill stay in those two places. Do not remember a rule for that same topic.

Skip chat transcripts, restatements of records already on file, ephemeral session state (this chat's errors, "MCP is down"), unverified catalogs, and details that live only in code.

Use MCP `remember` / `update` / `retire` or `archivist remember`. These write SQLite only.

Before remember, search with no scope argument. Search covers repo, global, and dev.
If a current record covers the topic, update it and keep its scope.
If search shows a gap, pick one scope:
- dev: about the person or this machine, including a short-gap answer
- repo: true only in this checkout
- global: true for the product in every checkout

Markdown under `records.repo` / `records.global` (default `~/.archivist`) / `records.dev` is optional import input, not the live archive.
