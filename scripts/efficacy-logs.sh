#!/usr/bin/env bash
# Evaluate the search, get, and cite entries in an arm's commands.log.
# Usage: efficacy-logs.sh <run-dir> <arm> <db-file>
# Copies the fixture log to <run-dir>/<arm>-commands.log, then prints:
#   transport, searches, queries, golden_surfaced, gets, golden_read,
#   cites, cites_invalid, golden_cited.
# Golden records come from <db-file>: accepted rules or decisions holding
# notebook-transfer, and the COPY-THEN-DELETE pitfall. CLI search results are
# not logged, so golden_surfaced counts MCP searches only (n/a when none).
set -euo pipefail

work="${EFFICACY_WORK:-${XDG_CACHE_HOME:-$HOME/.cache}/archivist-efficacy}"
if [ "$#" -ne 3 ]; then
  echo "usage: efficacy-logs.sh <run-dir> <arm> <db-file>" >&2
  exit 2
fi
R=$(realpath "$1")
arm=$2
db=$(realpath "$3")
log="$work/project/.archivist/commands.log"

if [ ! -f "$log" ]; then
  echo "logs: none"
  exit 0
fi
cp "$log" "$R/$arm-commands.log"

golden=$(sqlite3 "$db" "select id from records where status='accepted' and ((type in ('rule','decision') and body like '%notebook-transfer%') or (type='pitfall' and body like '%COPY-THEN-DELETE%'))" | paste -sd,)

python3 - "$R/$arm-commands.log" "$golden" <<'PY'
import json, sys

golden = set(filter(None, sys.argv[2].split(",")))
ins, outs = {}, {}
order = []
for line in open(sys.argv[1]):
    try:
        d = json.loads(line)
    except ValueError:
        continue
    key = d.get("id") or f'{d.get("ts")}{d.get("command")}'
    (ins if d.get("dir") == "in" else outs)[key] = d
    if d.get("dir") == "in":
        order.append(key)

def name(d):
    return d["command"].removeprefix("archivist ")

def first_arg(a):
    a = a or {}
    if "id" in a:
        return a["id"]
    return (a.get("args") or [""])[0]

def hit_id(h):
    rec = h.get("Record") or h.get("record") or {}
    return h.get("id") or rec.get("ID") or rec.get("id")

searches, gets, cites = [], [], []
transports = set()
for k in order:
    i, o = ins[k], outs.get(k, {})
    n = name(i)
    transports.add(i["source"])
    if n == "search":
        a = i.get("args") or {}
        q = a.get("query") or first_arg(a)
        res = o.get("result")
        ids = None
        if isinstance(res, dict) and res.get("truncated"):
            print("warning: truncated search result; golden_surfaced may undercount", file=sys.stderr)
        if i["source"] == "mcp" and res is not None:
            hits = res.get("results") if isinstance(res, dict) else res
            if isinstance(hits, list):
                ids = [hit_id(h) for h in hits]
        searches.append((q, ids))
    elif n == "get":
        gets.append(first_arg(i.get("args")))
    elif n == "cite":
        a = i.get("args") or {}
        eff = a.get("effect") or (a.get("flags") or {}).get("effect") or ""
        res = o.get("result")
        failed = not o or bool(o.get("error")) or (isinstance(res, dict) and bool(res.get("is_error")))
        cites.append((first_arg(a), eff, failed))

print("transport:", ",".join(sorted(t for t in transports if t in ("mcp", "cli"))) or "none")
print("searches:", len(searches))
for q, _ in searches:
    print("  query:", q)
mcp = [ids for _, ids in searches if ids is not None]
if mcp:
    print("golden_surfaced:", "yes" if any(golden & set(ids) for ids in mcp) else "no")
    ranks = [next((n + 1 for n, r in enumerate(ids) if r in golden), None) for ids in mcp]
    print("golden_rank:", ",".join(str(r) if r else "-" for r in ranks))
else:
    print("golden_surfaced: n/a")
print("gets:", len(gets))
print("golden_read:", "yes" if golden & set(gets) else "no")
print("cites:", len(cites))
for rid, eff, bad in cites:
    tag = "invalid" if bad else ("golden" if rid in golden else "other")
    print(f"  cite {rid} [{tag}]: {eff}")
print("cites_invalid:", sum(1 for _, _, b in cites if b))
print("golden_cited:", "yes" if any(r in golden and not b for r, _, b in cites) else "no")
print("golden_ids:", ",".join(sorted(golden)) or "none")
PY
