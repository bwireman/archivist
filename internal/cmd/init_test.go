package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bwireman/archivist/internal/config"
)

func TestApplyInitCreatesDecisionDirs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	if err := applyInit(root, config.Default()); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{config.DefaultDecisionsDir, config.DefaultArchiveDir} {
		if _, err := os.Stat(filepath.Join(root, dir)); err != nil {
			t.Fatalf("%s: %v", dir, err)
		}
	}
	if _, err := os.Stat(filepath.Join(root, config.DefaultGlobalDecisionsDir)); !os.IsNotExist(err) {
		t.Fatal("init should not create in-repo global docs by default")
	}
	if _, err := os.Stat(filepath.Join(home, ".archivist")); err != nil {
		t.Fatalf("home global dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, config.DefaultConfigName)); err != nil {
		t.Fatal(err)
	}
}

func TestAppendGitignoreInsertsNewline(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".gitignore")
	if err := os.WriteFile(path, []byte("key._"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := appendGitignore(path, ".archivist/\n"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := "key._\n.archivist/\n"
	if string(got) != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
