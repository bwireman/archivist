#!/usr/bin/env bash
# Paths for the efficacy fixture. The shim sets HOME to <work>/home/home,
# so the database and command log live there, not in the checkout.
# Source this file. It does not run commands.

efficacy_work() {
  if [ -n "${EFFICACY_WORK:-}" ]; then
    printf '%s\n' "$EFFICACY_WORK"
  elif [ -n "${XDG_CACHE_HOME:-}" ]; then
    printf '%s\n' "$XDG_CACHE_HOME/archivist-efficacy"
  else
    printf '%s\n' "$HOME/.cache/archivist-efficacy"
  fi
}

efficacy_project() {
  realpath "$(efficacy_work)/project"
}

efficacy_home() {
  printf '%s\n' "$(efficacy_work)/home/home"
}

# Same bytes as config.CheckoutHex: SHA-256 of the canonical project path.
efficacy_hex() {
  printf '%s' "$(efficacy_project)" | sha256sum | awk '{print $1}'
}

efficacy_archive_db() {
  printf '%s\n' "$(efficacy_home)/.archivist/archive.db"
}

efficacy_commands_log() {
  printf '%s\n' "$(efficacy_home)/.archivist/checkouts/$(efficacy_hex)/commands.log"
}

efficacy_checkout_config() {
  printf '%s\n' "$(efficacy_home)/.archivist/checkouts/$(efficacy_hex)/config.json"
}
