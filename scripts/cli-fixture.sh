#!/usr/bin/env bash
# Rebuild testdata/cli-project and check `archivist map` against saved output.
# Author, dates, and subjects come from testdata/cli-fixture/manifest.tsv so a
# second run produces the same commits. Git config, hooks, and identity come
# from this script, not the caller's environment. init and index use a HOME
# under the temp dir, so this does not read or write ~/.archivist.
set -euo pipefail

root=$(cd "$(dirname "$0")/.." && pwd)
src="$root/testdata/cli-fixture"
dest="$root/testdata/cli-project"
write_expect=0

usage() {
  cat <<'EOF'
Usage: cli-fixture.sh [--write-expect]

Rebuild testdata/cli-project from testdata/cli-fixture, run archivist init
and index with a throwaway HOME, then diff `archivist map` for each line in
queries.txt against expect/<query>.txt.

--write-expect   replace those expect files with the output just produced
EOF
}

case "${1:-}" in
"") ;;
--write-expect) write_expect=1 ;;
-h | --help)
  usage
  exit 0
  ;;
*)
  echo "cli-fixture.sh: unexpected argument: $1" >&2
  usage >&2
  exit 1
  ;;
esac

if [ "$#" -gt 1 ]; then
  echo "cli-fixture.sh: unexpected argument: $2" >&2
  usage >&2
  exit 1
fi

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
(cd "$root" && go build -o "$work/archivist" ./cmd/archivist)

# Empty config files so ~/.gitconfig, system config, and hooksPath cannot
# change trees or commit hashes. -c below covers the settings that do.
gitcfg="$work/gitconfig"
: >"$gitcfg"
export GIT_CONFIG_GLOBAL="$gitcfg"
export GIT_CONFIG_SYSTEM="$gitcfg"
export GIT_CONFIG_NOSYSTEM=1
unset GIT_DIR GIT_WORK_TREE GIT_INDEX_FILE GIT_NAMESPACE \
  GIT_OBJECT_DIRECTORY GIT_ALTERNATE_OBJECT_DIRECTORIES \
  GIT_CONFIG_COUNT GIT_CONFIG_PARAMETERS

rm -rf "$dest"
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
    echo "cli-fixture.sh: missing step $step" >&2
    exit 1
  fi
  cp -a "$overlay/." "$dest/"
  git -C "$dest" -c core.autocrlf=false -c core.filemode=true add -A
  GIT_AUTHOR_DATE="$when" GIT_COMMITTER_DATE="$when" \
    git -C "$dest" -c extensions.objectFormat=sha1 -c commit.gpgsign=false \
      -c core.autocrlf=false -c core.filemode=true \
      commit --no-verify -m "$subject" >/dev/null
done <"$src/manifest.tsv"

export HOME="$work/home"
mkdir -p "$HOME"
"$work/archivist" --path "$dest" init
"$work/archivist" --path "$dest" index

fail=0
mkdir -p "$src/expect"
while IFS= read -r query || [ -n "${query:-}" ]; do
  [ -z "${query:-}" ] && continue
  case "$query" in
  \#*) continue ;;
  esac
  slug=${query// /-}
  got="$work/$slug.txt"
  "$work/archivist" --path "$dest" map "$query" >"$got"
  expect="$src/expect/$slug.txt"
  if [ "$write_expect" -eq 1 ]; then
    cp "$got" "$expect"
    continue
  fi
  if [ ! -f "$expect" ]; then
    echo "cli-fixture.sh: missing $expect (run with --write-expect)" >&2
    fail=1
    continue
  fi
  if ! diff -u "$expect" "$got"; then
    fail=1
  fi
done <"$src/queries.txt"

exit "$fail"
