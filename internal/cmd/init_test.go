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
	if _, err := os.Stat(filepath.Join(root, config.DefaultDecisionsDir)); err != nil {
		t.Fatalf("%s: %v", config.DefaultDecisionsDir, err)
	}
	if _, err := os.Stat(filepath.Join(root, config.DefaultArchiveDir)); !os.IsNotExist(err) {
		t.Fatal("init should not create docs/archive unless records.write_docs is true")
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

func TestApplyInitWriteDocsCreatesArchiveDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	cfg := config.Default()
	cfg.Records.WriteDocs = true
	if err := applyInit(root, cfg); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, config.DefaultArchiveDir)); err != nil {
		t.Fatalf("docs/archive: %v", err)
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
