#!/usr/bin/env bash
# Check that Cursor has loaded the fixture's archivist MCP server, and no other.
# Run after moving the agent root into the fixture, before each archive arm
# (seed, with, plan). Never control. Never in parallel with the launch.
# Cursor lists a loaded server under the project's mcps/ directory only while
# that workspace is open. user-archivist is the real ~/.archivist and must not
# be loaded: a subagent will call it and read this repo's archive.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
# shellcheck source=efficacy-paths.sh
source "$root/scripts/efficacy-paths.sh"
project=$(efficacy_project)
folder=$(basename "$project")
slug=$(printf '%s' "$project" | tr -d . | tr / - | sed 's/^-//')
mcps="$HOME/.cursor/projects/$slug/mcps"
fixture_re="^project-[0-9]+-${folder}-archivist$"

ready=""
for _ in 1 2 3 4 5 6; do
  names=$(ls "$mcps" 2>/dev/null || true)
  fixture=$(printf '%s\n' "$names" | grep -E "$fixture_re" || true)
  others=$(printf '%s\n' "$names" | grep 'archivist$' | grep -vE "$fixture_re" || true)
  if [ -n "$fixture" ] && [ -z "$others" ]; then
    name=$(printf '%s\n' "$fixture" | head -1)
    if [ "$(printf '%s\n' "$fixture" | grep -c .)" -eq 1 ] &&
      [ -f "$mcps/$name/tools/search.json" ]; then
      ready=$name
      break
    fi
  fi
  if [ -n "$others" ]; then
    break
  fi
  sleep 5
done

if [ -n "$others" ]; then
  echo "efficacy-mcp-ready.sh: another archivist server is loaded ($others)." >&2
  echo "  It opens a different archive. Disable it in Cursor Settings > MCP for this workspace, then rerun." >&2
  echo "  Archive arms need only the fixture server (project-N-${folder}-archivist) in $mcps." >&2
  exit 1
fi
if [ -z "$ready" ]; then
  echo "efficacy-mcp-ready.sh: fixture archivist MCP is not loaded (no project-N-${folder}-archivist with tools/search.json in $mcps)." >&2
  echo "  With $project as the workspace root, enable that server in Cursor Settings > MCP." >&2
  exit 1
fi
echo "mcp ready: $ready"
