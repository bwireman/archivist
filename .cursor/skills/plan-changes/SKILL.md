---
name: plan-changes
description: Interview the user about architecture, requirements, and constraints, then propose a change plan grounded in Archivist records. Use when planning work, making a design choice, clarifying requirements, or before implementing something with open trade-offs. Do not use to implement, or to write a record (record-decision, record-rule, record-feature).
---

# Plan a change

Do not implement in this skill. Search the archive before asking; ask before assuming.

## 1. Source the archive

MCP preferred, else CLI. Do not trust chat memory.

1. `search` the topic. Also filter `--type decision`, `--type rule`, `--type feature`, `--type pitfall`, and `--type guide` when those matter.
2. `get` any current record that looks like the same topic.
3. `map` the subsystem if you need where it lives.
4. `check --paths` once likely files are known.

Cite record titles and ids in later questions and in the plan. If the archive already settled a choice, treat that as the default and ask whether this change revisits it.

## 2. Ask what is still unknown

Ask only what search did not answer. Cover, as needed:

- **Architecture** — where it lives, what it connects to, alternatives still in play
- **Requirements** — success criteria, scope, what "done" means
- **Avoid** — constraints, pitfalls, must-nots (archive `rule` / `pitfall` plus anything the user adds)

Use the host's structured-question tool if it has one; otherwise ask in chat. Prefer a few concrete options over open-ended essays. Do not proceed to a plan while a blocking question is unanswered.

## 3. Propose the plan

Lead with the recommendation. Include:

- What the archive already decided (and what it did not)
- The chosen approach and rejected alternatives
- Files or packages likely touched (`map` / `feature` `applies_to`)
- Risks and things to avoid
- Whether a lasting `decision` or `rule` should be written after the user confirms

If the user then chooses among alternatives, follow `record-decision`. If they state a must/must-not, follow `record-rule`. Implementation is a later turn.
