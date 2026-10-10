package archive

import (
	"database/sql"
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
	}, "")
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
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	id2, err := svc.Remember(&record.Record{
		Type:   record.TypeDecision,
		Scope:  record.ScopeRepo,
		Title:  "Shared title",
		Body:   "Second body.",
		Status: record.StatusAccepted,
	}, "")
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
	}, "")
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

func TestRememberWritesTheContextDatabase(t *testing.T) {
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
	extra := t.TempDir()
	extraDB, err := store.Open(config.StorePath(extra))
	if err != nil {
		t.Fatal(err)
	}
	if err := extraDB.Close(); err != nil {
		t.Fatal(err)
	}
	insertOrphan(t, config.StorePath(extra))
	canon, err := config.Canonical(extra)
	if err != nil {
		t.Fatal(err)
	}
	svc := New(repo, config.Default(), repoDB, homeDB)
	svc.ExtraRoots = []string{canon}

	homeBefore, _ := homeDB.RecordCount()
	id, err := svc.Remember(&record.Record{
		Type: record.TypeDecision, Scope: record.ScopeRepo, Title: "Extra fact",
		Body: "about the extra checkout", Status: record.StatusAccepted,
	}, canon)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := repoDB.GetRecordByID(id); err != nil || ok {
		t.Fatalf("primary contains extra id ok=%v err=%v", ok, err)
	}
	if n, _ := homeDB.RecordCount(); n != homeBefore {
		t.Fatalf("home count %d want %d", n, homeBefore)
	}
	if !hasOrphan(t, config.StorePath(extra)) {
		t.Fatal("remember removed orphan embed_queue row")
	}
	ro, err := store.OpenReadOnly(config.StorePath(extra))
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	if _, ok, err := ro.GetRecordByID(id); err != nil || !ok {
		t.Fatalf("extra missing id ok=%v err=%v", ok, err)
	}

	extraBefore, _ := ro.RecordCount()
	id2, err := svc.Remember(&record.Record{
		Type: record.TypeDecision, Scope: record.ScopeRepo, Title: "Primary fact",
		Body: "about this checkout", Status: record.StatusAccepted,
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := repoDB.GetRecordByID(id2); err != nil || !ok {
		t.Fatalf("primary missing id ok=%v err=%v", ok, err)
	}
	if n, _ := ro.RecordCount(); n != extraBefore {
		t.Fatalf("extra count %d want %d", n, extraBefore)
	}

	repoBefore, _ := repoDB.RecordCount()
	homeBefore, _ = homeDB.RecordCount()
	extraBefore, _ = ro.RecordCount()
	for _, scope := range []record.Scope{record.ScopeGlobal, record.ScopeDev} {
		if _, err := svc.Remember(&record.Record{
			Type: record.TypeDecision, Scope: scope, Title: "Wrong target",
			Body: "no", Status: record.StatusAccepted,
		}, canon); err == nil {
			t.Fatalf("scope %s with archive should fail", scope)
		}
	}
	if n, _ := repoDB.RecordCount(); n != repoBefore {
		t.Fatalf("repo count %d", n)
	}
	if n, _ := homeDB.RecordCount(); n != homeBefore {
		t.Fatalf("home count %d", n)
	}
	if n, _ := ro.RecordCount(); n != extraBefore {
		t.Fatalf("extra count %d", n)
	}

	other := t.TempDir()
	otherDB, err := store.Open(config.StorePath(other))
	if err != nil {
		t.Fatal(err)
	}
	if err := otherDB.Close(); err != nil {
		t.Fatal(err)
	}
	otherBefore := 0
	if _, err := svc.Remember(&record.Record{
		Type: record.TypeDecision, Scope: record.ScopeRepo, Title: "Unlisted",
		Body: "no", Status: record.StatusAccepted,
	}, other); err == nil {
		t.Fatal("unlisted archive should fail")
	}
	check, err := store.OpenReadOnly(config.StorePath(other))
	if err != nil {
		t.Fatal(err)
	}
	defer check.Close()
	if n, _ := check.RecordCount(); n != otherBefore {
		t.Fatalf("unlisted count %d", n)
	}
}

func TestRememberRelativeArchiveMatchesConfig(t *testing.T) {
	parent := t.TempDir()
	repo := filepath.Join(parent, "hub")
	extra := filepath.Join(parent, "other")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	extraDB, err := store.Open(config.StorePath(extra))
	if err != nil {
		t.Fatal(err)
	}
	if err := extraDB.Close(); err != nil {
		t.Fatal(err)
	}
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
	canon, err := config.Canonical(extra)
	if err != nil {
		t.Fatal(err)
	}
	svc := New(repo, config.Default(), repoDB, homeDB)
	svc.ExtraRoots = []string{canon}
	id, err := svc.Remember(&record.Record{
		Type: record.TypeDecision, Scope: record.ScopeRepo, Title: "Sibling",
		Body: "about the other checkout", Status: record.StatusAccepted,
	}, "../other")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok, err := repoDB.GetRecordByID(id); err != nil || ok {
		t.Fatalf("primary contains id ok=%v err=%v", ok, err)
	}
	ro, err := store.OpenReadOnly(config.StorePath(extra))
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	if _, ok, err := ro.GetRecordByID(id); err != nil || !ok {
		t.Fatalf("extra missing id ok=%v err=%v", ok, err)
	}
}

func TestUpdateAndRetireStayInExtra(t *testing.T) {
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
	extra := t.TempDir()
	path := config.StorePath(extra)
	extraDB, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	live := &record.Record{
		ID: "rec_live", Slug: "live", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Live", Body: "original", Status: record.StatusAccepted,
		SourcePath: "docs/decisions/live.md",
	}
	if err := extraDB.UpsertRecord(live); err != nil {
		t.Fatal(err)
	}
	if err := extraDB.Close(); err != nil {
		t.Fatal(err)
	}
	insertOrphan(t, path)
	canon, err := config.Canonical(extra)
	if err != nil {
		t.Fatal(err)
	}
	svc := New(repo, config.Default(), repoDB, homeDB)
	svc.ExtraRoots = []string{canon}
	if err := svc.Update("rec_live", func(r *record.Record) error {
		r.Body = "changed"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := repoDB.GetRecordByID("rec_live"); err != nil || ok {
		t.Fatalf("update copied id ok=%v err=%v", ok, err)
	}
	if !hasOrphan(t, path) {
		t.Fatal("update removed orphan embed_queue row")
	}
	if err := svc.Retire("rec_live", "rec_next"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := repoDB.GetRecordByID("rec_live"); err != nil || ok {
		t.Fatalf("retire copied id ok=%v err=%v", ok, err)
	}
	if !hasOrphan(t, path) {
		t.Fatal("retire removed orphan embed_queue row")
	}
	ro, err := store.OpenReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	got, ok, err := ro.GetRecordByID("rec_live")
	if err != nil || !ok {
		t.Fatalf("extra record ok=%v err=%v", ok, err)
	}
	if got.Status != record.StatusSuperseded || got.Body != "changed" {
		t.Fatalf("extra record %+v", got)
	}
}

func TestGetReadsExtraIDOnlyWhenFileIsReadOnly(t *testing.T) {
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
	extra := t.TempDir()
	path := config.StorePath(extra)
	db, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	rec := &record.Record{
		ID: "rec_ro", Slug: "only-extra", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Only extra", Body: "there", Status: record.StatusAccepted,
		SourcePath: "docs/decisions/only-extra.md",
	}
	if err := db.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	ro, err := store.OpenReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	canon, err := config.Canonical(extra)
	if err != nil {
		t.Fatal(err)
	}
	svc := New(repo, config.Default(), repoDB, homeDB)
	svc.Extras = []Extra{{Root: canon, DB: ro}}
	got, err := svc.Get("rec_ro")
	if err != nil {
		t.Fatal(err)
	}
	if got.Title != "Only extra" {
		t.Fatalf("title %q", got.Title)
	}
	if _, err := svc.Get("only-extra"); err == nil {
		t.Fatal("slug only in the extra should be not found")
	}
}

func insertOrphan(t *testing.T, path string) {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(`PRAGMA foreign_keys = OFF`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO embed_queue (record_id, text_hash, enqueued_at, attempts) VALUES ('rec_orphan', 'x', '2026-01-01T00:00:00Z', 0)`); err != nil {
		t.Fatal(err)
	}
}

func hasOrphan(t *testing.T, path string) bool {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM embed_queue WHERE record_id = 'rec_orphan'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n == 1
}
