// Package githooks embeds the post-commit hook copied into a checkout.
package githooks

import _ "embed"

// PostCommit is the post-commit hook script.
//
//go:embed post-commit
var PostCommit []byte
