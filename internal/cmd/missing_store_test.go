package cmd

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/store"
)

func TestMissingCheckoutLeavesHomeUntouched(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	db, err := store.Open(config.HomeStorePath())
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	path := config.HomeStorePath()
	past := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(path, past, past); err != nil {
		t.Fatal(err)
	}

	root := t.TempDir()
	commands := [][]string{
		{"--path", root, "index"},
		{"--path", root, "remember", "--type", "rule", "--scope", "repo", "--title", "T", "--body", "B"},
		{"--path", root, "embed", "--once"},
		{"--path", root, "update", "rec_missing", "--body", "B"},
		{"--path", root, "import"},
	}
	for _, args := range commands {
		cmd := NewRoot()
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs(args)
		_ = cmd.Execute()
	}
	if _, err := os.Stat(config.LegacyIndexPath(root)); !os.IsNotExist(err) {
		t.Fatal("command created a per-checkout index.db")
	}
}
