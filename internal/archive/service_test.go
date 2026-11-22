package archive

import (
	"os"
	"path/filepath"
	"strings"
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
	svc := New(repo, config.Default(), repoDB)
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
	n, err := repoDB.RecordCount()
	if err != nil || n != 1 {
		t.Fatalf("record count %d err=%v", n, err)
	}
	stored, ok, err := repoDB.GetRecordByID(id)
	if err != nil || !ok {
		t.Fatalf("stored ok=%v err=%v", ok, err)
	}
	if stored.Checkout != "" {
		t.Fatalf("checkout %q", stored.Checkout)
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
	svc := New(repo, config.Default(), repoDB)
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
	svc := New(repo, cfg, repoDB)
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

func TestRememberWritesTheCheckoutColumn(t *testing.T) {
	repo := t.TempDir()
	repoDB, err := store.Open(filepath.Join(t.TempDir(), "archive.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repoDB.Close() })
	extra := t.TempDir()
	canon, err := config.Canonical(extra)
	if err != nil {
		t.Fatal(err)
	}
	if err := repoDB.RegisterCheckout(canon, config.CheckoutHex(canon)); err != nil {
		t.Fatal(err)
	}
	svc := New(repo, config.Default(), repoDB)

	id, err := svc.Remember(&record.Record{
		Type: record.TypeDecision, Scope: record.ScopeRepo, Title: "Extra fact",
		Body: "about the extra checkout", Status: record.StatusAccepted,
	}, canon)
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := repoDB.GetRecordByID(id)
	if err != nil || !ok || got.Checkout != canon {
		t.Fatalf("extra row %+v ok=%v err=%v", got, ok, err)
	}

	id2, err := svc.Remember(&record.Record{
		Type: record.TypeDecision, Scope: record.ScopeRepo, Title: "Primary fact",
		Body: "about this checkout", Status: record.StatusAccepted,
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err = repoDB.GetRecordByID(id2)
	if err != nil || !ok || got.Checkout != repo {
		t.Fatalf("primary row %+v ok=%v err=%v", got, ok, err)
	}

	before, _ := repoDB.RecordCount()
	for _, scope := range []record.Scope{record.ScopeGlobal, record.ScopeDev} {
		if _, err := svc.Remember(&record.Record{
			Type: record.TypeDecision, Scope: scope, Title: "Wrong target",
			Body: "no", Status: record.StatusAccepted,
		}, canon); err == nil {
			t.Fatalf("scope %s with archive should fail", scope)
		}
	}
	other := t.TempDir()
	if _, err := svc.Remember(&record.Record{
		Type: record.TypeDecision, Scope: record.ScopeRepo, Title: "Unlisted",
		Body: "no", Status: record.StatusAccepted,
	}, other); err == nil {
		t.Fatal("unlisted archive should fail")
	}
	if n, _ := repoDB.RecordCount(); n != before {
		t.Fatalf("count %d want %d", n, before)
	}
}

func TestRememberRelativeArchiveMatchesConfig(t *testing.T) {
	parent := t.TempDir()
	repo := filepath.Join(parent, "hub")
	extra := filepath.Join(parent, "other")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	repoDB, err := store.Open(filepath.Join(t.TempDir(), "archive.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repoDB.Close() })
	canon, err := config.Canonical(extra)
	if err != nil {
		t.Fatal(err)
	}
	if err := repoDB.RegisterCheckout(canon, config.CheckoutHex(canon)); err != nil {
		t.Fatal(err)
	}
	svc := New(repo, config.Default(), repoDB)
	id, err := svc.Remember(&record.Record{
		Type: record.TypeDecision, Scope: record.ScopeRepo, Title: "Sibling",
		Body: "about the other checkout", Status: record.StatusAccepted,
	}, "../other")
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := repoDB.GetRecordByID(id)
	if err != nil || !ok || got.Checkout != canon {
		t.Fatalf("row %+v ok=%v err=%v", got, ok, err)
	}
}

func TestUpdateAndRetireStayOnTheSameRow(t *testing.T) {
	repo := t.TempDir()
	repoDB, err := store.Open(filepath.Join(t.TempDir(), "archive.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repoDB.Close() })
	extra := t.TempDir()
	canon, err := config.Canonical(extra)
	if err != nil {
		t.Fatal(err)
	}
	live := &record.Record{
		ID: "rec_live", Slug: "live", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Live", Body: "original", Status: record.StatusAccepted,
		SourcePath: "docs/decisions/live.md", Checkout: canon,
	}
	if err := repoDB.UpsertRecord(live); err != nil {
		t.Fatal(err)
	}
	svc := New(repo, config.Default(), repoDB)
	if err := svc.Update("rec_live", func(r *record.Record) error {
		r.Body = "changed"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Retire("rec_live", "rec_next"); err != nil {
		t.Fatal(err)
	}
	got, ok, err := repoDB.GetRecordByID("rec_live")
	if err != nil || !ok {
		t.Fatalf("record ok=%v err=%v", ok, err)
	}
	if got.Status != record.StatusSuperseded || got.Body != "changed" || got.Checkout != canon {
		t.Fatalf("record %+v", got)
	}
	if got.ID != live.ID || got.SourcePath != live.SourcePath {
		t.Fatalf("update changed identity %+v", got)
	}
}

func TestGetReadsExtraIDOnlyWhenFileIsReadOnly(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
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
	svc := New(repo, config.Default(), repoDB)
	_, err = svc.Get("rec_ro")
	if err == nil {
		t.Fatal("record in another file should be not found")
	}
}

func TestPatchTextReplacesAppliesTo(t *testing.T) {
	rec := &record.Record{AppliesTo: []string{"old/**"}}
	PatchText(rec, "", "", "", nil)
	if len(rec.AppliesTo) != 1 || rec.AppliesTo[0] != "old/**" {
		t.Fatalf("empty applies_to changed the list: %#v", rec.AppliesTo)
	}
	PatchText(rec, "Title", "", "", []string{"internal/store/**", "internal/cmd/link.go"})
	if rec.Title != "Title" || len(rec.AppliesTo) != 2 || rec.AppliesTo[1] != "internal/cmd/link.go" {
		t.Fatalf("patch %#v", rec)
	}
}

func TestSlugLookupStaysInCheckout(t *testing.T) {
	a := t.TempDir()
	b := t.TempDir()
	db, err := store.Open(filepath.Join(t.TempDir(), "archive.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, row := range []struct {
		id, checkout, body string
	}{
		{"rec_a", a, "here"},
		{"rec_b", b, "there"},
	} {
		rec := &record.Record{
			ID: row.id, Slug: "shared", Type: record.TypeDecision, Scope: record.ScopeRepo,
			Title: row.id, Status: record.StatusAccepted, Body: row.body,
			SourcePath: "docs/decisions/" + row.id + ".md", Checkout: row.checkout,
		}
		if err := db.UpsertRecord(rec); err != nil {
			t.Fatal(err)
		}
	}
	svc := New(a, config.Default(), db)
	got, err := svc.Get("shared")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "rec_a" {
		t.Fatalf("id %s checkout %s", got.ID, got.Checkout)
	}
	if err := svc.Update("shared", func(r *record.Record) error {
		r.Body = "changed"
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	other, ok, err := db.GetRecordByID("rec_b")
	if err != nil || !ok || other.Body != "there" {
		t.Fatalf("other body %q ok=%v err=%v", other.Body, ok, err)
	}
	mine, ok, err := db.GetRecordByID("rec_a")
	if err != nil || !ok || mine.Body != "changed" {
		t.Fatalf("updated body %q ok=%v err=%v", mine.Body, ok, err)
	}
	lonely := New(t.TempDir(), config.Default(), db)
	if _, err := lonely.Get("shared"); err == nil {
		t.Fatal("slug in another checkout was returned")
	}
}

func TestSlugLookupErrorsWhenAmbiguous(t *testing.T) {
	repo := t.TempDir()
	db, err := store.Open(filepath.Join(t.TempDir(), "archive.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repoRec := &record.Record{
		ID: "rec_repo", Slug: "shared", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Repo", Status: record.StatusAccepted, Body: "repo",
		SourcePath: "docs/decisions/repo.md", Checkout: repo,
	}
	globalRec := &record.Record{
		ID: "rec_global", Slug: "shared", Type: record.TypeDecision, Scope: record.ScopeGlobal,
		Title: "Global", Status: record.StatusAccepted, Body: "global",
		SourcePath: "global/shared.md",
	}
	if err := db.UpsertRecord(repoRec); err != nil {
		t.Fatal(err)
	}
	if err := db.UpsertRecord(globalRec); err != nil {
		t.Fatal(err)
	}
	svc := New(repo, config.Default(), db)
	if _, err := svc.Get("shared"); err == nil || !strings.Contains(err.Error(), "matches") {
		t.Fatalf("err %v", err)
	}
	if err := svc.Update("shared", func(r *record.Record) error {
		r.Body = "nope"
		return nil
	}); err == nil {
		t.Fatal("update resolved an ambiguous slug")
	}
	for _, id := range []string{"rec_repo", "rec_global"} {
		got, ok, err := db.GetRecordByID(id)
		if err != nil || !ok || got.Body == "nope" {
			t.Fatalf("%s body %q ok=%v err=%v", id, got.Body, ok, err)
		}
	}
}
