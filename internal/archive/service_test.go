package archive

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
)

func TestRememberGlobalStoresInHomeDB(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	repoDB, err := store.Open(filepath.Join(t.TempDir(), "repo.db"))
	if err != nil {
		t.Fatal(err)
	}
	homeDB, err := store.Open(filepath.Join(t.TempDir(), "home.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = repoDB.Close()
		_ = homeDB.Close()
	})
	svc := New(repo, config.Default(), repoDB, homeDB)
	id, err := svc.Remember(&record.Record{
		Type:   record.TypeDecision,
		Scope:  record.ScopeGlobal,
		Title:  "Home global",
		Body:   "Yes.",
		Status: record.StatusAccepted,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".archivist", "home-global.md")); !os.IsNotExist(err) {
		t.Fatal("remember should not write markdown under ~/.archivist")
	}
	if _, err := os.Stat(filepath.Join(repo, "docs/global-decisions/home-global.md")); !os.IsNotExist(err) {
		t.Fatal("default global should not write in-repo markdown")
	}
	n, err := homeDB.RecordCount()
	if err != nil || n != 1 {
		t.Fatalf("home record count %d err=%v", n, err)
	}
	rec, err := svc.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if rec.SourcePath != "global/home-global.md" {
		t.Fatalf("source %q", rec.SourcePath)
	}
}

func TestRememberSamePathReusesID(t *testing.T) {
	repo := t.TempDir()
	repoDB, err := store.Open(filepath.Join(t.TempDir(), "repo.db"))
	if err != nil {
		t.Fatal(err)
	}
	homeDB, err := store.Open(filepath.Join(t.TempDir(), "home.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = repoDB.Close()
		_ = homeDB.Close()
	})
	svc := New(repo, config.Default(), repoDB, homeDB)
	id1, err := svc.Remember(&record.Record{
		Type:   record.TypeDecision,
		Scope:  record.ScopeRepo,
		Title:  "Shared title",
		Body:   "First body.",
		Status: record.StatusAccepted,
	})
	if err != nil {
		t.Fatal(err)
	}
	id2, err := svc.Remember(&record.Record{
		Type:   record.TypeDecision,
		Scope:  record.ScopeRepo,
		Title:  "Shared title",
		Body:   "Second body.",
		Status: record.StatusAccepted,
	})
	if err != nil {
		t.Fatal(err)
	}
	if id1 != id2 {
		t.Fatalf("expected reused id, got %s then %s", id1, id2)
	}
	got, err := svc.Get(id1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Body != "Second body." {
		t.Fatalf("body %q", got.Body)
	}
	n, err := repoDB.RecordCount()
	if err != nil || n != 1 {
		t.Fatalf("record count %d err=%v", n, err)
	}
}

func TestRememberGlobalInRepoUsesLogicalPath(t *testing.T) {
	repo := t.TempDir()
	repoDB, err := store.Open(filepath.Join(t.TempDir(), "repo.db"))
	if err != nil {
		t.Fatal(err)
	}
	homeDB, err := store.Open(filepath.Join(t.TempDir(), "home.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = repoDB.Close()
		_ = homeDB.Close()
	})
	cfg := config.Default()
	cfg.Records.Global = config.DefaultGlobalDecisionsDir
	svc := New(repo, cfg, repoDB, homeDB)
	_, err = svc.Remember(&record.Record{
		Type:   record.TypeDecision,
		Scope:  record.ScopeGlobal,
		Title:  "In repo",
		Body:   "Yes.",
		Status: record.StatusAccepted,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repo, config.DefaultGlobalDecisionsDir, "in-repo.md")); !os.IsNotExist(err) {
		t.Fatal("remember should not write in-repo markdown")
	}
	got, err := svc.Get("in-repo")
	if err != nil {
		t.Fatal(err)
	}
	if got.SourcePath != filepath.ToSlash(filepath.Join(config.DefaultGlobalDecisionsDir, "in-repo.md")) {
		t.Fatalf("source %q", got.SourcePath)
	}
}
