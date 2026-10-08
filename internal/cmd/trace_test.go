package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

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

func TestTraceDefaultsToLastSevenDays(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	cfg.LogCommands = true
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(config.DataDir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	var log bytes.Buffer
	enc := json.NewEncoder(&log)
	for _, at := range []time.Time{time.Now().Add(-10 * 24 * time.Hour), time.Now().Add(-time.Hour)} {
		ts := at.UTC().Format(time.RFC3339Nano)
		for _, e := range []cmdlog.Entry{
			{TS: ts, Dir: "in", Source: "mcp", Command: "search", Args: map[string]any{"query": "q"}},
			{TS: ts, Dir: "out", Source: "mcp", Command: "search", Result: []any{}},
		} {
			if err := enc.Encode(e); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := os.WriteFile(config.CommandsLogPath(root), log.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) (string, error) {
		cmd := NewRoot()
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs(append([]string{"--path", root, "trace"}, args...))
		err := cmd.Execute()
		return buf.String(), err
	}
	out, err := run()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(out, "Window: since ") || !strings.Contains(out, "search 1,") {
		t.Fatalf("default window output: %s", out)
	}
	out, err = run("--all")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if strings.Contains(out, "Window:") || !strings.Contains(out, "search 2,") {
		t.Fatalf("--all output: %s", out)
	}
	if _, err := run("--all", "--since", "HEAD"); err == nil {
		t.Fatal("--all with --since should fail")
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
