package archive

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
)

func TestImportRepoRecord(t *testing.T) {
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
	content := `---
id: rec_abc
type: decision
scope: repo
status: accepted
title: Use SQLite
---

## Decision
Yes.
`
	path := filepath.Join(repo, config.DefaultDecisionsDir, "001-sqlite.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := New(repo, config.Default(), repoDB, homeDB)
	res, err := svc.Import()
	if err != nil {
		t.Fatal(err)
	}
	if res.Imported != 1 {
		t.Fatalf("imported %d", res.Imported)
	}
	rec, ok, err := repoDB.GetRecordByID("rec_abc")
	if err != nil || !ok {
		t.Fatalf("record: ok=%v err=%v", ok, err)
	}
	if rec.Title != "Use SQLite" {
		t.Fatalf("title %q", rec.Title)
	}
	depth, _ := repoDB.QueueDepth()
	if depth != 1 {
		t.Fatalf("queue %d", depth)
	}
}

func TestImportHomeGlobalAndDev(t *testing.T) {
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
	globalDir := config.ArchivistHome()
	if err := os.MkdirAll(globalDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(globalDir, "machine.md"), []byte(`---
id: rec_home
type: decision
scope: global
status: accepted
title: Machine wide
---

Body.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	devDir := config.UserRecordsDir()
	if err := os.MkdirAll(devDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(devDir, "personal.md"), []byte(`---
id: rec_dev
type: decision
scope: dev
status: accepted
title: Personal
---

Note.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := New(repo, config.Default(), repoDB, homeDB)
	res, err := svc.Import()
	if err != nil {
		t.Fatal(err)
	}
	if res.Imported != 2 {
		t.Fatalf("imported %d", res.Imported)
	}
	rec, ok, err := homeDB.GetRecordByID("rec_home")
	if err != nil || !ok {
		t.Fatalf("home global: ok=%v err=%v", ok, err)
	}
	if rec.SourcePath != "global/machine.md" {
		t.Fatalf("path %q", rec.SourcePath)
	}
	dev, ok, err := homeDB.GetRecordByID("rec_dev")
	if err != nil || !ok {
		t.Fatalf("dev: ok=%v err=%v", ok, err)
	}
	if dev.SourcePath != "user/personal.md" {
		t.Fatalf("dev path %q", dev.SourcePath)
	}
}

func TestImportMatchesByIDKeepsSourcePath(t *testing.T) {
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
	rec := &record.Record{
		ID:         "rec_feat",
		Slug:       "embed-queue",
		Type:       record.TypeFeature,
		Scope:      record.ScopeGlobal,
		Title:      "Old title",
		Status:     record.StatusAccepted,
		Body:       "Old body.",
		SourcePath: "docs/global-decisions/embed-queue.md",
	}
	rec.ContentHash = record.ContentHash(rec)
	if err := homeDB.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.Records.Global = config.DefaultGlobalDecisionsDir
	exportPath := filepath.Join(repo, cfg.Records.Export, "records", "global", "feature", "embed-queue.md")
	content := `---
id: rec_feat
type: feature
scope: global
status: accepted
title: Embed queue
---

## Purpose
Drain it.
`
	if err := os.MkdirAll(filepath.Dir(exportPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exportPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	svc := New(repo, cfg, repoDB, homeDB)
	res, err := svc.Import()
	if err != nil {
		t.Fatal(err)
	}
	if res.Updated != 1 {
		t.Fatalf("updated %d imported %d skipped %d", res.Updated, res.Imported, res.Skipped)
	}
	n, _ := homeDB.RecordCount()
	if n != 1 {
		t.Fatalf("record count %d", n)
	}
	got, ok, err := homeDB.GetRecordByID("rec_feat")
	if err != nil || !ok {
		t.Fatal(err)
	}
	if got.SourcePath != "docs/global-decisions/embed-queue.md" {
		t.Fatalf("source_path %q", got.SourcePath)
	}
	if got.Title != "Embed queue" {
		t.Fatalf("title %q", got.Title)
	}
}

func TestImportDoesNotDeleteDBOnly(t *testing.T) {
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
	rec := &record.Record{
		ID:         "rec_only",
		Slug:       "only-db",
		Type:       record.TypeDecision,
		Scope:      record.ScopeRepo,
		Title:      "DB only",
		Status:     record.StatusAccepted,
		Body:       "Yes.",
		SourcePath: "docs/decisions/only-db.md",
	}
	rec.ContentHash = record.ContentHash(rec)
	if err := repoDB.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	svc := New(repo, config.Default(), repoDB, homeDB)
	if _, err := svc.Import(); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := repoDB.GetRecordByID("rec_only"); err != nil || !ok {
		t.Fatalf("record removed: ok=%v err=%v", ok, err)
	}
}
