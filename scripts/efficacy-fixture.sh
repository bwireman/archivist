#!/usr/bin/env bash
# Rebuild the efficacy checkout from testdata/efficacy-fixture.
# Git config, hooks, and identity come from this script.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
src="$root/testdata/efficacy-fixture"
# Outside this repo so an agent in the checkout cannot walk up into the
# hidden tests or this repo's archive.
work="${EFFICACY_WORK:-${XDG_CACHE_HOME:-$HOME/.cache}/archivist-efficacy}"
case "$work/" in
"$root"/*)
  echo "efficacy-fixture.sh: EFFICACY_WORK must be outside $root" >&2
  exit 1
  ;;
esac
dest="$work/project"
origin="$work/origin.git"
home_dir="$work/home"

usage() {
  cat <<'EOF'
Usage: efficacy-fixture.sh [--prepare with|feature|plan|control] [snapshot-dir]

With no flags, replay the checkout at the last manifest commit and push main.
--prepare with     checkout pre-transfer, install an empty archive
--prepare feature  restore snapshot onto main and drop plan-changes
--prepare plan     restore snapshot onto main and install the unattended skill
--prepare control  replay main with no archive and .archivist-disabled
EOF
}

mode=""
snap=""
if [ "${1:-}" = "--prepare" ]; then
  mode="${2:-}"
  snap="${3:-}"
  case "$mode" in
  with | control) ;;
  feature | plan)
    if [ -z "$snap" ] || [ ! -d "$snap" ]; then
      echo "efficacy-fixture.sh: snapshot dir required" >&2
      exit 1
    fi
    ;;
  *)
    usage >&2
    exit 1
    ;;
  esac
elif [ "$#" -gt 0 ]; then
  usage >&2
  exit 1
fi

gitcfg=$(mktemp)
trap 'rm -f "$gitcfg"' EXIT
: >"$gitcfg"
export GIT_CONFIG_GLOBAL="$gitcfg"
export GIT_CONFIG_SYSTEM="$gitcfg"
export GIT_CONFIG_NOSYSTEM=1
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_NAMESPACE \
  GIT_OBJECT_DIRECTORY GIT_ALTERNATE_OBJECT_DIRECTORIES \
  GIT_CONFIG_COUNT GIT_CONFIG_PARAMETERS

replay() {
  if [ -e "$dest" ]; then
    chmod -R u+w "$dest" || true
  fi
  if [ -e "$origin" ]; then
    chmod -R u+w "$origin" || true
  fi
  rm -rf "$dest" "$origin"
  mkdir -p "$dest"
  git -C "$dest" -c init.defaultBranch=main -c extensions.objectFormat=sha1 init -b main >/dev/null
  export GIT_AUTHOR_NAME=Fixture
  export GIT_AUTHOR_EMAIL=fixture@example.com
  export GIT_COMMITTER_NAME=Fixture
  export GIT_COMMITTER_EMAIL=fixture@example.com
  while IFS=$'\t' read -r when subject step || [ -n "${when:-}" ]; do
    [ -z "${when:-}" ] && continue
    case "$when" in
    \#*) continue ;;
    esac
    overlay="$src/steps/$step"
    if [ ! -d "$overlay" ]; then
      echo "efficacy-fixture.sh: missing step $step" >&2
      exit 1
    fi
    # A step's DELETE file lists paths to remove; cp -a cannot delete.
    if [ -f "$overlay/DELETE" ]; then
      while IFS= read -r gone || [ -n "$gone" ]; do
        [ -n "$gone" ] && git -C "$dest" rm -q -- "$gone"
      done <"$overlay/DELETE"
    fi
    tar -C "$overlay" --exclude=./DELETE -cf - . | tar -C "$dest" -xpf -
    git -C "$dest" -c core.autocrlf=false -c core.filemode=true add -A
    GIT_AUTHOR_DATE="$when" GIT_COMMITTER_DATE="$when" \
      git -C "$dest" -c extensions.objectFormat=sha1 -c commit.gpgsign=false \
        -c core.autocrlf=false -c core.filemode=true \
        commit --no-verify -m "$subject" >/dev/null
  done <"$src/manifest.tsv"
  git -c init.defaultBranch=main -c extensions.objectFormat=sha1 init --bare "$origin" >/dev/null
  git -C "$dest" remote add origin "$origin"
  git -C "$dest" push -u origin main >/dev/null
}

install_shim() {
  # Binary built from this tree. A PATH archivist can be stale and would
  # open ~/.archivist. HOME is wiped so a new seed does not see the last run.
  rm -rf "$home_dir/home"
  mkdir -p "$dest/bin" "$dest/.cursor" "$home_dir/home"
  (cd "$root" && go build -o "$home_dir/archivist" ./cmd/archivist)
  sed -e "s|__HOME__|$home_dir/home|" -e "s|__BIN__|$home_dir/archivist|" \
    "$src/scripts/archivist-home" >"$dest/bin/archivist"
  chmod +x "$dest/bin/archivist"
  cat >"$dest/.cursor/mcp.json" <<EOF
{
  "mcpServers": {
    "archivist": {
      "command": "$dest/bin/archivist",
      "args": ["--path", "$dest", "mcp"]
    }
  }
}
EOF
}

# `skills install --cli` pins installed files. The unattended plan skill is
# copied after install, so it is pinned here. Already pinned text is skipped.
pin_cli() {
  [ -d "$dest/.cursor" ] || return 0
  if ! command -v python3 >/dev/null 2>&1; then
    echo "efficacy-fixture.sh: python3 is required to pin the CLI onto ./bin/archivist" >&2
    exit 1
  fi
  python3 - "$dest/.cursor" <<'PY'
import pathlib, re, sys
root = pathlib.Path(sys.argv[1])
pat = re.compile(r"(?<![\w./])archivist\b")
for path in root.rglob("*"):
    if path.suffix not in {".md", ".mdc"}:
        continue
    text = path.read_text()
    updated = pat.sub("./bin/archivist", text)
    if updated != text:
        path.write_text(updated)
PY
}

# Feature and plan arms inherit this through the snapshot's .archivist.json.
set_embed() {
  python3 - "$src/run.json" "$dest/.archivist.json" <<'PY'
import json, sys
run = json.load(open(sys.argv[1]))
cfg = json.load(open(sys.argv[2]))
# commands.log is how a run proves the seed called MCP.
cfg["log_commands"] = True
ollama = cfg.setdefault("ollama", {})
for key in ("embed_model", "embed_num_ctx"):
    if key in run:
        ollama[key] = run[key]
with open(sys.argv[2], "w") as f:
    json.dump(cfg, f, indent=2)
    f.write("\n")
PY
}

restore_snapshot() {
  replay
  mkdir -p "$dest/.archivist" "$dest/bin"
  cp -a "$snap/index.db" "$dest/.archivist/index.db"
  cp -a "$snap/.archivist.json" "$dest/.archivist.json"
  cp -a "$snap/.archivist-install.json" "$dest/.archivist-install.json"
  rm -rf "$dest/.cursor"
  cp -a "$snap/.cursor" "$dest/.cursor"
  cp -a "$snap/bin/archivist" "$dest/bin/archivist"
  chmod +x "$dest/bin/archivist"
}

case "$mode" in
"") replay ;;
control)
  replay
  # Subagents inherit PATH, so this is what stops a bare `archivist` call.
  : >"$dest/.archivist-disabled"
  ;;
with)
  replay
  pre=$(git -C "$dest" log --format=%H --fixed-strings --grep='restate the README transfer section' -n 1)
  if [ -z "$pre" ]; then
    echo "efficacy-fixture.sh: pre-transfer commit not found" >&2
    exit 1
  fi
  git -C "$dest" checkout -B pre-transfer "$pre" >/dev/null
  git -C "$dest" push -u origin pre-transfer >/dev/null
  install_shim
  "$dest/bin/archivist" --path "$dest" init
  set_embed
  "$dest/bin/archivist" --path "$dest" skills install --target cursor --cli ./bin/archivist
  if [ -d "$dest/docs/decisions" ] && [ -z "$(ls -A "$dest/docs/decisions")" ]; then
    rmdir "$dest/docs/decisions"
  fi
  ;;
feature)
  restore_snapshot
  rm -rf "$dest/.cursor/skills/plan-changes"
  ;;
plan)
  restore_snapshot
  mkdir -p "$dest/.cursor/skills/plan-changes"
  cp -a "$src/skills/plan-changes/SKILL.md" "$dest/.cursor/skills/plan-changes/SKILL.md"
  pin_cli
  ;;
esac
