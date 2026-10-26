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

func TestBuildSkipsReview(t *testing.T) {
	at := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC)
	rep, err := Build(call("review", at, nil, map[string]any{"ok": true}), Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Counts["review"] != 0 {
		t.Fatalf("counts %+v", rep.Counts)
	}
}

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

func TestBuildPairsCallIDWhenLaterSearchFinishesFirst(t *testing.T) {
	base := time.Date(2026, 10, 15, 22, 0, 0, 0, time.UTC)
	stamp := func(at time.Time) string { return at.UTC().Format(time.RFC3339Nano) }
	hit := roundTrip(t, []map[string]any{{
		"id": "rec_beta", "title": "Beta", "source": "fts", "status": "accepted",
	}})
	entries := []cmdlog.Entry{
		{TS: stamp(base), Dir: "in", Source: "mcp", Command: "search", Args: map[string]any{"query": "stuck"}},
		{TS: stamp(base.Add(time.Second)), Dir: "in", ID: "c_a", Source: "mcp", Command: "search", Args: map[string]any{"query": "alpha"}},
		{TS: stamp(base.Add(2 * time.Second)), Dir: "in", ID: "c_b", Source: "mcp", Command: "search", Args: map[string]any{"query": "beta"}},
		{TS: stamp(base.Add(3 * time.Second)), Dir: "out", ID: "c_b", Source: "mcp", Command: "search", Result: hit},
		{TS: stamp(base.Add(4 * time.Second)), Dir: "out", ID: "c_a", Source: "mcp", Command: "search", Result: []any{}},
	}
	rep, err := Build(entries, Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.KeywordOnly) != 1 || rep.KeywordOnly[0].Query != "beta" {
		t.Fatalf("keyword: %+v", rep.KeywordOnly)
	}
	if len(rep.EmptySearches) != 1 || rep.EmptySearches[0].Query != "alpha" {
		t.Fatalf("empty: %+v", rep.EmptySearches)
	}
	if rep.Counts["search"] != 2 {
		t.Fatalf("counts: %+v", rep.Counts)
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

func TestBuildUntilInclusive(t *testing.T) {
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	until := start.Add(time.Hour)
	cite := func(at time.Time, id string) []cmdlog.Entry {
		return call("cite", at, map[string]any{"id": id, "effect": id}, map[string]any{"id": id, "effect": id})
	}
	var entries []cmdlog.Entry
	entries = append(entries, cite(start.Add(-time.Second), "rec_before")...)
	entries = append(entries, cite(start, "rec_start")...)
	entries = append(entries, cite(until, "rec_until")...)
	entries = append(entries, cite(until.Add(time.Nanosecond), "rec_after")...)

	rep, err := Build(entries, Options{After: start, Until: until}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got := citeIDs(rep)
	want := []string{"rec_start", "rec_until"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("cites %v", got)
	}

	open, err := Build(entries, Options{After: start}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(citeIDs(open), ",") != "rec_start,rec_until,rec_after" {
		t.Fatalf("open cites %v", citeIDs(open))
	}

	same, err := Build(cite(start, "rec_only"), Options{After: start, Until: start}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(citeIDs(same), ",") != "rec_only" {
		t.Fatalf("equal bounds cites %v", citeIDs(same))
	}
}

func TestBuildUntilDropsLaterActivity(t *testing.T) {
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	until := start.Add(time.Hour)
	later := until.Add(time.Hour)
	entries := call("cite", start.Add(time.Minute), map[string]any{"id": "rec_used", "effect": "kept the change"}, map[string]any{
		"id": "rec_used", "effect": "kept the change",
	})
	entries = append(entries, call("cite", later, map[string]any{"id": "rec_later", "effect": "after the change"}, map[string]any{
		"id": "rec_later", "effect": "after the change",
	})...)
	entries = append(entries, call("search", later, map[string]any{"query": "unrelated"}, []any{})...)
	cat := fakeCat{
		recs: map[string]*record.Record{
			"rec_used": {ID: "rec_used", Title: "Used", Type: record.TypeFeature, Status: record.StatusAccepted, AppliesTo: []string{"internal/cmd/**"}},
		},
		rules: []*record.Record{
			{ID: "rec_later", Title: "Later rule", Type: record.TypeRule, Status: record.StatusAccepted, AppliesTo: []string{"internal/cmd/**"}},
			{ID: "rec_miss", Title: "Never retrieved", Type: record.TypeRule, Status: record.StatusAccepted, AppliesTo: []string{"internal/cmd/**"}},
		},
	}
	paths := []string{"internal/cmd/foo.go", "internal/cmd/bar.go"}
	bounded, err := Build(entries, Options{
		Since: "abc", After: start, Until: until, Paths: paths, GitNote: noteRev,
	}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(citeIDs(bounded), ",") != "rec_used" {
		t.Fatalf("cites %v", citeIDs(bounded))
	}
	if bounded.Counts["search"] != 0 {
		t.Fatalf("search count %d", bounded.Counts["search"])
	}
	if strings.Join(ruleIDs(bounded.MissedRules), ",") != "rec_later,rec_miss" {
		t.Fatalf("missed %v", ruleIDs(bounded.MissedRules))
	}
	if strings.Join(bounded.ChangedPaths, ",") != strings.Join(paths, ",") {
		t.Fatalf("paths %v", bounded.ChangedPaths)
	}
	text := Format(bounded)
	for _, needle := range []string{"<= timestamp <=", "does not narrow paths", "internal/cmd/bar.go", "rec_used"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("missing %q\n%s", needle, text)
		}
	}
	if strings.Contains(text, "after the change") || strings.Contains(text, "no --until") {
		t.Fatal(text)
	}

	open, err := Build(entries, Options{Since: "abc", After: start, Paths: paths, GitNote: noteRev}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(citeIDs(open), ",") != "rec_used,rec_later" {
		t.Fatalf("open cites %v", citeIDs(open))
	}
	if open.Counts["search"] != 1 {
		t.Fatalf("open search %d", open.Counts["search"])
	}
	if strings.Join(ruleIDs(open.MissedRules), ",") != "rec_miss" {
		t.Fatalf("open missed %v", ruleIDs(open.MissedRules))
	}
	openText := Format(open)
	if !strings.Contains(openText, "no --until, so later activity is included") || !strings.Contains(openText, "after the change") {
		t.Fatal(openText)
	}
}

func TestGitChangedQualifiesPaths(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	tracked := filepath.Join(dir, "tracked.go")
	if err := os.WriteFile(tracked, []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "tracked.go")
	runGit(t, dir, "commit", "-m", "base")
	if err := os.WriteFile(tracked, []byte("package p\n\nfunc F() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	staged := filepath.Join(dir, "staged.go")
	if err := os.WriteFile(staged, []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "staged.go")
	if err := os.WriteFile(filepath.Join(dir, "untracked.go"), []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	join, err := GitChanged(dir, "HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(join.Note, "not an ancestor") {
		t.Fatalf("HEAD note: %s", join.Note)
	}
	if !strings.Contains(join.Note, "untracked files are omitted") || !strings.Contains(join.Note, "--until does not narrow paths") {
		t.Fatalf("note: %s", join.Note)
	}
	got := strings.Join(join.Paths, ",")
	if got != "staged.go,tracked.go" {
		t.Fatalf("dirty paths: %s", got)
	}

	when := time.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	timed, err := GitChanged(dir, when)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(timed.Note, "commits at or after") {
		t.Fatalf("time note: %s", timed.Note)
	}
	if strings.Join(timed.Paths, ",") != "staged.go,tracked.go" {
		t.Fatalf("time paths: %v", timed.Paths)
	}

	if _, err := GitChanged(dir, "HEAD~1..HEAD"); err == nil || !strings.Contains(err.Error(), "revision range") {
		t.Fatalf("range err: %v", err)
	}

	runGit(t, dir, "reset", "--hard")
	runGit(t, dir, "clean", "-fd")
	runGit(t, dir, "checkout", "-b", "side")
	if err := os.WriteFile(filepath.Join(dir, "side.go"), []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "side.go")
	runGit(t, dir, "commit", "-m", "side")
	runGit(t, dir, "checkout", "-")
	other, err := GitChanged(dir, "side")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(other.Note, "not an ancestor of HEAD") {
		t.Fatalf("non-ancestral note: %s", other.Note)
	}
	found := false
	for _, p := range other.Paths {
		if p == "side.go" {
			found = true
		}
	}
	if !found {
		t.Fatalf("non-ancestral paths: %v", other.Paths)
	}
}

func TestParseUntil(t *testing.T) {
	got, err := ParseUntil("2026-10-01T01:00:00Z")
	if err != nil || !got.Equal(time.Date(2026, 10, 1, 1, 0, 0, 0, time.UTC)) {
		t.Fatalf("%v %v", got, err)
	}
	if _, err := ParseUntil(""); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{"HEAD", "HEAD~1..HEAD", "yesterday"} {
		if _, err := ParseUntil(bad); err == nil || !strings.Contains(err.Error(), "RFC3339") {
			t.Fatalf("%s: %v", bad, err)
		}
	}
	start := time.Date(2026, 10, 1, 2, 0, 0, 0, time.UTC)
	if err := ValidateWindow(start, start.Add(-time.Second)); err == nil || !strings.Contains(err.Error(), "before") {
		t.Fatalf("inverted: %v", err)
	}
	if err := ValidateWindow(start, start); err != nil {
		t.Fatal(err)
	}
	if err := ValidateWindow(time.Time{}, start); err != nil {
		t.Fatal(err)
	}
}

func TestBuildVectorOnlyExcluded(t *testing.T) {
	at := time.Date(2026, 10, 4, 18, 0, 0, 0, time.UTC)
	vec := roundTrip(t, []retrieve.Result{{
		Record: &record.Record{ID: "rec_miss", Title: "Reinstall rules", Status: record.StatusAccepted, AppliesTo: []string{"internal/cmd/**"}},
		Source: "vector",
	}})
	entries := call("search", at, map[string]any{"query": "porter"}, vec)
	entries = append(entries, call("get", at.Add(time.Minute), map[string]any{"id": "rec_miss"}, map[string]any{
		"ID": "rec_miss", "Title": "Reinstall rules",
	})...)
	rep, err := Build(entries, Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.VectorOnly) != 1 || rep.VectorOnly[0].Query != "porter" {
		t.Fatalf("vector-only: %+v", rep.VectorOnly)
	}
	if len(rep.FollowGets) != 1 || rep.FollowGets[0].ID != "rec_miss" {
		t.Fatalf("follow: %+v", rep.FollowGets)
	}
	if rep.Retrieved != 1 || rep.Cited != 0 {
		t.Fatalf("cited %d retrieved %d", rep.Cited, rep.Retrieved)
	}

	joinEntries := call("search", at, map[string]any{"query": "porter"}, vec)
	joinEntries = append(joinEntries, call("cite", at.Add(time.Minute), map[string]any{"id": "rec_hit", "effect": "used it"}, map[string]any{
		"id": "rec_hit", "effect": "used it",
	})...)
	cat := fakeCat{
		recs: map[string]*record.Record{
			"rec_hit": {ID: "rec_hit", Title: "Command log", Type: record.TypeFeature, Status: record.StatusAccepted, AppliesTo: []string{"internal/cmd/**"}},
		},
		rules: []*record.Record{
			{ID: "rec_miss", Title: "Reinstall rules", Type: record.TypeRule, Status: record.StatusAccepted, AppliesTo: []string{"internal/cmd/**"}},
		},
	}
	joined, err := Build(joinEntries, Options{Since: "HEAD", Paths: []string{"internal/cmd/foo.go"}}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if len(joined.Overlap) != 1 || joined.Overlap[0].ID != "rec_hit" {
		t.Fatalf("overlap: %+v", joined.Overlap)
	}
	if len(joined.MissedRules) != 1 || joined.MissedRules[0].ID != "rec_miss" {
		t.Fatalf("missed: %+v", joined.MissedRules)
	}
	if joined.Retrieved != 0 || joined.Cited != 1 {
		t.Fatalf("cited %d retrieved %d", joined.Cited, joined.Retrieved)
	}

	wide := make([]string, 31)
	for i := range wide {
		wide[i] = "internal/cmd/f.go"
	}
	wideRep, err := Build(joinEntries, Options{Since: "HEAD", Paths: wide}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if !wideRep.WideDiff || !strings.Contains(Format(wideRep), "overlap is coarse") {
		t.Fatalf("wide: %+v\n%s", wideRep.WideDiff, Format(wideRep))
	}
	narrow := make([]string, 30)
	copy(narrow, wide)
	narrowRep, err := Build(joinEntries, Options{Since: "HEAD", Paths: narrow}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if narrowRep.WideDiff {
		t.Fatal("30 paths should not be a wide diff")
	}
}

func TestFlatSearchCardCountsAndOverlaps(t *testing.T) {
	at := time.Date(2026, 10, 4, 20, 0, 0, 0, time.UTC)
	card := []any{map[string]any{
		"id": "rec_hit", "title": "T", "status": "accepted", "source": "hybrid",
	}}
	entries := call("search", at, map[string]any{"query": "card"}, card)
	entries = append(entries, call("get", at.Add(time.Minute), map[string]any{"id": "rec_hit"}, map[string]any{
		"id": "rec_hit", "title": "T",
	})...)
	cat := fakeCat{recs: map[string]*record.Record{
		"rec_hit": {ID: "rec_hit", Title: "T", Type: record.TypeFeature, Status: record.StatusAccepted, AppliesTo: []string{"internal/cmd/**"}},
	}}
	rep, err := Build(entries, Options{Since: "HEAD", After: at.Add(-time.Second), Paths: []string{"internal/cmd/foo.go"}}, cat)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.EmptySearches) != 0 {
		t.Fatalf("empty searches: %+v", rep.EmptySearches)
	}
	if len(rep.Overlap) != 1 || rep.Overlap[0].ID != "rec_hit" {
		t.Fatalf("overlap: %+v", rep.Overlap)
	}
	if len(rep.FollowGets) != 1 || rep.FollowGets[0].ID != "rec_hit" {
		t.Fatalf("follow: %+v", rep.FollowGets)
	}

	vec := []any{map[string]any{
		"id": "rec_vec", "title": "V", "status": "accepted", "source": "vector",
	}}
	vrep, err := Build(call("search", at, map[string]any{"query": "vec"}, vec), Options{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(vrep.VectorOnly) != 1 || vrep.Retrieved != 0 {
		t.Fatalf("vector card consulted: vector-only %+v retrieved %d", vrep.VectorOnly, vrep.Retrieved)
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

func citeIDs(rep *Report) []string {
	var ids []string
	for _, c := range rep.Cites {
		ids = append(ids, c.ID)
	}
	return ids
}

func ruleIDs(rows []RecordHit) []string {
	var ids []string
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	return ids
}

func call(command string, at time.Time, args, result any) []cmdlog.Entry {
	stamp := at.UTC().Format(time.RFC3339Nano)
	return []cmdlog.Entry{
		{TS: stamp, Dir: "in", Source: "mcp", Command: command, Args: args},
		{TS: stamp, Dir: "out", Source: "mcp", Command: command, Result: result},
	}
}

func TestParseSearchEnvelope(t *testing.T) {
	// bare-array-and-object
	hits, kind := parseSearch([]any{map[string]any{"id": "rec_old"}})
	if kind != "hits" || len(hits) != 1 || hits[0].id != "rec_old" {
		t.Fatalf("bare array %#v %s", hits, kind)
	}
	hits, kind = parseSearch(map[string]any{
		"results": []any{map[string]any{"id": "rec_new"}},
		"links":   []any{},
	})
	if kind != "hits" || len(hits) != 1 || hits[0].id != "rec_new" {
		t.Fatalf("object %#v %s", hits, kind)
	}
	if _, kind = parseSearch(map[string]any{"truncated": true, "bytes": 1}); kind != "truncated" {
		t.Fatalf("truncated %s", kind)
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
