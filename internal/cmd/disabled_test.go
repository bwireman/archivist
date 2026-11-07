package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/config"
)

func TestDisabledMarkerRefusesCommands(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, config.DisabledMarker), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "transfer")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	commands := [][]string{
		{"--path", root, "init"},
		{"--path", root, "search", "x"},
		{"--path", sub, "index"},
		{"--path", sub, "remember", "--type", "rule", "--scope", "repo", "--title", "T", "--body", "B"},
		{"--path", root, "mcp"},
	}
	for _, args := range commands {
		cmd := NewRoot()
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs(args)
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), "archivist is disabled here") {
			t.Fatalf("%v: %v\n%s", args, err, buf.String())
		}
	}
	for _, p := range []string{filepath.Join(root, config.DefaultDataDir), filepath.Join(home, ".archivist")} {
		if _, err := os.Stat(p); !os.IsNotExist(err) {
			t.Fatalf("%s was created", p)
		}
	}

	cmd := NewRoot()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"--path", root, "version"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
}
