package store_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"

	_ "modernc.org/sqlite"
)

func TestOpenExistingKeepsVectorsOnOldSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	old := `
CREATE TABLE records (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL,
    type TEXT NOT NULL,
    scope TEXT NOT NULL,
    title TEXT NOT NULL,
    status TEXT NOT NULL,
    severity TEXT,
    body TEXT NOT NULL,
    source_path TEXT NOT NULL UNIQUE,
    tags TEXT,
    applies_to TEXT,
    supersedes TEXT,
    superseded_by TEXT,
    provenance_commit TEXT,
    content_hash TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE TABLE record_vectors (
    record_id TEXT PRIMARY KEY,
    model TEXT NOT NULL,
    dim INTEGER NOT NULL,
    embedding BLOB NOT NULL
);
CREATE TABLE files (path TEXT PRIMARY KEY, content_hash TEXT NOT NULL, package_name TEXT, indexed_at TEXT NOT NULL);
CREATE TABLE symbols (id INTEGER PRIMARY KEY AUTOINCREMENT, file_path TEXT NOT NULL, name TEXT NOT NULL, kind TEXT NOT NULL, line INTEGER NOT NULL, doc_line TEXT, exported INTEGER NOT NULL DEFAULT 0);
CREATE TABLE symbol_edges (id INTEGER PRIMARY KEY AUTOINCREMENT, from_file TEXT NOT NULL, to_path TEXT NOT NULL, edge_type TEXT NOT NULL);
CREATE TABLE commits (hash TEXT PRIMARY KEY, subject TEXT, body TEXT, author TEXT, authored_at TEXT, indexed_at TEXT NOT NULL);
CREATE TABLE record_links (kind TEXT NOT NULL, from_id TEXT NOT NULL DEFAULT '', to_root TEXT NOT NULL, to_id TEXT NOT NULL DEFAULT '', PRIMARY KEY (kind, from_id, to_root, to_id));
CREATE TABLE meta (key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE embed_queue (record_id TEXT PRIMARY KEY, text_hash TEXT NOT NULL, enqueued_at TEXT NOT NULL, attempts INTEGER NOT NULL DEFAULT 0, last_error TEXT);
`
	if _, err := db.Exec(old); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := db.Exec(`INSERT INTO records (id, slug, type, scope, title, status, body, source_path, content_hash, created_at, updated_at)
VALUES ('rec_old', 'old', 'decision', 'global', 'Old', 'accepted', 'body', 'global/old.md', 'hash', ?, ?)`, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO record_vectors (record_id, model, dim, embedding) VALUES ('rec_old', 'm', 1, x'00000000')`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	st, err := store.OpenExisting(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	emb, model, ok, err := st.GetRecordVector("rec_old")
	if err != nil || !ok || model != "m" {
		t.Fatalf("vector ok=%v model=%s emb=%v err=%v", ok, model, emb, err)
	}
	rec, ok, err := st.GetRecordByPath("", "global/old.md")
	if err != nil || !ok || rec.ID != "rec_old" {
		t.Fatalf("record %+v ok=%v err=%v", rec, ok, err)
	}
}

func TestMergeTwoCheckoutsKeepSharedKeys(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	a := t.TempDir()
	b := t.TempDir()
	seedLegacy(t, a, "rec_a")
	seedLegacy(t, b, "rec_b")

	home, err := store.Open(config.HomeStorePath())
	if err != nil {
		t.Fatal(err)
	}
	defer home.Close()
	if err := home.MergeCheckout(a); err != nil {
		t.Fatal(err)
	}
	if err := home.MergeCheckout(b); err != nil {
		t.Fatal(err)
	}
	if err := home.MergeCheckout(a); err != nil {
		t.Fatal(err)
	}
	assertCheckout(t, home, a, "rec_a")
	assertCheckout(t, home, b, "rec_b")
	if _, err := os.Stat(config.LegacyIndexPath(a)); !os.IsNotExist(err) {
		t.Fatalf("legacy a still present: %v", err)
	}
	if _, err := os.Stat(config.LegacyIndexPath(b)); !os.IsNotExist(err) {
		t.Fatalf("legacy b still present: %v", err)
	}

	dupRoot := t.TempDir()
	seedLegacy(t, dupRoot, "rec_a")
	if err := home.MergeCheckout(dupRoot); err == nil {
		t.Fatal("duplicate id merged")
	}
	if _, err := os.Stat(config.LegacyIndexPath(dupRoot)); err != nil {
		t.Fatalf("duplicate source removed: %v", err)
	}

	fresh := t.TempDir()
	t.Setenv("HOME", fresh)
	legacyOnly := t.TempDir()
	seedLegacy(t, legacyOnly, "rec_keep")
	before := config.HomeStorePath()
	if _, err := store.OpenReadOnly(before); err == nil {
		t.Fatal("read-only open created the home database")
	}
	if _, err := os.Stat(before); !os.IsNotExist(err) {
		t.Fatalf("home database was created: %v", err)
	}
	if _, err := os.Stat(config.LegacyIndexPath(legacyOnly)); err != nil {
		t.Fatal(err)
	}
}

func seedLegacy(t *testing.T, root, id string) {
	t.Helper()
	st, err := store.Open(config.LegacyIndexPath(root))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rec := &record.Record{
		ID: id, Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Shared", Slug: "shared", Status: record.StatusAccepted,
		Body: "body", SourcePath: "docs/shared.md",
	}
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	if err := st.SetRecordVector(id, "m", []float32{1}); err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := st.ReplaceFileMap(store.FileRecord{
		Path: "same.go", ContentHash: "h", IndexedAt: now,
	}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertCommit(store.CommitRecord{
		Hash: "abc", Subject: "same", AuthoredAt: now, IndexedAt: now,
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.SetMeta(store.MetaLastIndexedAt, now.Format(time.RFC3339)); err != nil {
		t.Fatal(err)
	}
	peer := t.TempDir()
	if err := os.MkdirAll(peer, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := st.InsertLink(root, store.Link{Kind: store.LinkRepo, FromID: id, ToRoot: peer}); err != nil {
		t.Fatal(err)
	}
}

func assertCheckout(t *testing.T, st *store.Store, root, id string) {
	t.Helper()
	rec, ok, err := st.GetRecordByPath(root, "docs/shared.md")
	if err != nil || !ok || rec.ID != id || rec.Slug != "shared" {
		t.Fatalf("record %s: %+v ok=%v err=%v", root, rec, ok, err)
	}
	if _, _, ok, err := st.GetRecordVector(id); err != nil || !ok {
		t.Fatalf("vector %s: ok=%v err=%v", id, ok, err)
	}
	if _, ok, err := st.GetFile(root, "same.go"); err != nil || !ok {
		t.Fatalf("file %s: ok=%v err=%v", root, ok, err)
	}
	hashes, err := st.CommitHashes(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := hashes["abc"]; !ok {
		t.Fatalf("commit missing for %s", root)
	}
	links, err := st.ListLinks()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, link := range links {
		if link.FromCheckout == root && link.FromID == id {
			found = true
		}
	}
	if !found {
		t.Fatalf("link missing for %s in %#v", root, links)
	}
	if _, ok, err := st.GetCheckoutMeta(root, store.MetaLastIndexedAt); err != nil || !ok {
		t.Fatalf("indexed meta %s: ok=%v err=%v", root, ok, err)
	}
}
