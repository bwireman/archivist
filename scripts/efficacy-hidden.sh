#!/usr/bin/env bash
# Score a notes checkout with transfer tests that are not in the fixture tree.
# Copies the test in, runs it, and deletes it before exiting.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
src="$root/testdata/efficacy-hidden/transfer_hidden_test.go"
dest="${1:-}"
if [ -z "$dest" ] || [ ! -f "$dest/go.mod" ]; then
  echo "usage: efficacy-hidden.sh <notes-checkout>" >&2
  exit 2
fi
dest=$(cd "$dest" && pwd)
target="$dest/transfer/hidden_test.go"
mkdir -p "$dest/transfer"
cleanup() { rm -f "$target"; }
trap cleanup EXIT
cp "$src" "$target"
set +e
out=$(go -C "$dest" test ./transfer -count=1 -run '^TestHidden' 2>&1)
code=$?
set -e
printf '%s\n' "$out"
if printf '%s\n' "$out" | grep -q 'no tests to run'; then
  exit 1
fi
exit "$code"
