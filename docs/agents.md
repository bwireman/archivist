# Agent rules, skills, and this checkout

`archivist skills install --target cursor|claude|agents-md|copilot` (`codex` is an alias for `agents-md`) writes:

- **Rules** (always on): consult the archive, cite a retrieved record that changed the work before the turn ends, ask when a user-facing choice or preference is unsettled, write a short record when a later session would look it up, including an answer after an empty search (skip chat glut), keep the code map current as source changes (`archivist index`), refresh after changes. Each rule is its own file (`rules/consult.md`, `rules/cite.md`, `rules/ask.md`, `rules/record.md`, `rules/current.md`, `rules/refresh.md`). Cursor: `.cursor/rules/archivist-*.mdc`. `agents-md` / `copilot` get a single concatenated file only.
- **Skills** (on demand): `init-archive`, `plan-changes`, `record-decision`, `record-rule`, `record-feature`, `refresh-archive`, `publish-archive`. Write skills are the per-type procedure (search first, short body); the record rule is when to write. `init-archive` scans the repo and embeds the gaps so a new archive is searchable. `plan-changes` searches the archive, writes `.archivist/plans/<slug>.md`, runs an adversarial review, and stops before phase 1. A later accept only sets the plan accepted. A later continue implements the first pending phase. Cursor files come from `archivist skills install --target cursor`. Claude files come from `archivist skills install --target claude`. `agents-md` and Copilot stay rules-only.

Templates live in `rules/` and `skills/` in this repo and are embedded in the CLI. `skills install` uses those shipped templates, so it works in any repo; if the target checkout has its own `rules/` or `skills/`, those override the embedded copies.

Every install also writes `.archivist-install.json` with the CLI version, the binary’s git commit (ldflags `Commit`, otherwise Go’s `vcs.revision`), the host this run installed, and SHA-256 hashes of the rules and skills that run wrote. Cursor hashes the wrapped `.mdc`; other hosts hash the trimmed rule body. `agents-md` and `copilot` record no skill hashes. `--target codex` is an alias for `agents-md`.

Every command compares those installed rule files with the templates this binary would write (a checkout-local `rules/` directory still overrides the embedded copies). When they differ, or the manifest cannot be read, archivist prints a reminder to run `archivist skills install --target <host>` on stderr and continues. No manifest means rules were never installed, so it stays quiet.

## Makefile (this checkout)

```bash
make install    # go install into GOPATH/bin
make build
make test
make import           # upsert typed markdown into SQLite
make index            # code map + git
make embed            # needs Ollama
make export           # no-op unless records.write_docs
make refresh-archive  # import, index, embed, export
```

## CI

Pull requests and pushes to `main` run the Go test suite and `go vet`. Each successful commit to `main` publishes a GitHub pre-release tagged `v<version>-<short-sha>` with a source archive plus compiled CLI binaries for Linux amd64 and Apple Silicon macOS. `scripts/install.sh` installs the newest of those releases. Every file includes a SHA-256 checksum. `build-info.txt` records the CLI version and full commit SHA. Each binary’s `*.build-info.txt` also records the UTC build time stamped into that binary.
