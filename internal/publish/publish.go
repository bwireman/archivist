package publish

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/export"
	"github.com/bwireman/archivist/internal/store"
)

func Publish(repoRoot string, cfg *config.Config, repoDB, homeDB *store.Store, name string) error {
	dest, ok := cfg.Publish.Destinations[name]
	if !ok {
		return fmt.Errorf("unknown publish destination %q", name)
	}
	if len(dest.Command) == 0 {
		return fmt.Errorf("destination %q has empty command", name)
	}
	bundle := filepath.Join(repoRoot, ".archivist", "bundle")
	if err := os.RemoveAll(bundle); err != nil {
		return err
	}
	if err := export.WriteBundle(repoDB, homeDB, bundle); err != nil {
		return err
	}
	args := make([]string, len(dest.Command))
	for i, arg := range dest.Command {
		args[i] = strings.ReplaceAll(arg, "{{bundle}}", bundle)
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
