---
id: rec_d3c2e36d4c63cff21aad
type: rule
scope: global
status: accepted
title: Bump codemap.Version when extractors change
severity: must
applies_to: [internal/codemap/**]
tags: [codemap]
---

When adding or changing a code-map extractor (tree-sitter specs, line-based languages, or the generic fallback), increment `codemap.Version`. Index skips unchanged files by content hash; a stale `codemap_version` in meta is what forces a remap of existing trees.
