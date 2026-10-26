package review

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bwireman/archivist/internal/cmdlog"
	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
)

func TestLastReviewIgnoresDueChecksAndFailures(t *testing.T) {
	now := time.Date(2026, 10, 26, 15, 0, 0, 0, time.UTC)
	full := now.Add(-2 * time.Hour)
	entries := []cmdlog.Entry{}
	entries = append(entries, reviewCall("c_due", now.Add(-time.Minute), map[string]any{"due": true}, "")...)
	entries = append(entries, reviewCall("c_cli", now.Add(-2*time.Minute), map[string]any{"flags": map[string]any{"due": "true"}}, "")...)
	entries = append(entries, reviewCall("c_fail", now.Add(-30*time.Minute), nil, "boom")...)
	entries = append(entries, reviewCall("c_full", full, map[string]any{"flags": map[string]any{"json": "true"}}, "")...)
	last := LastReview(entries)
	if !last.Equal(full) {
		t.Fatalf("last %s", last)
	}
	if due(full, now) {
		t.Fatal("a review two hours ago should not be due")
	}
	if due(now.Add(-Interval+time.Second), now) {
		t.Fatal("a review inside the window is due")
	}
	if !due(now.Add(-Interval), now) {
		t.Fatal("a review exactly one interval ago is due")
	}
	if !LastReview(nil).IsZero() {
		t.Fatal("empty log should have no last review")
	}
}

func TestClockLoggingOff(t *testing.T) {
	root := t.TempDir()
	cfg := config.Default()
	rep, err := Clock(root, cfg, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if rep.Due || !rep.LoggingOff {
		t.Fatalf("%+v", rep)
	}
	text := FormatDue(rep)
	if text != "log_commands is false; review has no clock\ndue: false\n" {
		t.Fatalf("format: %q", text)
	}
}

func TestBuildListsEvidence(t *testing.T) {
	since := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	at := since.Add(time.Hour)
	seen := &record.Record{ID: "rec_seen", Type: record.TypeRule, Scope: record.ScopeRepo, Title: "Seen", Status: record.StatusAccepted, AppliesTo: []string{"internal/cmd/**"}}
	miss := &record.Record{ID: "rec_miss", Type: record.TypeRule, Scope: record.ScopeRepo, Title: "Miss", Status: record.StatusAccepted, AppliesTo: []string{"internal/cmd/**"}}
	old := &record.Record{ID: "rec_old", Type: record.TypeRule, Scope: record.ScopeRepo, Title: "Old", Status: record.StatusSuperseded, AppliesTo: []string{"missing/**"}}
	global := &record.Record{ID: "rec_global", Type: record.TypeFeature, Scope: record.ScopeGlobal, Title: "Global", Status: record.StatusAccepted, AppliesTo: []string{"nope/**"}}
	gone := &record.Record{ID: "rec_gone", Type: record.TypeFeature, Scope: record.ScopeGlobal, Title: "Gone", Status: record.StatusAccepted, AppliesTo: []string{"internal/missing/**"}}
	stale := &record.Record{ID: "rec_stale", Type: record.TypeRule, Scope: record.ScopeRepo, Title: "Stale", Status: record.StatusAccepted, AppliesTo: []string{"missing/**"}}
	feat := &record.Record{ID: "rec_feat", Type: record.TypeFeature, Scope: record.ScopeRepo, Title: "Feat", Status: record.StatusAccepted, AppliesTo: []string{"?.go"}}
	near := &record.Record{ID: "rec_near", Type: record.TypeDecision, Scope: record.ScopeRepo, Title: "Near", Status: record.StatusAccepted}
	far := &record.Record{ID: "rec_far", Type: record.TypeDecision, Scope: record.ScopeRepo, Title: "Far", Status: record.StatusAccepted}
	base := &record.Record{ID: "rec_base", Type: record.TypeDecision, Scope: record.ScopeRepo, Title: "Base", Status: record.StatusAccepted}
	plain := &record.Record{ID: "rec_plain", Type: record.TypeDecision, Scope: record.ScopeRepo, Title: "Plain", Status: record.StatusAccepted}

	result := map[string]any{
		"results": []any{
			map[string]any{"id": "rec_seen", "title": "Seen", "source": "fts", "applies_to": []any{"internal/cmd/**"}},
		},
	}
	stamp := at.UTC().Format(time.RFC3339Nano)
	entries := []cmdlog.Entry{
		{ID: "c_search", TS: stamp, Dir: "in", Source: "mcp", Command: "search", Args: map[string]any{"query": "rules"}},
		{ID: "c_search", TS: stamp, Dir: "out", Source: "mcp", Command: "search", Result: result},
	}
	files := []string{"a.go", "b.go", "c.go", "d.go", "e.go", "internal/cmd/review.go"}
	rep, err := Build(Input{
		Since:   since,
		Entries: entries,
		Paths:   []string{"e.go", "internal/cmd/review.go"},
		Files:   files,
		Dirs:    []string{"internal"},
		Records: []*record.Record{seen, miss, old, global, gone, stale, feat, near, far, base, plain},
		Vectors: map[string][]float32{
			"rec_base": {1, 0},
			"rec_near": unitAt(0.85),
			"rec_far":  {0, 1},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.MissedRules) != 1 || rep.MissedRules[0].ID != "rec_miss" {
		t.Fatalf("missed %+v", rep.MissedRules)
	}
	if len(rep.StaleAppliesTo) != 2 || rep.StaleAppliesTo[0].ID != "rec_gone" || rep.StaleAppliesTo[1].ID != "rec_stale" {
		t.Fatalf("stale %+v", rep.StaleAppliesTo)
	}
	if len(rep.Features) != 1 || rep.Features[0].ID != "rec_feat" || !rep.Features[0].Changed || rep.Features[0].FilesOmitted != 1 {
		t.Fatalf("feature %+v", rep.Features)
	}
	if got := rep.Features[0].Files; len(got) != 4 || got[0] != "a.go" || got[3] != "d.go" {
		t.Fatalf("files %+v", got)
	}
	if len(rep.Similar) != 1 || rep.Similar[0].A.ID != "rec_base" || rep.Similar[0].B.ID != "rec_near" {
		t.Fatalf("similar %+v", rep.Similar)
	}
	if rep.Similar[0].Cosine < similarMinCosine {
		t.Fatalf("cosine %v", rep.Similar[0].Cosine)
	}
	if rep.Unembedded != 7 {
		t.Fatalf("unembedded %d", rep.Unembedded)
	}
	text := Format(rep)
	for _, needle := range []string{"Missed rules: 1", "rec_miss Miss", "Stale applies_to: 2", "rec_gone Gone", "rec_stale Stale", "Similar: 1", "Features: 1", "rec_feat Feat changed", "Unembedded: 7"} {
		if !strings.Contains(text, needle) {
			t.Fatalf("format missing %q\n%s", needle, text)
		}
	}
}

func TestBuildSkipsMissesWhenLoggingOffOrGitSkipped(t *testing.T) {
	rule := &record.Record{ID: "rec_miss", Type: record.TypeRule, Scope: record.ScopeRepo, Title: "Miss", Status: record.StatusAccepted, AppliesTo: []string{"internal/cmd/**"}}
	in := Input{
		Since:   time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC),
		Paths:   []string{"internal/cmd/review.go"},
		Records: []*record.Record{rule},
	}
	in.LoggingOff = true
	rep, err := Build(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.MissedRules) != 0 || !rep.LoggingOff {
		t.Fatalf("logging off %+v", rep.MissedRules)
	}
	in.LoggingOff = false
	in.GitSkipped = "not a git repository"
	rep, err = Build(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.MissedRules) != 0 || rep.ChangedPaths != 0 {
		t.Fatalf("git skipped %+v paths %d", rep.MissedRules, rep.ChangedPaths)
	}
}

func TestBuildWideDiffAndSimilarCap(t *testing.T) {
	paths := make([]string, 31)
	for i := range paths {
		paths[i] = "internal/cmd/f.go"
	}
	rule := &record.Record{ID: "rec_miss", Type: record.TypeRule, Scope: record.ScopeRepo, Title: "Miss", Status: record.StatusAccepted, AppliesTo: []string{"internal/cmd/**"}}
	var recs []*record.Record
	vecs := map[string][]float32{}
	for i := range 7 {
		id := "rec_" + string(rune('a'+i))
		recs = append(recs, &record.Record{ID: id, Type: record.TypeDecision, Scope: record.ScopeRepo, Title: id, Status: record.StatusAccepted})
		vecs[id] = []float32{1, 0}
	}
	recs = append(recs, rule)
	rep, err := Build(Input{
		Since:   time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC),
		Paths:   paths,
		Records: recs,
		Vectors: vecs,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.WideDiff || len(rep.MissedRules) != 1 {
		t.Fatalf("wide %v missed %+v", rep.WideDiff, rep.MissedRules)
	}
	if len(rep.Similar) != maxSimilar || rep.SimilarOmitted != 1 {
		t.Fatalf("similar %d omitted %d", len(rep.Similar), rep.SimilarOmitted)
	}
	if rep.Similar[0].A.ID != "rec_a" || rep.Similar[0].B.ID != "rec_b" {
		t.Fatalf("first pair %+v %+v", rep.Similar[0].A, rep.Similar[0].B)
	}
}

func TestListFilesSkipsArchiveDir(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "keep.go"), []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".archivist"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".archivist", "index.db"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	files, err := ListFiles(root, config.Default())
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0] != "keep.go" {
		t.Fatalf("%v", files)
	}
}

func reviewCall(id string, at time.Time, args any, outErr string) []cmdlog.Entry {
	stamp := at.UTC().Format(time.RFC3339Nano)
	return []cmdlog.Entry{
		{ID: id, TS: stamp, Dir: "in", Source: "cli", Command: "archivist review", Args: args},
		{ID: id, TS: stamp, Dir: "out", Source: "cli", Command: "archivist review", Error: outErr},
	}
}

func unitAt(c float64) []float32 {
	return []float32{float32(c), float32(math.Sqrt(1 - c*c))}
}
