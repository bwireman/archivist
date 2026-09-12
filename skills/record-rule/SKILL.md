# Record a rule

Record must-dos and can't-dos as typed rules.

1. Run `archivist remember --type rule --scope repo --severity must-not --title "..." --body "..." --applies-to "internal/**"`.
2. Set severity to `must`, `must-not`, `should`, or `should-not`.
3. Set `applies_to` globs so `archivist check` can match touched paths.
