package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/cmdlog"
	"github.com/bwireman/archivist/internal/config"
)

func TestCommandLogCLIRemember(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	cfg := config.Default()
	cfg.LogCommands = true
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}

	cmd := NewRoot()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{
		"--path", root, "remember",
		"--title", "Logged", "--body", "Yes.",
		"--type", "decision", "--scope", "repo",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}

	data, err := os.ReadFile(config.CommandsLogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d\n%s", len(lines), data)
	}
	var in, out cmdlog.Entry
	if err := json.Unmarshal([]byte(lines[0]), &in); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &out); err != nil {
		t.Fatal(err)
	}
	if in.Dir != "in" || in.Source != "cli" || !strings.HasSuffix(in.Command, "remember") {
		t.Fatalf("in: %+v", in)
	}
	if out.Dir != "out" || out.Error != "" {
		t.Fatalf("out: %+v", out)
	}
}

func TestCommandLogOffWritesNothing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	if err := config.Save(root, config.Default()); err != nil {
		t.Fatal(err)
	}

	cmd := NewRoot()
	cmd.SetArgs([]string{
		"--path", root, "remember",
		"--title", "Quiet", "--body", "No log.",
		"--type", "decision", "--scope", "repo",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(config.CommandsLogPath(root)); !os.IsNotExist(err) {
		t.Fatal("expected no commands.log when log_commands is false")
	}
}

func TestCommandLogSkipsVersion(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.LogCommands = true
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}

	cmd := NewRoot()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--path", root, "version"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(config.CommandsLogPath(root)); !os.IsNotExist(err) {
		t.Fatal("version should not write commands.log")
	}
}

func TestArchiveCommandSelection(t *testing.T) {
	root := NewRoot()
	want := map[string]bool{
		"search": true, "status": true, "remember": true, "index": true,
		"version": false, "init": false, "mcp": false, "skills": false,
	}
	for _, c := range root.Commands() {
		if expect, ok := want[c.Name()]; ok && archiveCommand(c) != expect {
			t.Fatalf("%s archiveCommand=%v want %v", c.Name(), archiveCommand(c), expect)
		}
	}
	migrate := root.Commands()
	var records bool
	for _, c := range migrate {
		if c.Name() != "migrate" {
			continue
		}
		for _, child := range c.Commands() {
			if child.Name() == "records" {
				records = archiveCommand(child)
			}
		}
	}
	if !records {
		t.Fatal("migrate records should be logged")
	}
}
