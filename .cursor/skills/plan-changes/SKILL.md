---
name: plan-changes
description: >
  Interview the user, write .archivist/plans/<slug>.md, run an adversarial review, and
  stop before phase 1. A later accept only sets Status: accepted. A later
  continue implements the first pending phase and stops. Use when planning
  work, or when the user accepts, revises, or continues a file in .archivist/plans/.
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

Write `.archivist/plans/<slug>.md` with `Status: proposed` and pending phases under `## Phases`. Each phase has an outcome, files, ordered steps, a verification check, and a sketch where a step is easy to get wrong. The sketch binds that step unless a later message overrides it.

Run an adversarial review with a general read-only subagent, or answer the attack checklist with `Reviewer: self` when the host has none. Bugbot does not satisfy this review. Append the findings under `## Adversarial review`. Leave `Status: proposed` and phase `Status: pending`. Do not call `remember`, `update`, or `retire`. Stop.

## 4. Later messages

Accept is legal only when `## Adversarial review` names `self` or a general read-only subagent name and id, the name is not bugbot, and every checklist item is answered. `go` does not accept and does not run phases.

- `accept` or `accepted` while `proposed`, and the review qualifies: set `Status: accepted`. Run no steps. Stop.
- `revise`: edit the phase bodies, set `Status: proposed` and every phase `Status: pending`, ask which file when several in `.archivist/plans/` are open, review again, and stop.
- `reject` or `rejected`: set `Status: rejected`. Stop.
- `continue` or `next`: implement the first pending phase only when `Status` is `accepted` or `in progress` and the review qualified. Then stop.
- `continue`, `next`, or `go` while `proposed`: leave the file proposed.

Attack checklist: which gate can be skipped and still look done; which current record contradicts a step; which file would be written unnamed or named and skipped; which later message resumes the wrong phase or runs more than one; whether the review names its reviewer and bugbot is absent.

On continue, if the plan's record slug already exists, `update` that record instead of calling `remember`. Mark the phase `done` only after its verification check passes. One message runs at most one pending phase.
