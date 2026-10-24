---
name: link-records
description: >
  Create or remove a checkout record link when a record in another repo would
  help this one. Use when search or get shows a cross-checkout connection worth
  keeping, or when the user asks to link repositories. Do not use for a weak
  search hit or on a plan-changes planning turn.
---

# Link records across checkouts

## link-records-procedure

1. Call MCP `links`. Do not insert a duplicate.
2. Choose one kind.
   - `record`: this checkout's record and that checkout's record are the same topic.
   - `pin`: no local record, and every search should include that one remote record.
   - `repo`: a local record means a search that hits it should also search that checkout.
3. Call MCP `link` with `kind`, `from_id`, and `to`. `from_id` is a record in this checkout. `to` is `checkout#id` for `record` and `pin`, and `checkout` for `repo`. Omit `from_id` for `pin`. Use `archivist link` when MCP is down.
4. Read `links` on the next MCP `search` or MCP `get`.

The home store is not a link target. Follow is one hop. A remote record with empty `applies_to` returns no source files. Source files named by `applies_to` are capped at 4 files, the first 2KiB of each, and 24KiB for the response.

MCP `unlink` removes a link that was wrong. A weak or merely nearby hit is not a link. Do not call `link` or `unlink` on a plan-changes planning turn.
