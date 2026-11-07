#!/usr/bin/env bash
# Start an efficacy run: create the run dir, check Ollama, record run facts.
# Usage: efficacy-preflight.sh <run-dir>
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
if [ "$#" -ne 1 ]; then
  echo "usage: efficacy-preflight.sh <run-dir>" >&2
  exit 2
fi
R=$1
if [ -d "$R" ] && [ -n "$(ls -A "$R")" ]; then
  echo "efficacy-preflight.sh: $R already has files" >&2
  exit 1
fi
mkdir -p "$R"
R=$(realpath "$R")

ollama="${OLLAMA:-http://localhost:11434}"
model=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1])).get("embed_model",""))' \
  "$root/testdata/efficacy-fixture/run.json")
if ! tags=$(curl -sf --max-time 5 "$ollama/api/tags"); then
  echo "efficacy-preflight.sh: Ollama not reachable at $ollama" >&2
  exit 1
fi
if [ -n "$model" ] && ! python3 -c '
import json, sys
names = {m.get("name", "") for m in json.loads(sys.argv[1]).get("models", [])}
want = sys.argv[2]
sys.exit(0 if want in names or want + ":latest" in names else 1)
' "$tags" "$model"; then
  echo "efficacy-preflight.sh: Ollama lacks embed_model $model (ollama pull $model)" >&2
  exit 1
fi

basename "$R" >"$R/utc.txt"
archivist version >"$R/version.txt"
git -C "$root" rev-parse HEAD >"$R/archivist-commit.txt"
git -C "$root" status --porcelain >"$R/repo-dirty.txt"
(cd "$root" && stat -c '%n %Y' .archivist/index.db "$HOME/.archivist/archive.db") >"$R/mtimes-before.txt"
echo "preflight ok: $R"
