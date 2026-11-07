#!/usr/bin/env bash
# Build the blind reviewer prompt from prompts/review.md and the arm patches.
# Usage: efficacy-review-prompt.sh <run-dir>
# Writes <run-dir>/review-prompt.txt and prints its SHA-256. Pass that file's
# exact bytes to the reviewer.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
if [ "$#" -ne 1 ]; then
  echo "usage: efficacy-review-prompt.sh <run-dir>" >&2
  exit 2
fi
R=$(realpath "$1")
out="$R/review-prompt.txt"
{
  cat "$root/testdata/efficacy-fixture/prompts/review.md"
  for arm in with plan control; do
    printf '\n### %s\n\n' "$arm"
    if [ -s "$R/$arm.patch" ]; then
      printf '```diff\n'
      cat "$R/$arm.patch"
      printf '```\n'
    else
      printf '(missing)\n'
    fi
  done
} >"$out"
sha256sum "$out" | cut -d' ' -f1
