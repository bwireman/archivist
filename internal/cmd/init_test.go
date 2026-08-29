package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bwireman/archivist/internal/config"
)

func TestApplyInitCreatesDecisionDirs(t *testing.T) {
	root := t.TempDir()
	if err := applyInit(root, config.Default()); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{config.DefaultDecisionsDir, config.DefaultGlobalDecisionsDir} {
		if _, err := os.Stat(filepath.Join(root, dir)); err != nil {
			t.Fatalf("%s: %v", dir, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, config.DefaultConfigName)); err != nil {
		t.Fatal(err)
	}
}
