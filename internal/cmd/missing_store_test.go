package cmd

import (
	"bytes"
	"os"
	"strings"
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
	before, err := os.Stat(path)
	if err != nil {
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
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "run archivist init") {
			t.Fatalf("%v: %v\n%s", args, err, buf.String())
		}
	}
	if _, err := os.Stat(config.StorePath(root)); !os.IsNotExist(err) {
		t.Fatal("command created the checkout database")
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(before.ModTime()) || after.Size() != before.Size() {
		t.Fatalf("home store changed mtime %s -> %s size %d -> %d", before.ModTime(), after.ModTime(), before.Size(), after.Size())
	}
}
