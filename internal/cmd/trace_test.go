package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bwireman/archivist/internal/cmdlog"
	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
	"github.com/bwireman/archivist/internal/trace"
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

func TestCiteExtraIDWhenReadOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	extra := t.TempDir()
	seedCheckoutRecord(t, extra, &record.Record{
		ID: "rec_cite_extra", Slug: "cite-extra", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Cite extra", Body: "elsewhere", Status: record.StatusAccepted,
		SourcePath: "docs/decisions/cite-extra.md",
	})
	path := config.StorePath(extra)
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	cfg := config.Default()
	cfg.LogCommands = true
	cfg.Archives = []string{extra}
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	ensureQueryStores(t, root)
	cmd := NewRoot()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--path", root, "cite", "rec_cite_extra", "--effect", "used the extra"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
	if !strings.Contains(buf.String(), "rec_cite_extra") {
		t.Fatalf("cite output: %s", buf.String())
	}
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	db, err := store.OpenReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	n, err := db.RecordCount()
	if err != nil || n != 1 {
		t.Fatalf("extra count %d err=%v", n, err)
	}
}

func TestTraceUntilFiltersLaterLog(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	traceGit(t, root, "init")
	cmdPath := filepath.Join(root, "internal", "cmd")
	if err := os.MkdirAll(cmdPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cmdPath, "foo.go"), []byte("package cmd\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	traceGit(t, root, "add", "internal/cmd/foo.go")
	traceGitAt(t, root, "2026-10-01T00:00:00Z", "commit", "-m", "change")
	first := traceGitOut(t, root, "rev-parse", "HEAD")
	if err := os.WriteFile(filepath.Join(cmdPath, "bar.go"), []byte("package cmd\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	traceGit(t, root, "add", "internal/cmd/bar.go")
	traceGitAt(t, root, "2026-10-01T02:00:00Z", "commit", "-m", "later commit")
	if err := os.WriteFile(filepath.Join(cmdPath, "foo.go"), []byte("package cmd\n\nfunc F() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cmdPath, "untracked.go"), []byte("package cmd\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default()
	cfg.LogCommands = true
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	ensureQueryStores(t, root)
	for _, rec := range []*record.Record{
		{ID: "rec_used", Slug: "used", Type: record.TypeFeature, Scope: record.ScopeRepo, Title: "Used", Body: "in window", Status: record.StatusAccepted, AppliesTo: []string{"internal/cmd/**"}, SourcePath: "docs/features/used.md"},
		{ID: "rec_later", Slug: "later", Type: record.TypeRule, Scope: record.ScopeRepo, Title: "Later rule", Body: "after", Status: record.StatusAccepted, AppliesTo: []string{"internal/cmd/**"}, SourcePath: "docs/rules/later.md"},
		{ID: "rec_miss", Slug: "miss", Type: record.TypeRule, Scope: record.ScopeRepo, Title: "Never retrieved", Body: "miss", Status: record.StatusAccepted, AppliesTo: []string{"internal/cmd/**"}, SourcePath: "docs/rules/miss.md"},
	} {
		seedCheckoutRecord(t, root, rec)
	}
	writeTraceLog(t, root,
		traceCite("2026-10-01T00:30:00Z", "rec_used", "kept the change"),
		traceCite("2026-10-01T03:00:00Z", "rec_later", "after the change"),
		traceSearch("2026-10-01T03:00:00Z", "unrelated"),
	)

	run := func(args ...string) (string, error) {
		cmd := NewRoot()
		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)
		cmd.SetArgs(append([]string{"--path", root, "trace"}, args...))
		err := cmd.Execute()
		return buf.String(), err
	}

	out, err := run("--since", first, "--until", "2026-10-01T01:00:00Z")
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	for _, needle := range []string{
		"<= timestamp <=",
		"does not narrow paths",
		"internal/cmd/foo.go",
		"internal/cmd/bar.go",
		"untracked files are omitted",
		"Cites: 1",
		"rec_used",
		"kept the change",
		"Rules matching the diff and absent from the log: 2",
		"rec_later",
		"rec_miss",
		"search 0,",
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("missing %q\n%s", needle, out)
		}
	}
	if strings.Contains(out, "after the change") || strings.Contains(out, "no --until") || strings.Contains(out, "untracked.go") {
		t.Fatal(out)
	}

	js, err := run("--since", first, "--until", "2026-10-01T01:00:00Z", "--json")
	if err != nil {
		t.Fatalf("%v\n%s", err, js)
	}
	var rep trace.Report
	if err := json.Unmarshal([]byte(js), &rep); err != nil {
		t.Fatalf("%v\n%s", err, js)
	}
	if len(rep.Cites) != 1 || rep.Cites[0].ID != "rec_used" || rep.Cites[0].Effect != "kept the change" {
		t.Fatalf("json cites %+v", rep.Cites)
	}
	if rep.Until.IsZero() || len(rep.MissedRules) != 2 || len(rep.ChangedPaths) != 2 {
		t.Fatalf("json until %s missed %d paths %v", rep.Until, len(rep.MissedRules), rep.ChangedPaths)
	}
	if !strings.Contains(rep.GitNote, "does not narrow paths") {
		t.Fatalf("git note %s", rep.GitNote)
	}

	open, err := run("--since", first)
	if err != nil {
		t.Fatalf("%v\n%s", err, open)
	}
	for _, needle := range []string{"no --until, so later activity is included", "after the change", "Cites: 2", "search 1,", "Rules matching the diff and absent from the log: 1"} {
		if !strings.Contains(open, needle) {
			t.Fatalf("missing %q\n%s", needle, open)
		}
	}

	all, err := run("--all", "--until", "2026-10-01T01:00:00Z")
	if err != nil {
		t.Fatalf("%v\n%s", err, all)
	}
	if !strings.Contains(all, "timestamp <= 2026-10-01T01:00:00Z") || !strings.Contains(all, "Cites: 1") || strings.Contains(all, "after the change") || strings.Contains(all, "Since:") {
		t.Fatal(all)
	}

	if _, err := run("--until", "HEAD"); err == nil || !strings.Contains(err.Error(), "RFC3339") {
		t.Fatalf("rev until: %v", err)
	}
	if _, err := run("--since", first, "--until", "2026-09-01T00:00:00Z"); err == nil || !strings.Contains(err.Error(), "before") {
		t.Fatalf("inverted: %v", err)
	}
	if _, err := run("--since", first+"..HEAD"); err == nil || !strings.Contains(err.Error(), "revision range") {
		t.Fatalf("range: %v", err)
	}
	if _, err := run("--until", "2000-01-01T00:00:00Z"); err == nil || !strings.Contains(err.Error(), "before") {
		t.Fatalf("default window inverted: %v", err)
	}
}

func traceCite(at, id, effect string) []cmdlog.Entry {
	return []cmdlog.Entry{
		{TS: at, Dir: "in", Source: "mcp", Command: "cite", Args: map[string]any{"id": id, "effect": effect}},
		{TS: at, Dir: "out", Source: "mcp", Command: "cite", Result: map[string]any{"id": id, "effect": effect}},
	}
}

func traceSearch(at, query string) []cmdlog.Entry {
	return []cmdlog.Entry{
		{TS: at, Dir: "in", Source: "mcp", Command: "search", Args: map[string]any{"query": query}},
		{TS: at, Dir: "out", Source: "mcp", Command: "search", Result: []any{}},
	}
}

func writeTraceLog(t *testing.T, root string, groups ...[]cmdlog.Entry) {
	t.Helper()
	if err := os.MkdirAll(config.DataDir(root), 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	for _, group := range groups {
		for _, e := range group {
			if err := enc.Encode(e); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := os.WriteFile(config.CommandsLogPath(root), buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func traceGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	traceGitAt(t, dir, "", args...)
}

func traceGitAt(t *testing.T, dir, when string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	if when != "" {
		cmd.Env = append(cmd.Env, "GIT_AUTHOR_DATE="+when, "GIT_COMMITTER_DATE="+when)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func traceGitOut(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git %s: %v", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out))
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
