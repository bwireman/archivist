package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

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
	cfg := config.Default()
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
	if got.RecordCount != 1 {
		t.Fatalf("record_count %d", got.RecordCount)
	}
	if len(got.Extras) != 0 {
		t.Fatalf("extras %+v", got.Extras)
	}
}

func TestStatusPromptsWhenCodeMapIsOld(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	ensureQueryStores(t, root)
	canon, err := config.Canonical(root)
	if err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(config.StorePath(root))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.ReplaceFileMap(store.FileRecord{
		Checkout: canon, Path: "a.go", ContentHash: "h", IndexedAt: time.Now().UTC(),
	}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := db.SetCheckoutMeta(canon, store.MetaCodemapVersion, "1"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	cmd := NewRoot()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--path", root, "status"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Run `archivist index`") {
		t.Fatalf("status:\n%s", stdout.String())
	}
}

func TestStatusFileCountAndIndexedAtAreThisCheckout(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	other := t.TempDir()
	ensureQueryStores(t, root)
	canon, err := config.Canonical(root)
	if err != nil {
		t.Fatal(err)
	}
	otherCanon, err := config.Canonical(other)
	if err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(config.StorePath(root))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RegisterCheckout(otherCanon, config.CheckoutHex(otherCanon)); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Truncate(time.Second)
	otherWhen := now.Add(-2 * time.Hour)
	if err := db.ReplaceFileMap(store.FileRecord{
		Checkout: canon, Path: "a.go", ContentHash: "a", IndexedAt: now,
	}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := db.ReplaceFileMap(store.FileRecord{
		Checkout: otherCanon, Path: "b.go", ContentHash: "b", IndexedAt: otherWhen,
	}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := db.SetCheckoutMeta(canon, store.MetaLastIndexedAt, now.Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	if err := db.SetCheckoutMeta(otherCanon, store.MetaLastIndexedAt, otherWhen.Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	if err := db.SetMeta(store.MetaLastIndexedAt, otherWhen.Format(time.RFC3339)); err != nil {
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
		FileCount     int    `json:"file_count"`
		LastIndexedAt string `json:"last_indexed_at"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.FileCount != 1 {
		t.Fatalf("file_count %d\n%s", got.FileCount, stdout.String())
	}
	if got.LastIndexedAt != now.Format("2006-01-02 15:04:05 UTC") {
		t.Fatalf("last_indexed_at %q want %s\n%s", got.LastIndexedAt, now.Format(time.RFC3339), stdout.String())
	}
}

func TestStatusFallsBackToHomeMetaWhenCheckoutEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	ensureQueryStores(t, root)
	db, err := store.Open(config.StorePath(root))
	if err != nil {
		t.Fatal(err)
	}
	when := time.Now().UTC().Truncate(time.Second)
	if err := db.ReplaceFileMap(store.FileRecord{
		Path: "legacy.go", ContentHash: "h", IndexedAt: when,
	}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := db.SetMeta(store.MetaLastIndexedAt, when.Format(time.RFC3339)); err != nil {
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
		FileCount     int    `json:"file_count"`
		LastIndexedAt string `json:"last_indexed_at"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.FileCount != 0 || got.LastIndexedAt != when.Format("2006-01-02 15:04:05 UTC") {
		t.Fatalf("status %+v\n%s", got, stdout.String())
	}
}

func TestFormatMapPrintsReindexBeforeHits(t *testing.T) {
	out := formatMap(store.MapResult{
		Reindex: "Run `archivist index` to rebuild it.",
		CodeSearch: store.CodeSearch{Symbols: []store.Symbol{{
			FilePath: "a.go", Name: "Search", Kind: "method_declaration", Line: 1,
		}}},
	})
	if !strings.HasPrefix(out, "Run `archivist index` to rebuild it.\n") || !strings.Contains(out, "a.go:1 Search") {
		t.Fatalf("map text:\n%s", out)
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
