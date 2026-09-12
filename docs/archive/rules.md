# Rules

## Append gitignore entries on their own line

- Severity: must
- Applies to: internal/cmd/**

When appending a line to `.gitignore`, insert a newline first if the file does not already end with one. Otherwise `init` glues `.archivist/` onto the last existing entry and the ignore never matches.

---

