// Package skills embeds the on-demand agent skill templates.
package skills

import "embed"

// FS holds the on-demand skill templates shipped with the CLI.
//
//go:embed */SKILL.md
var FS embed.FS
