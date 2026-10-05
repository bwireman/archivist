# Phased plan-changes

Status: done

## Recommendation

Teach `plan-changes` to write a concrete plan, run an adversarial review, and stop. A later `accept` only sets `Status: accepted`. A later `continue` implements the first pending phase.

Phase 1 is done. HEAD stayed `1cfd175cfe74fe49820427f695a1f83cf64a260d`. No commit. The files on disk include the last review's fixes. The sketches below are the pre-fix text the review attacked.

## Archive

- Consult stays lookup-only. The short gap stays in the always-on ask rule. [Ask the gap always-on; full interviews stay in plan-changes](rec_33bedbb05b5b8d4b70d4) is retired only inside phase 1, after the new decision exists.
- [Agent rules and skills install](rec_ad421afa2e5a5bc25fe9): this checkout installs from `skills/` and `rules/`. `agents-md` and Copilot stay rules-only. Skill hashes are outside the stale-rules check.
- [Typed archive records](rec_179dbf51d8a090fef8d7): `remember` writes SQLite only. A global title slugifies to logical path `docs/global-decisions/<slug>.md` and reuses the id already stored at that path. Import ignores this file because it has no `id:` field. Leave `docs/` uncreated.
- [Review every agent commit before ending the turn](rec_a997429c149f23367516): bugbot plus caveman-review stays on post-commit code diffs. A bugbot result does not satisfy `## Adversarial review`. Phase 1 does not commit.

## Plan file contract

Path: `plans/<slug>.md` at the repo root. No `id:` front matter. Leave the file when the work is done.

```markdown
# <title>

Status: proposed | accepted | rejected | in progress | done

## Recommendation

## Phases

### Phase 1 — <name>
- Status: pending | done
- Outcome:
- Files:
- Steps:
  1. ...
- Verify:
- Sketch:

## Adversarial review
Reviewer: <generalPurpose name and id | self>
```

`## Adversarial review` names `self` or a general read-only subagent name and id. The name bugbot does not satisfy the gate.

A revision is an edit to the recommendation, the file contract, or a phase body. The revision turn sets `Status: proposed` and every phase `Status: pending`, runs the review, appends it, and stops.

These turns may change the top `Status` line and nothing else:

- `accept` or `accepted` while `proposed`: set `Status: accepted`. Run no steps. Stop.
- `reject` or `rejected`: set `Status: rejected`. Leave the file.

`continue` or `next` runs the first pending phase only when `Status` is `accepted` or `in progress`. Any other message while `proposed`, including `continue`, `next`, and `go`, leaves the file proposed. One message runs at most one phase, then stops. Several open files: ask which. Open means `proposed`, or `accepted` / `in progress` with a pending phase.

The turn that writes the plan or appends a review may set `Status` only to `proposed`, and phase lines only to `pending`.

## Loop

1. Interview. Search the archive before asking. Ask only what the archive did not answer. The always-on ask rule keeps the one-question gap.
2. Write `plans/<slug>.md` with `Status: proposed` and pending phases. Each phase has an outcome, files, ordered steps, a verification check, and a sketch where a step is easy to get wrong. The sketch binds that step unless a later message overrides it.
3. Adversarial review. Use a general read-only subagent. When the host has none, answer the checklist and set `Reviewer: self`. The parent appends the findings verbatim and discards reviewer file edits. Phase bodies stay unchanged in that turn.
4. Stop. Leave `Status: proposed` and phase `Status: pending`. Do not call `remember`, `update`, or `retire`.
5. Wait for a later message. `accept` only flips status. `continue` implements one pending phase.

Accept is legal only when `## Adversarial review` names `self` or a general read-only subagent name and id, the name is not bugbot, and the review answers every checklist item for the current phase bodies.

### Attack checklist

1. Which gate can the agent skip and still look done?
2. Which current record contradicts a step?
3. Which file would be written that the plan does not name, or named and not written?
4. Which later message would resume the wrong phase or run more than one?
5. Does the review name its reviewer, and is bugbot absent from this gate?

## Phases

### Phase 1 — Skill and archive in one turn

- Status: done
- Outcome: The installed skill stops before phase 1 on a planning turn. After a later accept, continue implements the first pending phase. The archive uses a new decision id, and the feature describes that loop.
- Files:
  - `skills/plan-changes/SKILL.md` (copied from the sketch)
  - `rules/record.md` and `rules/ask.md` (copied from the sketches)
  - `internal/skills/install_test.go`
  - `README.md` (one sentence)
  - `plans/phased-plan-changes.md` (top `Status` and phase `Status` only, after verify passes)
  - SQLite: remember the decision below, retire `rec_33bedbb05b5b8d4b70d4`, update `rec_a01ba05c2a5c2f8ca0d7` to the feature body below
  - `archivist skills install --target cursor` rewrites `.cursor/rules/archivist-ask.mdc`, `archivist-cite.mdc`, `archivist-consult.mdc`, `archivist-current.mdc`, `archivist-record.mdc`, `archivist-refresh.mdc`; `.cursor/skills/{init-archive,plan-changes,publish-archive,record-decision,record-feature,record-rule,refresh-archive}/SKILL.md`; `.archivist-install.json`; `.githooks/post-commit`
- Untouched: `skills/fs.go`, `rules/fs.go`, `internal/mcp/server.go`, `.cursor/rules/commit-review.mdc`, caveman and simplify skills, `.claude/`, `docs/`. No git commit.
- Steps:
  1. Confirm the top `Status` is `accepted` and this phase is `pending`. If not, stop without edits.
  2. Write `skills/plan-changes/SKILL.md` from the skill sketch.
  3. Write `rules/record.md` and `rules/ask.md` from their sketches.
  4. In `internal/skills/install_test.go`, keep the assertions for `Search the archive before asking` and `always-on ask rule`. Add assertions that the installed plan skill contains `stop before phase 1`, `Status: proposed`, and `one pending phase`, and does not contain `Do not implement in this skill`.
  5. Replace the README plan-changes sentence with: `plan-changes` searches the archive, writes `plans/<slug>.md`, runs an adversarial review, and stops before phase 1. A later accept only sets the plan accepted. A later continue implements one pending phase. Cursor files come from `archivist skills install --target cursor`. Claude files come from `archivist skills install --target claude`. `agents-md` and Copilot stay rules-only.
  6. `get` slug `plan-changes-stops-after-review-then-one-phase-per-later-turn`. If it exists, keep that id and do not `remember`. If it does not, `remember` with scope `global`, type `decision`, the title and body in the decision sketch. Then `retire` `rec_33bedbb05b5b8d4b70d4` with `superseded_by` set to that id. If that row is already superseded by that same id, do not retire again. `update` `rec_a01ba05c2a5c2f8ca0d7` in place to the feature sketch. Do not remember a rule.
  7. Run `archivist skills install --target cursor`.
  8. Run `go test ./internal/skills/ ./internal/mcp/` and `archivist index`. Run `archivist embed --once` and skip it when Ollama is down. Skip `archivist import` when `docs/decisions` and `docs/global-decisions` are absent. Run `archivist export` only as the no-op it is while `records.write_docs` is false. Do not create `docs/`.
- Verify: All of these pass before either status line changes to `done`. `get` the new slug and read an id other than `rec_33bedbb05b5b8d4b70d4`, status `accepted`, source path `docs/global-decisions/plan-changes-stops-after-review-then-one-phase-per-later-turn.md`. `get` `rec_33bedbb05b5b8d4b70d4` shows `superseded` and `superseded_by` equal to the new id. `get` `rec_a01ba05c2a5c2f8ca0d7` contains `stops before phase 1` and does not contain `does not implement`. `.cursor/skills/plan-changes/SKILL.md` matches `skills/plan-changes/SKILL.md`, contains `stop before phase 1`, `Status: proposed`, `one pending phase`, `Search the archive before asking`, and `always-on ask rule`, and does not contain `Do not implement in this skill`. `rules/ask.md` still contains `Ask when unsure` and `short gap`. `rules/record.md` contains `Do not remember a rule for that same topic`. Tests pass. No `.claude/` directory. No `docs/` directory created by this phase. Then set this phase `Status: done` and the top `Status: done`.
- Sketch:

Skill file, copied as `skills/plan-changes/SKILL.md`:

```markdown
---
name: plan-changes
description: >
  Interview the user, write plans/<slug>.md, run an adversarial review, and
  stop before phase 1. A later accept only sets Status: accepted. A later
  continue implements one pending phase and stops. Use when planning work,
  or when the user accepts, revises, or continues a file in plans/.
  The planning turn does not write archive records.
---

# Plan a change

Search the archive before asking. The always-on ask rule handles one unsettled user-facing choice or missing preference. This skill is the full interview. The planning turn stops before phase 1.

## 1. Source the archive

MCP preferred, else CLI. Do not trust chat memory.

1. `search` the topic. Also filter by decision, rule, feature, pitfall, and guide when those matter.
2. `get` any current record that looks like the same topic.
3. `map` the subsystem if you need where it lives.
4. `check --paths` once likely files are known.

Cite record titles and ids in later questions and in the plan. If the archive already settled a choice, treat that as the default and ask whether this change revisits it.

## 2. Ask what is still unknown

Ask only what search did not answer. Cover architecture, requirements, and what to avoid. Use the host's structured-question tool when it has one. Do not proceed to a plan while a blocking question is unanswered.

## 3. Write the plan and stop before phase 1

Write `plans/<slug>.md` with `Status: proposed` and pending phases under `## Phases`. Each phase has an outcome, files, ordered steps, a verification check, and a sketch where a step is easy to get wrong. The sketch binds that step unless a later message overrides it.

Run an adversarial review with a general read-only subagent, or answer the attack checklist with `Reviewer: self` when the host has none. Bugbot does not satisfy this review. Append the findings under `## Adversarial review`. Leave `Status: proposed` and phase `Status: pending`. Do not call `remember`, `update`, or `retire`. Stop.

## 4. Later messages

- `accept` or `accepted` while `proposed`: set `Status: accepted`. Run no steps. Stop.
- `revise`: edit the phase bodies, set `Status: proposed`, review again, stop.
- `reject` or `rejected`: set `Status: rejected`. Stop.
- `continue` or `next`: implement one pending phase only when `Status` is `accepted` or `in progress`. Then stop.
- `continue`, `next`, or `go` while `proposed`: leave the file proposed.

Accept is legal only when the review names `self` or a general read-only subagent name and id, the name is not bugbot, and every checklist item is answered.

Attack checklist: which gate can be skipped and still look done; which current record contradicts a step; which file would be written unnamed or named and skipped; which later message resumes the wrong phase or runs more than one; whether the review names its reviewer and bugbot is absent.

On continue, if the plan's record slug already exists, do not `remember` again. Mark the phase `done` only after its verification check passes. One message runs at most one pending phase.
```

`rules/record.md`, full file:

```markdown
# Record decisions, rules, and features

Scan this conversation for durable knowledge. When a lasting choice, constraint, or capability description appears, write it to the archive in this turn. Do not wait for "remember this." Do not leave it only in chat.

- `decision` — a real choice among alternatives (Context, Decision, Consequences).
- `rule` — must/must-not or should/should-not that future work should follow.
- `feature` — how a capability works, what it connects to, and how to invoke it.

Search first. Update or retire an existing record instead of adding a parallel note. One current document per topic. Keep bodies short: facts, not narration.

Musts that an accepted plan-changes phase assigns to a named decision and the skill stay in those two places. Do not remember a rule for that same topic.

Skip chat transcripts, restatements of records already on file, ephemeral session state (this chat's errors, "MCP is down"), unverified catalogs, and details that live only in code.

Use MCP `remember` / `update` / `retire` or `archivist remember`. These write SQLite only. Scope: `repo` this checkout, `global` the product, `dev` personal. Markdown under `records.repo` / `records.global` (default `~/.archivist`) / `records.dev` is optional import input, not the live archive.
```

`rules/ask.md`, full file:

```markdown
# Ask when unsure

After search, ask before acting when a gap would change the result. Do not guess those gaps. Do not ask about small implementation details.

Ask with a few concrete options when either is true:

- A user-facing choice has real alternatives and no current record settles it.
- A personal or project preference (style, defaults, scope) would change the result and is not already recorded.

If search returns a superseded record and a current one on the same topic, follow the current record.

This rule is the short gap. A full interview — architecture, requirements, what to avoid, then a plan — belongs in the on-demand `plan-changes` skill. After the user answers here, continue the task.

A plan-changes interview stores its decision in the accepted phase the plan names. The planning turn does not remember, update, or retire.

Store the answer once so later sessions on this machine do not ask again. Use scope `dev`: a `decision` when they chose among alternatives, a `rule` when they stated a should or must. Product-wide choices stay scope `global`. Search first; a current dev, repo, or global record on the topic counts as settled.
```

Decision sketch. Title: `Plan-changes stops after review, then one phase per later turn`. Slug and logical source path follow that title: `docs/global-decisions/plan-changes-stops-after-review-then-one-phase-per-later-turn.md`. Scope `global`. Body:

```markdown
## Context
rec_33bedbb05b5b8d4b70d4 kept consult lookup-only and put full interviews in plan-changes, which stopped before any code. This change adds a plan file, an adversarial review, and one implementation phase per later turn.

## Decision
- Consult stays lookup-only.
- The always-on ask rule still handles one unsettled user-facing choice or missing preference.
- Personal answers stay dev scope. Product-wide choices stay global.
- plan-changes writes plans/<slug>.md, runs an adversarial review, and stops with Status: proposed. A later accept only sets Status: accepted. A later continue implements one pending phase and stops.
- The planning turn does not remember, update, or retire.
- The musts for this loop live in this decision and the plan-changes skill.

## Consequences
- rec_33bedbb05b5b8d4b70d4 is superseded by this decision.
- Cursor and Claude receive the skill from skills install. agents-md and Copilot stay rules-only.
- Bugbot remains the post-commit code reviewer.
```

Feature sketch for `rec_a01ba05c2a5c2f8ca0d7`, title unchanged:

```markdown
## Purpose
On-demand skill that interviews, writes a phased plan, runs an adversarial review, and stops before phase 1. After a later accept, continue implements one pending phase.

## Behavior
Template `skills/plan-changes/SKILL.md` is installed by `archivist skills install` for Cursor and Claude (`agents-md` / `copilot` stay rules-only). The agent must search the archive before asking, including decision, rule, feature, pitfall, and guide when those matter, then get, map, and check once files are known. It asks only what the archive did not answer. It writes `plans/<slug>.md` with `Status: proposed`, appends an adversarial review, and stops before phase 1. The planning turn does not remember, update, or retire.

A later accept only sets `Status: accepted`. Continue implements one pending phase when the file is accepted or in progress, then stops. The short gap stays in the always-on ask rule.

## Connects to
- Always-on consult rule (lookup) and ask rule (short gap)
- Write skills `record-decision`, `record-rule`, and `record-feature` only when an accepted phase names a record write
- MCP/CLI: `search`, `get`, `map`, `check`

## Entry points
- Template: `skills/plan-changes/SKILL.md`
- CLI: `archivist skills install`
```

## Adversarial review

Reviewer: generalPurpose subagent [Re-review revised plan](f4001ecd-969f-4868-b95e-413b1b758112)

high: The ask and record sketches still require a same-turn archive write. `rules/ask.md` keeps "After the user answers here, continue the task" and "Store the answer once" after a planning-turn ban that does not override them. `rules/record.md` still says to write a lasting choice in this turn; the new sentence only blocks a rule for musts already stored in the decision and the skill. A planning turn can remember and continue into phase 1. Make the planning-turn ban win for remember, update, and retire, including interview answers, and state that those writes happen only in the accepted phase that names them.

high: Phase verify never checks the new ask sentences, the decision body, or the README plan-changes sentence, and it never checks that HEAD did not move. Today's `rules/ask.md` already contains `Ask when unsure` and `short gap`. Step 6 leaves an existing slug's body untouched. Both status lines can then become done, and a commit in that turn still runs bugbot under rec_a997429c149f23367516. Before either status line changes, require the planning-turn ban in `rules/ask.md` and the decision body, require the new README sentence, `update` the decision when the slug exists, and require HEAD unchanged.

high: `go test ./internal/mcp/` compiles the embed into the test binary. The running MCP process keeps the old binary, so initialize instructions stay the previous `ask.md` and `record.md` (rec_179dbf51d8a090fef8d7). Rebuild the archivist binary that process runs and restart it before verify. Do not treat a passing test as a live instruction update.

high: The skill sketch tells `continue` to run one pending phase, not the first, and `revise` does not set phase lines back to pending or say to ask which file when several are open. The next `continue` skips a revised phase left at `done` and can run a later pending one. Copy the contract into the skill: first pending phase only, revision sets every phase to pending, ask which open file, and `go` does not run phases.

medium: `accept` while proposed sets `Status: accepted` and stops with no review check. Step 1 and verify never read `## Adversarial review`. Accept, then continue, can reach `done` without a qualifying review. The accept bullet and step 1 both stop unless the review names self or a general read-only subagent id, the name is not bugbot, and every checklist item is answered.

medium: rec_ad421afa2e5a5bc25fe9 still says plan-changes is only a full interview and that ask stores a dev-scope record, and no step updates it. rec_33bedbb05b5b8d4b70d4 and rec_a01ba05c2a5c2f8ca0d7 still say the skill does not implement while steps 2–5 write the implementing skill. Update rec_ad421afa2e5a5bc25fe9 in this phase. Remember the new decision, retire rec_33bedbb05b5b8d4b70d4, and update rec_a01ba05c2a5c2f8ca0d7 before rewriting the skill.

medium: Step 1 runs phase 1 only when the top status is `accepted`. The contract and the skill also run `continue` when the status is `in progress`, and that message then stops with no edits. Treat `accepted` and `in progress` the same in step 1.

low: Step 8 updates `.archivist/index.db`, `.archivist/index.db-wal`, and `.archivist/index.db-shm`, and appends `.archivist/commands.log` because `log_commands` is true. The file list does not name them. Name those `.archivist/` writes as command side effects, not extra source files.

check: The planning-turn stop can be skipped. Installed ask and record text still say to continue and to write this turn, and verify accepts today's `rules/ask.md`, a stub decision, an unchanged README, and a new commit, then sets both status lines to done.
check: rec_ad421afa2e5a5bc25fe9 contradicts the new skill and is never updated. rec_33bedbb05b5b8d4b70d4 and rec_a01ba05c2a5c2f8ca0d7 contradict steps 2–5 until step 6.
check: `.archivist/index.db`, `.archivist/index.db-wal`, `.archivist/index.db-shm`, and `.archivist/commands.log` are written and not named. Every listed source file has a write step.
check: `continue` after `revise` skips a phase left at `done` and can run a later pending phase because the skill does not say "first" or reset phase status. `continue` while `in progress` runs nothing because step 1 allows only `accepted`.
check: The reviewer is a generalPurpose subagent id. Bugbot is absent from this gate.
totals: 4 high 3 medium 1 low
