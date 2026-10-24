# Link records across checkouts

When you decide a record in another checkout would help a later session in this one, store a link. A weak or merely nearby hit is not a link. Do not link a checkout to itself. Do not call `link` or `unlink` on a plan-changes planning turn.

Use MCP `link` (`archivist link` when MCP is down). `kind` is `record`, `pin`, or `repo`. `from_id` is a record in this checkout. `to` is `checkout#id` for `record` and `pin`, and `checkout` for `repo`. Omit `from_id` for `pin`.

- `record`: this checkout's record and that checkout's record are the same topic.
- `pin`: no local record, and every search in this checkout should include that one remote record.
- `repo`: a local record means a search that hits it should also search that checkout.

Call MCP `links` first and do not insert a duplicate. MCP `unlink` removes a link that was wrong. Read `links` on search and get before you treat the other checkout as unknown.
