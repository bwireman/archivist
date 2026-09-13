---
id: rec_8240f8ac1fec1a19cb38
type: rule
scope: global
status: accepted
title: Embed --once must process every queued item
severity: must
applies_to: [internal/embed/**, internal/store/**]
tags: [embed, queue]
---

`archivist embed --worker --once` must process every row currently in `embed_queue` (repo and home). It must not stop after a batch size, and it must not skip remaining items because one sibling failed. Queue rows with no matching record must be deleted, not counted as successfully embedded.
