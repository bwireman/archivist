package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/review"
	"github.com/bwireman/archivist/internal/store"
)

func TestReviewDueCheckDoesNotResetTheClock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	cfg := config.Default()
	cfg.LogCommands = true
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "keep.go"), []byte("package p\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	repo := openTempStore(t, config.StorePath(root))
	homeDB := openTempStore(t, config.HomeStorePath())
	putRecord(t, repo, &record.Record{
		ID: "rec_stale", Slug: "stale", Type: record.TypeRule, Scope: record.ScopeRepo,
		Title: "Stale", Status: record.StatusAccepted, Body: "gone", SourcePath: "docs/rules/stale.md",
		AppliesTo: []string{"missing/**"},
	})
	putRecord(t, repo, &record.Record{
		ID: "rec_feat", Slug: "feat", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "Feat", Status: record.StatusAccepted, Body: "here", SourcePath: "docs/features/feat.md",
		AppliesTo: []string{"keep.go"},
	})
	putRecord(t, repo, &record.Record{
		ID: "rec_repo", Slug: "repo-near", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Repo near", Status: record.StatusAccepted, Body: "same", SourcePath: "docs/decisions/repo-near.md",
	})
	putRecord(t, homeDB, &record.Record{
		ID: "rec_home", Slug: "home-near", Type: record.TypeDecision, Scope: record.ScopeGlobal,
		Title: "Home near", Status: record.StatusAccepted, Body: "same", SourcePath: "global/home-near.md",
	})
	putRecord(t, homeDB, &record.Record{
		ID: "rec_global", Slug: "global-gap", Type: record.TypeFeature, Scope: record.ScopeGlobal,
		Title: "Global", Status: record.StatusAccepted, Body: "elsewhere", SourcePath: "global/global-gap.md",
		AppliesTo: []string{"nope/**"},
	})
	vec := []float32{1, 0}
	if err := repo.SetRecordVector("rec_repo", config.DefaultEmbedModel, vec); err != nil {
		t.Fatal(err)
	}
	if err := homeDB.SetRecordVector("rec_home", config.DefaultEmbedModel, vec); err != nil {
		t.Fatal(err)
	}
	repo.Close()
	homeDB.Close()

	due := runReview(t, root, "--due", "--json")
	var clock review.DueReport
	if err := json.Unmarshal([]byte(due), &clock); err != nil {
		t.Fatal(err)
	}
	if !clock.Due || clock.LoggingOff {
		t.Fatalf("first clock %+v\n%s", clock, due)
	}
	due = runReview(t, root, "--due", "--json")
	if err := json.Unmarshal([]byte(due), &clock); err != nil {
		t.Fatal(err)
	}
	if !clock.Due {
		t.Fatalf("due check counted as a review: %+v", clock)
	}

	out := runReview(t, root, "--json")
	var rep review.Report
	if err := json.Unmarshal([]byte(out), &rep); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if rep.GitSkipped == "" {
		t.Fatal("expected git skip")
	}
	if len(rep.StaleAppliesTo) != 1 || rep.StaleAppliesTo[0].ID != "rec_stale" {
		t.Fatalf("stale %+v", rep.StaleAppliesTo)
	}
	if len(rep.Features) != 1 || rep.Features[0].ID != "rec_feat" || len(rep.Features[0].Files) != 1 || rep.Features[0].Files[0] != "keep.go" {
		t.Fatalf("features %+v", rep.Features)
	}
	if len(rep.Similar) != 1 || rep.Similar[0].A.ID != "rec_home" || rep.Similar[0].B.ID != "rec_repo" {
		t.Fatalf("similar %+v", rep.Similar)
	}
	for _, card := range rep.StaleAppliesTo {
		if card.ID == "rec_global" {
			t.Fatal("global record listed as stale")
		}
	}

	due = runReview(t, root, "--due", "--json")
	if err := json.Unmarshal([]byte(due), &clock); err != nil {
		t.Fatal(err)
	}
	if clock.Due || clock.LastReview.IsZero() {
		t.Fatalf("full review should clear due: %+v", clock)
	}
}

func TestReviewDueWhenLoggingOff(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	out := runReview(t, root, "--due")
	if out != "log_commands is false; review has no clock\ndue: false\n" {
		t.Fatalf("output: %q", out)
	}
}

func openTempStore(t *testing.T, path string) *store.Store {
	t.Helper()
	db, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func putRecord(t *testing.T, db *store.Store, rec *record.Record) {
	t.Helper()
	if err := db.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
}

func runReview(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := NewRoot()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(append([]string{"--path", root, "review"}, args...))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
	return buf.String()
}
