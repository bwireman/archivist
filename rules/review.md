# Review the archive occasionally

After the user's request is done, call MCP `review` with `due` true (`archivist review --due` when MCP is unavailable). When `due` is false, or `logging_off` is set, stop. When `due` is true, call `review`. Do not update, retire, or re-rank until the user agrees.

A full review is due every 7 days. A due check does not count as a review. With `log_commands` off there is no clock.

- `missed_rules`: current rules whose paths match the last 7 days of work and never appeared in that log. Report each one as a retrieval gap. When `wide_diff` is true, say the diff is too broad for a miss to mean the rule was skipped.
- `similar`: current records whose embeddings are close. Read each pair. A disagreement is a potential update or retire. Overlap that repeats the same facts is a potential condensation: keep one current record and shorten or retire the other. A feature and its decision stay two records.
- `stale_applies_to`: records whose `applies_to` matches no file. Repo records are always about this checkout. Global and dev records are listed only when a leading directory of `applies_to` exists here. Each one is a potential update.
- `features`: current features with matching files, up to 4 paths each. Read each body against those files. A behavior text that no longer matches the code is a potential update.

When any potential change exists, ask the user before writing. Name each change, the record ids, and whether it is an update, a retire, or a condensation. Ask with a few concrete options. If they agree, make only those changes.

This pass does not score importance. A record that was not retrieved is not ranked lower.
