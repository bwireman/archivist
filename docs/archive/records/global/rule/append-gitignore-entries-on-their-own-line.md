---
id: rec_184e12358de7c5b12414
type: rule
scope: global
status: accepted
title: Append gitignore entries on their own line
severity: must
applies_to: [internal/cmd/**]
tags: [init, gitignore]
---

When appending a line to `.gitignore`, insert a newline first if the file does not already end with one. Otherwise `init` glues `.archivist/` onto the last existing entry and the ignore never matches.
