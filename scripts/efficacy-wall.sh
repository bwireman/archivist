#!/usr/bin/env bash
# Write one wall-clock stamp.
# Usage: efficacy-wall.sh <run-dir> <name> start|end
# <name> is seed, with, plan, plan-resume-N, control, or review. Run this with
# unsandboxed permissions while the agent root is the fixture: a sandboxed
# shell cannot write the run dir then, and a missing stamp makes scoring fail.
set -euo pipefail
if [ "$#" -ne 3 ] || { [ "$3" != start ] && [ "$3" != end ]; }; then
  echo "usage: efficacy-wall.sh <run-dir> <name> start|end" >&2
  exit 2
fi
f="$1/$2-wall-$3.txt"
date +%s >"$f" || {
  echo "efficacy-wall.sh: cannot write $f (needs unsandboxed shell)" >&2
  exit 1
}
echo "$f"
