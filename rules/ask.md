# Ask when unsure

After search, ask before acting when a gap would change the result. Do not guess those gaps. Do not ask about small implementation details.

Ask with a few concrete options when either is true:

- A user-facing choice has real alternatives and no current record settles it.
- A personal or project preference (style, defaults, scope) would change the result and is not already recorded.

If search returns a superseded record and a current one on the same topic, follow the current record.

This rule is the short gap. A full interview — architecture, requirements, what to avoid, then a plan — belongs in the on-demand `plan-changes` skill. After the user answers here, continue the task.

Store the answer once so later sessions on this machine do not ask again. Use scope `dev`: a `decision` when they chose among alternatives, a `rule` when they stated a should or must. Product-wide choices stay scope `global`. Search first; a current dev, repo, or global record on the topic counts as settled.
