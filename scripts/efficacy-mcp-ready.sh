#!/usr/bin/env bash
# Check that Cursor has loaded the fixture's archivist MCP server.
# Run after --prepare with and after moving the agent root into the fixture,
# before launching the seed. Cursor lists a loaded server under the project's
# mcps/ directory only while that workspace is open.
set -euo pipefail

work="${EFFICACY_WORK:-${XDG_CACHE_HOME:-$HOME/.cache}/archivist-efficacy}"
slug=$(printf '%s' "$work/project" | tr -d . | tr / - | sed 's/^-//')
mcps="$HOME/.cursor/projects/$slug/mcps"
for _ in 1 2 3 4 5 6; do
  if ls "$mcps" 2>/dev/null | grep -q 'archivist$'; then
    echo "mcp ready: $(ls "$mcps" | grep 'archivist$')"
    exit 0
  fi
  sleep 5
done
echo "efficacy-mcp-ready.sh: Cursor has not loaded the fixture archivist MCP server (no *archivist in $mcps)." >&2
echo "  With $work/project as the workspace root, enable archivist in Cursor Settings > MCP." >&2
exit 1
