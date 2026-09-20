package githooks

import _ "embed"

//go:embed post-commit
var PostCommit []byte
