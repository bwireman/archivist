package skills

import (
	"os"
	"path/filepath"

	"github.com/bwireman/archivist/githooks"
)

const githooksDir = ".githooks"

// InstallGithooks writes .githooks/post-commit from the embedded template.
func InstallGithooks(repoRoot string) error {
	dst := filepath.Join(repoRoot, githooksDir, "post-commit")
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, githooks.PostCommit, 0o755)
}
