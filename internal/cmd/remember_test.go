package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/store"
)

func TestRememberArchiveWritesOnlyExtra(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	extra := t.TempDir()
	db, err := store.Open(config.StorePath(extra))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Archives = []string{extra}
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	cmd := NewRoot()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{
		"--path", root, "remember",
		"--archive", extra,
		"--type", "decision", "--scope", "repo",
		"--title", "Extra only", "--body", "lives there",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
	primary, err := store.Open(config.StorePath(root))
	if err != nil {
		t.Fatal(err)
	}
	defer primary.Close()
	n, err := primary.RecordCount()
	if err != nil || n != 0 {
		t.Fatalf("primary count %d err=%v", n, err)
	}
	extraDB, err := store.OpenReadOnly(config.StorePath(extra))
	if err != nil {
		t.Fatal(err)
	}
	defer extraDB.Close()
	if n, err := extraDB.RecordCount(); err != nil || n != 1 {
		t.Fatalf("extra count %d err=%v", n, err)
	}
}

func TestRememberMissingArchive(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	missing := t.TempDir()
	cfg := config.Default()
	cfg.Archives = []string{missing}
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	cmd := NewRoot()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{
		"--path", root, "remember",
		"--archive", missing,
		"--type", "decision", "--scope", "repo",
		"--title", "Nope", "--body", "no",
	})
	err := cmd.Execute()
	canon := mustCanonCmd(t, missing)
	if err == nil || (!strings.Contains(err.Error(), missing) && !strings.Contains(err.Error(), canon)) {
		t.Fatalf("err=%v", err)
	}
	if _, statErr := os.Stat(filepath.Join(missing, ".archivist")); !os.IsNotExist(statErr) {
		t.Fatal("remember created .archivist")
	}
}

func TestRememberFlagDocumented(t *testing.T) {
	if newRememberCmd().Flags().Lookup("archive") == nil {
		t.Fatal("missing --archive")
	}
}
