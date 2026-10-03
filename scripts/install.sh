#!/usr/bin/env bash
# Install the newest Archivist binary from GitHub Releases.
# Successful commits to main publish a pre-release; this script installs that
# newest release (or ARCHIVIST_TAG when set).
set -euo pipefail

repo="${ARCHIVIST_REPO:-bwireman/archivist}"
dest="${ARCHIVIST_INSTALL_DIR:-${HOME}/.local/bin}"
tag="${ARCHIVIST_TAG:-}"

usage() {
  cat <<'EOF'
Usage: install.sh

Download the newest Archivist GitHub release built from main, verify its
SHA-256 checksum, and install the archivist binary.

Published binaries: linux/amd64, darwin/arm64.

Environment:
  ARCHIVIST_INSTALL_DIR   install directory (default: ~/.local/bin)
  ARCHIVIST_REPO          GitHub owner/name (default: bwireman/archivist)
  ARCHIVIST_TAG           release tag to install (default: newest release)
EOF
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
  usage
  exit 0
fi
if [ "$#" -gt 0 ]; then
  echo "install.sh: unexpected argument: $1" >&2
  usage >&2
  exit 1
fi

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "install.sh: required command not found: $1" >&2
    exit 1
  fi
}

need curl
need awk
need install

if [[ ! "${repo}" =~ ^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$ ]]; then
  echo "install.sh: ARCHIVIST_REPO must be owner/name" >&2
  exit 1
fi

os="$(uname -s)"
machine="$(uname -m)"
case "${os}:${machine}" in
  Linux:x86_64) asset="archivist_linux_amd64" ;;
  Darwin:arm64) asset="archivist_darwin_arm64" ;;
  *)
    echo "install.sh: no release binary for ${os} ${machine} (published: linux/amd64, darwin/arm64)" >&2
    exit 1
    ;;
esac

if [ -z "${tag}" ]; then
  # /releases/latest omits pre-releases. Main publishes only pre-releases,
  # so the newest item from /releases is the latest main build.
  json="$(
    curl -fsSL \
      -H "Accept: application/vnd.github+json" \
      -H "X-GitHub-Api-Version: 2022-11-28" \
      "https://api.github.com/repos/${repo}/releases?per_page=1"
  )"
  tag="$(awk -F'"' '/"tag_name":/ { print $4; exit }' <<<"${json}")"
  if [ -z "${tag}" ]; then
    echo "install.sh: could not read the latest release tag for ${repo}" >&2
    exit 1
  fi
fi

if [[ ! "${tag}" =~ ^[A-Za-z0-9._+-]+$ ]]; then
  echo "install.sh: refusing release tag: ${tag}" >&2
  exit 1
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "${tmpdir}"' EXIT

base="https://github.com/${repo}/releases/download/${tag}"
curl -fsSL "${base}/${asset}" -o "${tmpdir}/archivist"
curl -fsSL "${base}/${asset}.sha256" -o "${tmpdir}/archivist.sha256"

expected="$(awk 'NR == 1 { print $1; exit }' "${tmpdir}/archivist.sha256")"
if [[ ! "${expected}" =~ ^[0-9a-fA-F]{64}$ ]]; then
  echo "install.sh: checksum file for ${asset} is not a SHA-256 digest" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "${tmpdir}/archivist" | awk '{ print $1; exit }')"
elif command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "${tmpdir}/archivist" | awk '{ print $1; exit }')"
else
  echo "install.sh: need sha256sum or shasum to verify the download" >&2
  exit 1
fi

expected_lc="$(printf '%s' "${expected}" | tr 'A-F' 'a-f')"
actual_lc="$(printf '%s' "${actual}" | tr 'A-F' 'a-f')"
if [ "${actual_lc}" != "${expected_lc}" ]; then
  echo "install.sh: SHA-256 mismatch for ${asset} (${tag})" >&2
  exit 1
fi

mkdir -p "${dest}"
install -m 755 "${tmpdir}/archivist" "${dest}/archivist"

echo "installed ${dest}/archivist from ${tag}"
"${dest}/archivist" version

case ":${PATH}:" in
  *":${dest}:"*) ;;
  *)
    echo "install.sh: ${dest} is not on PATH" >&2
    ;;
esac
