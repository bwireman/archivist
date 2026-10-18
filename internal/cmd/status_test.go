package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
)

func TestStatusListsExtraSeparatelyWhenReadOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	extra := t.TempDir()
	seedCheckoutRecord(t, extra, &record.Record{
		ID: "rec_status", Slug: "status-extra", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Status extra", Body: "counted apart", Status: record.StatusAccepted,
		SourcePath: "docs/decisions/status-extra.md",
	})
	path := config.StorePath(extra)
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	cfg := config.Default()
	cfg.Archives = []string{extra}
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	ensureQueryStores(t, root)
	cmd := NewRoot()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--path", root, "status", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, stderr.String())
	}
	var got struct {
		RecordCount int `json:"record_count"`
		Extras      []struct {
			Root        string `json:"root"`
			RecordCount int    `json:"record_count"`
		} `json:"extras"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("%v\n%s", err, stdout.String())
	}
	if got.RecordCount != 0 {
		t.Fatalf("record_count %d includes the extra", got.RecordCount)
	}
	canon := mustCanonCmd(t, extra)
	if len(got.Extras) != 1 || got.Extras[0].Root != canon || got.Extras[0].RecordCount != 1 {
		t.Fatalf("extras %+v", got.Extras)
	}
}

func TestStatusListsMissingAndGiveups(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	ensureQueryStores(t, root)
	db, err := store.Open(config.StorePath(root))
	if err != nil {
		t.Fatal(err)
	}
	rec := &record.Record{
		ID: "rec_gap", Slug: "gap", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "Gap record", Status: record.StatusAccepted, Body: "no vector yet",
		SourcePath: "docs/global-decisions/gap.md",
	}
	if err := db.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	if err := db.RecordGiveup(rec.ID, 5, "boom"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	cmd := NewRoot()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--path", root, "status", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, stderr.String())
	}
	var got struct {
		Missing []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"missing_vectors"`
		Giveups []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"embed_giveups"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("%v\n%s", err, stdout.String())
	}
	if len(got.Missing) != 1 || got.Missing[0].ID != rec.ID || got.Missing[0].Title != rec.Title {
		t.Fatalf("missing %+v", got.Missing)
	}
	if len(got.Giveups) != 1 || got.Giveups[0].ID != rec.ID {
		t.Fatalf("giveups %+v", got.Giveups)
	}
}
