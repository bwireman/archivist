# Record a design decision

When you choose between real alternatives, record a decision in the archive.

1. Run `archivist remember --type decision --scope repo --title "..." --body "..."` or use the MCP `remember` tool.
2. Use scope `repo` for checkout-specific decisions, `global` for product-wide, `dev` for personal.
3. Run `archivist index --plain` then `archivist export` to refresh the generated tree.

Body should include Context, Decision, and Consequences sections.
