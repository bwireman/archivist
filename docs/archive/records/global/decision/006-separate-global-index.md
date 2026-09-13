---
id: rec_bbf3e07cb5eeec0bd837
type: decision
scope: global
status: superseded
title: Separate SQLite index for global ADRs
superseded_by: rec_config014
---

Superseded by rec_config014. Two SQLite files remain, but paths and roles are fixed: `.archivist/index.db` (checkout) and `~/.archivist/archive.db` (global + dev). `global.db`, `store.global_path`, and dump are gone.
