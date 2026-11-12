#!/usr/bin/env bash
# Score one efficacy arm in the fixture checkout.
# Usage: efficacy-score.sh <run-dir> <arm> [db-name] [--agent <id>]...
# The parent writes <arm>-wall-start.txt before launch and <arm>-wall-end.txt
# when the agent returns; this script never writes wall files.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
work="${EFFICACY_WORK:-${XDG_CACHE_HOME:-$HOME/.cache}/archivist-efficacy}"
W="$work/project"

if [ "$#" -lt 2 ]; then
  echo "usage: efficacy-score.sh <run-dir> <arm> [db-name] [--agent <id>]..." >&2
  exit 2
fi
R=$(realpath "$1")
arm=$2
shift 2
db=""
agents=()
while [ "$#" -gt 0 ]; do
  case "$1" in
  --agent)
    agents+=("$2")
    shift 2
    ;;
  *)
    db=$1
    shift
    ;;
  esac
done

if [ ! -f "$R/$arm-wall-end.txt" ]; then
  echo "harness: missing wall-end ($R/$arm-wall-end.txt)" >&2
  exit 1
fi

# Keep the printed score so efficacy-summary.sh can compare runs later.
exec > >(tee "$R/$arm-score.txt")

echo "== $arm"
if (cd "$W" && go test ./transfer -count=1 >"$R/$arm-test.txt" 2>&1); then
  echo "go test ./transfer: pass"
else
  echo "go test ./transfer: fail"
fi
if "$root/scripts/efficacy-hidden.sh" "$W" >"$R/$arm-hidden.txt" 2>&1; then
  echo "hidden: pass"
else
  echo "hidden: fail"
fi

install_re='^(\.archivist|\.archivist\.json|\.archivist-install\.json|\.archivist-disabled|\.cursor/|bin/)'
(
  cd "$W"
  untracked=$(git ls-files --others --exclude-standard | grep -vE "$install_re" || true)
  if [ -n "$untracked" ]; then
    # shellcheck disable=SC2086
    git add -N $untracked
  fi
  git diff -- . ':!.archivist.json' ':!.archivist-install.json' ':!.archivist-disabled' ':!.cursor' ':!bin'
) >"$R/$arm.patch"
echo "patch lines: $(wc -l <"$R/$arm.patch")"

if grep -rq notebook-transfer "$W" --include='*.go'; then
  echo "notebook-transfer: present"
else
  echo "notebook-transfer: absent"
fi
if [ -d "$W/transfer" ] && grep -lq '^package transfer$' "$W"/transfer/*.go 2>/dev/null &&
  find "$W/transfer" -name '*.go' ! -name '*_test.go' | grep -q .; then
  echo "package transfer: present"
else
  echo "package transfer: absent"
fi

python3 - "$W/README.md" <<'PY'
import re, sys
lines = open(sys.argv[1]).read().splitlines()
hits = [i for i, l in enumerate(lines) if "COPY-THEN-DELETE" in l]
ok = not hits or all(
    any(re.search(r"supersed|obsolete", l, re.I) for l in lines[max(0, i - 3):i + 4])
    for i in hits
)
print("docs_cleanup:", "yes" if ok else "no")
PY

if [ -f "$R/seed.db" ] && [ -n "$db" ] && [ -f "$W/.archivist/index.db" ]; then
  out=$(cd "$root" && EFFICACY_CHECKPOINT="$W/.archivist/index.db" \
    go test ./internal/efficacy/ -run '^TestCheckpointWALOnDB$' -count=1 -v 2>&1)
  if grep -q -- '--- PASS: TestCheckpointWALOnDB' <<<"$out"; then
    cp "$W/.archivist/index.db" "$R/$db"
    echo "${db%.db}_db: copied"
    q="select id||' '||type||' '||status||' '||coalesce(superseded_by,'')||' '||length(body)||' '||title from records order by id"
    if command diff <(sqlite3 "$R/seed.db" "$q") <(sqlite3 "$R/$db" "$q") >"$R/$arm-archive.diff"; then
      echo "archive: unchanged"
    else
      echo "archive: changed (see $arm-archive.diff)"
    fi
  else
    echo "${db%.db}_db: harness"
  fi
else
  echo "archive: skipped"
fi

if [ -f "$W/.archivist/commands.log" ]; then
  python3 - "$W/.archivist/commands.log" <<'PY'
import json, sys
src = set()
for line in open(sys.argv[1]):
    try:
        src.add(json.loads(line).get("source", ""))
    except ValueError:
        pass
print("mcp:", "used" if "mcp" in src else ("cli" if src else "none"))
PY
else
  echo "mcp: none"
fi

if [ -n "$db" ] && [ -f "$R/$db" ]; then
  "$root/scripts/efficacy-logs.sh" "$R" "$arm" "$R/$db"
else
  echo "logs: skipped"
fi

transcript=""
if [ "${#agents[@]}" -gt 0 ]; then
  transcript=$(find "$HOME/.cursor/projects" -path '*/subagents/*' -name "${agents[0]}.jsonl" 2>/dev/null | head -1)
fi
if [ -n "$transcript" ]; then
  python3 - "$transcript" <<'PY'
import json, re, sys
calls = 0
hist = False
pat = re.compile(r"\bgit\b.*\b(log|show|blame|diff|grep|cat-file|checkout|restore|rev-list|reflog|stash)\b|origin\.git")
for line in open(sys.argv[1]):
    d = json.loads(line)
    for block in (d.get("message") or {}).get("content") or []:
        if block.get("type") != "tool_use":
            continue
        calls += 1
        cmd = (block.get("input") or {}).get("command") or ""
        if pat.search(cmd):
            hist = True
print("tool_calls:", calls)
print("history_read:", "yes" if hist else "no")
PY
else
  echo "tool_calls: unavailable"
  echo "history_read: unknown"
fi

python3 - "$R" "$arm" <<'PY'
import pathlib, sys
r, arm = pathlib.Path(sys.argv[1]), sys.argv[2]
starts = [p for p in r.glob("*-wall-start.txt")
          if p.name == f"{arm}-wall-start.txt" or (arm == "plan" and p.name.startswith("plan-resume-"))]
total = 0
for s in starts:
    e = s.with_name(s.name.replace("-wall-start", "-wall-end"))
    if not e.exists():
        sys.exit(f"harness: unpaired wall file {s.name}")
    total += int(e.read_text()) - int(s.read_text())
print(f"wall: {total}s")
PY
