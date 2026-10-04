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

func TestTraceLoggingOff(t *testing.T) {
	root := t.TempDir()
	cmd := NewRoot()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--path", root, "trace"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "log_commands is false; no commands.log") {
		t.Fatalf("output: %s", buf.String())
	}
}

func TestTraceDoesNotLogItself(t *testing.T) {
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
	cmd.SetArgs([]string{"--path", root, "trace"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
	if _, err := os.Stat(config.CommandsLogPath(root)); !os.IsNotExist(err) {
		t.Fatal("trace should not create commands.log")
	}
	if !strings.Contains(buf.String(), "no commands logged yet") {
		t.Fatalf("output: %s", buf.String())
	}
}

func TestCiteRejectedWhenLoggingOff(t *testing.T) {
	root := t.TempDir()
	if err := config.Save(root, config.Default()); err != nil {
		t.Fatal(err)
	}
	cmd := NewRoot()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--path", root, "cite", "rec_missing", "--effect", "kept it"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "log_commands is false") {
		t.Fatalf("err=%v\n%s", err, buf.String())
	}
	if _, statErr := os.Stat(config.CommandsLogPath(root)); !os.IsNotExist(statErr) {
		t.Fatal("cite wrote a log while log_commands is false")
	}
}

func TestCiteWritesLog(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	cfg := config.Default()
	cfg.LogCommands = true
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}

	remember := NewRoot()
	var buf bytes.Buffer
	remember.SetOut(&buf)
	remember.SetErr(&buf)
	remember.SetArgs([]string{
		"--path", root, "remember",
		"--title", "Cited choice", "--body", "Keep it.",
		"--type", "decision", "--scope", "repo",
	})
	if err := remember.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
	data, err := os.ReadFile(config.CommandsLogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	id := rememberID(t, data)
	cite := NewRoot()
	buf.Reset()
	cite.SetOut(&buf)
	cite.SetErr(&buf)
	cite.SetArgs([]string{"--path", root, "cite", id, "--effect", "kept the choice"})
	if err := cite.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
	data, err = os.ReadFile(config.CommandsLogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"command":"archivist cite"`) || !strings.Contains(string(data), "kept the choice") {
		t.Fatalf("log:\n%s", data)
	}
}

func rememberID(t *testing.T, data []byte) string {
	t.Helper()
	for line := range strings.SplitSeq(strings.TrimSpace(string(data)), "\n") {
		var e cmdlog.Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatal(err)
		}
		if e.Dir != "out" {
			continue
		}
		m, _ := e.Result.(map[string]any)
		id, _ := m["id"].(string)
		if strings.HasPrefix(id, "rec_") {
			return id
		}
	}
	t.Fatalf("no remember id in log:\n%s", data)
	return ""
}
