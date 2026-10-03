package trace

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bwireman/archivist/internal/cmdlog"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/retrieve"
)

func TestBuildEmpty(t *testing.T) {
	rep, err := Build(nil, Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Cited != 0 || rep.Retrieved != 0 {
		t.Fatalf("%+v", rep)
	}
	text := Format(rep)
	if !strings.Contains(text, "Cited / retrieved: 0/0") {
		t.Fatal(text)
	}
}

func TestBuildDigest(t *testing.T) {
	base := time.Date(2026, 10, 3, 15, 0, 0, 0, time.UTC)
	hit := roundTrip(t, []retrieve.Result{{
		Record: &record.Record{
			ID: "rec_hit", Title: "Use SQLite", Status: record.StatusSuperseded,
			AppliesTo: []string{"internal/cmd/**"},
		},
		Source: "fts",
		Score:  0.016,
	}})
	check := roundTrip(t, map[string]any{
		"Matches": []any{map[string]any{
			"Record":   map[string]any{"ID": "rec_rule", "Title": "Bind SQL", "AppliesTo": []any{"internal/store/**"}},
			"Reason":   "applies_to glob match",
			"Severity": "must",
		}},
	})
	var entries []cmdlog.Entry
	entries = append(entries, call("search", base, map[string]any{"query": "sqlite"}, hit)...)
	entries = append(entries, call("search", base.Add(time.Minute), map[string]any{"query": "nothing"}, []any{})...)
	entries = append(entries, call("search", base.Add(2*time.Minute), map[string]any{"query": "huge"}, map[string]any{
		"truncated": true, "bytes": 99999, "preview": "{",
	})...)
	entries = append(entries, call("get", base.Add(3*time.Minute), map[string]any{"id": "rec_hit"}, map[string]any{
		"ID": "rec_hit", "Title": "Use SQLite",
	})...)
	entries = append(entries, call("get", base.Add(4*time.Minute), map[string]any{"id": "rec_other"}, map[string]any{
		"ID": "rec_other", "Title": "Elsewhere",
	})...)
	entries = append(entries, call("remember", base.Add(5*time.Minute), map[string]any{"title": "After search"}, map[string]any{
		"id": "rec_new",
	})...)
	entries = append(entries, call("remember", base.Add(40*time.Minute), map[string]any{"title": "Later"}, map[string]any{
		"id": "rec_late",
	})...)
	entries = append(entries, call("check", base.Add(6*time.Minute), map[string]any{"paths": "internal/store/store.go"}, check)...)
	entries = append(entries, call("cite", base.Add(7*time.Minute), map[string]any{"id": "rec_hit", "effect": "kept sqlite"}, map[string]any{
		"id": "rec_hit", "effect": "kept sqlite",
	})...)
	entries = append(entries, call("index", base, nil, map[string]any{"files_indexed": 1})...)

	rep, err := Build(entries, Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.EmptySearches) != 1 || rep.EmptySearches[0].Query != "nothing" {
		t.Fatalf("empty: %+v", rep.EmptySearches)
	}
	if len(rep.Incomplete) != 1 || rep.Incomplete[0].Query != "huge" {
		t.Fatalf("incomplete: %+v", rep.Incomplete)
	}
	if len(rep.KeywordOnly) != 1 || len(rep.SupersededTop) != 1 || rep.SupersededTop[0].ID != "rec_hit" {
		t.Fatalf("search quality: keyword %d superseded %+v", len(rep.KeywordOnly), rep.SupersededTop)
	}
	if len(rep.FollowGets) != 1 || rep.FollowGets[0].ID != "rec_hit" {
		t.Fatalf("follow: %+v", rep.FollowGets)
	}
	if len(rep.UnmatchedGets) != 1 || rep.UnmatchedGets[0].ID != "rec_other" {
		t.Fatalf("unmatched: %+v", rep.UnmatchedGets)
	}
	if len(rep.RemembersNoSearch) != 1 || rep.RemembersNoSearch[0].ID != "rec_late" || rep.RemembersNoSearch[0].Title != "Later" {
		t.Fatalf("remembers: %+v", rep.RemembersNoSearch)
	}
	if len(rep.CheckMatches) != 1 || rep.CheckMatches[0].ID != "rec_rule" || rep.CheckMatches[0].Note != "applies_to glob match" {
		t.Fatalf("check: %+v", rep.CheckMatches)
	}
	if rep.Counts["index"] != 0 || rep.Counts["search"] != 3 {
		t.Fatalf("counts: %+v", rep.Counts)
	}
	if rep.Cited != 1 || rep.Retrieved != 3 {
		t.Fatalf("cited %d retrieved %d", rep.Cited, rep.Retrieved)
	}
	text := Format(rep)
	for _, needle := range []string{"Incomplete searches: 1", "Follow-through gets: 1", "Remembers with no prior search: 1", "Cited / retrieved: 1/3"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("missing %q\n%s", needle, text)
		}
	}
}

func TestBuildSinceOverlap(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	path := filepath.Join(dir, "internal", "cmd")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(path, "foo.go")
	if err := os.WriteFile(file, []byte("package cmd\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "internal/cmd/foo.go")
	runGit(t, dir, "commit", "-m", "add foo")
	if err := os.WriteFile(file, []byte("package cmd\n\nfunc F() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	join, err := GitChanged(dir, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if join.Skipped != "" || join.After.IsZero() {
		t.Fatalf("join: %+v", join)
	}
	found := false
	for _, p := range join.Paths {
		if p == "internal/cmd/foo.go" {
			found = true
		}
	}
	if !found {
		t.Fatalf("paths: %v", join.Paths)
	}

	skipped, err := GitChanged(t.TempDir(), "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if skipped.Skipped != "not a git repository" {
		t.Fatalf("skipped: %+v", skipped)
	}

	at := join.After.Add(time.Second)
	hit := roundTrip(t, []retrieve.Result{{
		Record: &record.Record{ID: "rec_hit", Title: "Command log", Status: record.StatusAccepted, AppliesTo: []string{"internal/cmd/**"}},
		Source: "hybrid",
	}})
	entries := call("search", at, map[string]any{"query": "log"}, hit)
	entries = append(entries, call("get", at.Add(time.Minute), map[string]any{"id": "rec_plain"}, map[string]any{
		"ID": "rec_plain", "Title": "No glob",
	})...)
	old := call("search", join.After.Add(-time.Hour), map[string]any{"query": "stale"}, roundTrip(t, []retrieve.Result{{
		Record: &record.Record{ID: "rec_old", Title: "Old", Status: record.StatusAccepted},
		Source: "fts",
	}}))
	entries = append(old, entries...)

	cat := fakeCat{
		recs: map[string]*record.Record{
			"rec_hit":   {ID: "rec_hit", Title: "Command log", Type: record.TypeFeature, Status: record.StatusAccepted, AppliesTo: []string{"internal/cmd/**"}},
			"rec_plain": {ID: "rec_plain", Title: "No glob", Type: record.TypeFeature, Status: record.StatusAccepted},
		},
		rules: []*record.Record{
			{ID: "rec_miss", Title: "Reinstall rules", Type: record.TypeRule, Status: record.StatusAccepted, AppliesTo: []string{"internal/cmd/**"}},
			{ID: "rec_else", Title: "Other", Type: record.TypeRule, Status: record.StatusAccepted, AppliesTo: []string{"docs/**"}},
			{ID: "rec_oldrule", Title: "Retired", Type: record.TypeRule, Status: record.StatusSuperseded, AppliesTo: []string{"internal/cmd/**"}},
		},
	}
	rep, err := Build(entries, Options{Since: "HEAD", After: join.After, Paths: join.Paths}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Overlap) != 1 || rep.Overlap[0].ID != "rec_hit" {
		t.Fatalf("overlap: %+v", rep.Overlap)
	}
	if len(rep.NoGlob) != 1 || rep.NoGlob[0].ID != "rec_plain" {
		t.Fatalf("noglob: %+v", rep.NoGlob)
	}
	if len(rep.MissedRules) != 1 || rep.MissedRules[0].ID != "rec_miss" {
		t.Fatalf("missed: %+v", rep.MissedRules)
	}
	if rep.Retrieved != 2 {
		t.Fatalf("retrieved %d (old search should be outside the window)", rep.Retrieved)
	}
	text := Format(rep)
	if !strings.Contains(text, "Rules matching the diff and absent from the log: 1") || !strings.Contains(text, "internal/cmd/foo.go") {
		t.Fatal(text)
	}
}

func TestPrepareCite(t *testing.T) {
	if _, err := PrepareCite(false, "kept it"); err != ErrCiteDisabled {
		t.Fatalf("got %v", err)
	}
	if _, err := PrepareCite(true, "line\nbreak"); err == nil {
		t.Fatal("expected one-line error")
	}
	got, err := PrepareCite(true, "  kept it  ")
	if err != nil || got != "kept it" {
		t.Fatalf("%q %v", got, err)
	}
}

type fakeCat struct {
	recs  map[string]*record.Record
	rules []*record.Record
}

func (f fakeCat) Record(id string) (*record.Record, error) {
	return f.recs[id], nil
}

func (f fakeCat) Rules() ([]*record.Record, error) {
	return f.rules, nil
}

func call(command string, at time.Time, args, result any) []cmdlog.Entry {
	stamp := at.UTC().Format(time.RFC3339Nano)
	return []cmdlog.Entry{
		{TS: stamp, Dir: "in", Source: "mcp", Command: command, Args: args},
		{TS: stamp, Dir: "out", Source: "mcp", Command: command, Result: result},
	}
}

func roundTrip(t *testing.T, v any) any {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test",
		"GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test",
		"GIT_COMMITTER_EMAIL=test@example.com",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}
