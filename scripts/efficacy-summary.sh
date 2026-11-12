#!/usr/bin/env bash
# Compare scored runs.
# Usage: efficacy-summary.sh <run-dir>...
# Reads <arm>-score.txt (written by efficacy-score.sh) for arms with-archive,
# plan, control. Prints one row per run and arm, then pass counts per arm.
set -euo pipefail
if [ "$#" -lt 1 ]; then
  echo "usage: efficacy-summary.sh <run-dir>..." >&2
  exit 2
fi
python3 - "$@" <<'PY'
import pathlib, sys

arms = ["with", "plan", "control"]
cols = ["hidden", "mcp", "transport", "searches", "gets", "cites", "golden_surfaced", "golden_rank", "golden_read", "golden_cited", "history_read", "tool_calls", "wall"]
print("run | arm | " + " | ".join(cols))
tally = {a: [0, 0] for a in arms}
for d in sys.argv[1:]:
    p = pathlib.Path(d)
    for a in arms:
        f = p / f"{a}-score.txt"
        if not f.exists():
            continue
        kv = {}
        for line in f.read_text().splitlines():
            if ":" in line and not line.startswith(" "):
                k, v = line.split(":", 1)
                kv[k.strip()] = v.strip()
        tally[a][1] += 1
        if kv.get("hidden") == "pass":
            tally[a][0] += 1
        print(f"{p.name} | {a} | " + " | ".join(kv.get(c, "-") for c in cols))
print()
for a in arms:
    ok, n = tally[a]
    print(f"{a}: hidden pass {ok}/{n}")
PY
