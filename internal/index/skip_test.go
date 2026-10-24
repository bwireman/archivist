package index_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/index"
)

func TestSkipRelUsesGitignore(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte("secret.go\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "secret.go"), []byte("package secret\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "keep.go"), []byte("package keep\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	skipSecret, err := index.SkipRel(root, config.Default(), "secret.go")
	if err != nil {
		t.Fatal(err)
	}
	if !skipSecret {
		t.Fatal("secret.go should be skipped")
	}
	skipKeep, err := index.SkipRel(root, config.Default(), "keep.go")
	if err != nil {
		t.Fatal(err)
	}
	if skipKeep {
		t.Fatal("keep.go should be read")
	}
}
