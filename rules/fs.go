package rules

import "embed"

// FS holds the always-on rule templates shipped with the CLI.
//
//go:embed *.md
var FS embed.FS
