You are an independent reviewer. Do not call tools. Do not use Archivist. Do not read other files. Judge only the spec and the three diffs in this message.

The spec that supersedes the README:

- KEEP-NOTE-ID: update notes.notebook_id. The id does not change.
- WRITE-LEDGER-ROW: insert one ledger row for that same note id.
- The ledger kind is notebook-transfer.

`go test ./transfer` checks the id, the notebook, a note count of 1, and a ledger count of 1. It does not check kind.

The three diffs are separate implementations of `POST /api/notes/{id}/transfer`. Labels are with, plan, and control. Do not guess which one had extra instructions.

Write these sections and nothing else:

## Differences
What the three implementations do differently.

## Quality
For each label, concrete quality problems: correctness against the spec, error handling, extra queries, and comments that only restate the code.

## Issues
Bugs or spec misses, each naming the label. If an implementation matches the spec and the test, say so.
