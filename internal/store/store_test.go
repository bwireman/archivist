package store_test

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
	"github.com/bwireman/archivist/internal/version"

	_ "modernc.org/sqlite"
)

func TestRecordRoundTrip(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	rec := &record.Record{
		ID:         "rec_test",
		Slug:       "test-rule",
		Type:       record.TypeRule,
		Scope:      record.ScopeRepo,
		Title:      "Test rule",
		Status:     record.StatusAccepted,
		Severity:   record.SeverityMust,
		Body:       "Do the thing.",
		SourcePath: "docs/decisions/test-rule.md",
		AppliesTo:  []string{"internal/**"},
	}
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	got, ok, err := st.GetRecordByID("rec_test")
	if err != nil || !ok {
		t.Fatalf("get record: ok=%v err=%v", ok, err)
	}
	if got.Title != rec.Title {
		t.Fatalf("title: %s", got.Title)
	}
	depth, err := st.QueueDepth()
	if err != nil || depth != 1 {
		t.Fatalf("queue depth: %d err=%v", depth, err)
	}
	if err := st.SetRecordVector(rec.ID, "test-model", []float32{1, 0, 0}); err != nil {
		t.Fatal(err)
	}
	depth, _ = st.QueueDepth()
	if depth != 0 {
		t.Fatalf("expected empty queue, got %d", depth)
	}
}

func TestSchemaNewerThanCLI(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.SetMeta(store.MetaSchemaVersion, "999"); err != nil {
		t.Fatal(err)
	}
	_, err = store.Open(filepath.Join(dir, "test.db"))
	if err == nil {
		t.Fatal("expected schema error")
	}
	var schemaErr *store.SchemaError
	if !errors.As(err, &schemaErr) {
		t.Fatalf("expected SchemaError, got %v", err)
	}
	if schemaErr.Have != 999 || schemaErr.Want != version.Schema {
		t.Fatalf("schema error: %+v", schemaErr)
	}
}

func TestFileMapRoundTrip(t *testing.T) {
	dir := t.TempDir()
	st, err := store.Open(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	now := time.Now().UTC()
	if err := st.ReplaceFileMap(store.FileRecord{
		Path: "foo.go", ContentHash: "abc", PackageName: "foo", IndexedAt: now,
	}, []store.Symbol{{FilePath: "foo.go", Name: "Bar", Kind: "func", Line: 1, Exported: true}}, nil); err != nil {
		t.Fatal(err)
	}
	syms, err := st.SymbolsForFile("foo.go")
	if err != nil || len(syms) != 1 {
		t.Fatalf("symbols: %v err=%v", syms, err)
	}
	if err := st.DeleteFile("foo.go"); err != nil {
		t.Fatal(err)
	}
	n, _ := st.FileCount()
	if n != 0 {
		t.Fatalf("expected 0 files, got %d", n)
	}
}

func TestDeleteRecordByPathDropsQueue(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rec := &record.Record{
		ID:         "rec_drop",
		Slug:       "drop-me",
		Type:       record.TypeDecision,
		Scope:      record.ScopeRepo,
		Title:      "Drop me",
		Status:     record.StatusAccepted,
		Body:       "gone",
		SourcePath: "docs/decisions/drop-me.md",
	}
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	if err := st.DeleteRecordByPath(rec.SourcePath); err != nil {
		t.Fatal(err)
	}
	depth, _ := st.QueueDepth()
	if depth != 0 {
		t.Fatalf("queue depth %d after delete", depth)
	}
	if _, ok, err := st.GetRecordByID(rec.ID); err != nil || ok {
		t.Fatalf("record still present ok=%v err=%v", ok, err)
	}
}

func TestDequeueEmbedZeroLimitReturnsAll(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	for i := 0; i < 20; i++ {
		rec := &record.Record{
			ID:         fmt.Sprintf("rec_%02d", i),
			Slug:       fmt.Sprintf("rec-%02d", i),
			Type:       record.TypeDecision,
			Scope:      record.ScopeRepo,
			Title:      fmt.Sprintf("Title %d", i),
			Status:     record.StatusAccepted,
			Body:       "body",
			SourcePath: fmt.Sprintf("docs/decisions/rec-%02d.md", i),
		}
		if err := st.UpsertRecord(rec); err != nil {
			t.Fatal(err)
		}
	}
	limited, err := st.DequeueEmbed(16)
	if err != nil {
		t.Fatal(err)
	}
	if len(limited) != 16 {
		t.Fatalf("limit 16 returned %d", len(limited))
	}
	all, err := st.DequeueEmbed(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 20 {
		t.Fatalf("limit 0 returned %d, want 20", len(all))
	}
}

func TestOpenPurgesOrphanQueueAndVectors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	rec := &record.Record{
		ID:         "rec_ghost",
		Slug:       "ghost",
		Type:       record.TypeDecision,
		Scope:      record.ScopeRepo,
		Title:      "Ghost",
		Status:     record.StatusAccepted,
		Body:       "body",
		SourcePath: "docs/decisions/ghost.md",
	}
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	if err := st.SetRecordVector(rec.ID, "test", []float32{1, 0, 0, 0}); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(0)")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO embed_queue(record_id, text_hash, enqueued_at, attempts) VALUES (?, 'x', '2026-01-01T00:00:00Z', 0)`, rec.ID); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM records WHERE id = ?`, rec.ID); err != nil {
		_ = db.Close()
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	st, err = store.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	depth, _ := st.QueueDepth()
	if depth != 0 {
		t.Fatalf("queue depth %d after open, want 0", depth)
	}
	if _, _, ok, err := st.GetRecordVector(rec.ID); err != nil || ok {
		t.Fatalf("orphan vector still present ok=%v err=%v", ok, err)
	}
}

func TestUpsertRecordSamePathKeepsID(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	first := &record.Record{
		ID: "rec_orig", Slug: "same-path", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Original", Status: record.StatusAccepted, Body: "first",
		SourcePath: "docs/decisions/same-path.md",
	}
	if err := st.UpsertRecord(first); err != nil {
		t.Fatal(err)
	}
	second := &record.Record{
		ID: "rec_new", Slug: "same-path", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Updated", Status: record.StatusAccepted, Body: "second",
		SourcePath: "docs/decisions/same-path.md",
	}
	if err := st.UpsertRecord(second); err != nil {
		t.Fatal(err)
	}
	if second.ID != "rec_orig" {
		t.Fatalf("adopted id %s", second.ID)
	}
	got, ok, err := st.GetRecordByID("rec_orig")
	if err != nil || !ok {
		t.Fatalf("original id missing ok=%v err=%v", ok, err)
	}
	if got.Body != "second" {
		t.Fatalf("body %q", got.Body)
	}
	if _, ok, err := st.GetRecordByID("rec_new"); err != nil || ok {
		t.Fatalf("new id should not exist ok=%v err=%v", ok, err)
	}
	hits, err := st.SearchFTS("Updated", 5, store.RecordFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].RecordID != "rec_orig" {
		t.Fatalf("fts hits %+v", hits)
	}
	depth, _ := st.QueueDepth()
	if depth != 1 {
		t.Fatalf("queue depth %d", depth)
	}
	items, err := st.DequeueEmbed(0)
	if err != nil || len(items) != 1 || items[0].RecordID != "rec_orig" {
		t.Fatalf("queue %+v err=%v", items, err)
	}
}

func TestUpsertUnchangedDoesNotRequeue(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rec := &record.Record{
		ID: "rec_stable", Slug: "stable", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Stable", Status: record.StatusAccepted, Body: "same",
		SourcePath: "docs/decisions/stable.md",
	}
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	if err := st.SetRecordVector(rec.ID, "test", []float32{1, 0}); err != nil {
		t.Fatal(err)
	}
	again := *rec
	again.ContentHash = rec.ContentHash
	if err := st.UpsertRecord(&again); err != nil {
		t.Fatal(err)
	}
	depth, _ := st.QueueDepth()
	if depth != 0 {
		t.Fatalf("unchanged upsert requeued, depth %d", depth)
	}
}

func TestGetRecordsByIDs(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	for i, id := range []string{"rec_a", "rec_b"} {
		rec := &record.Record{
			ID: id, Slug: id, Type: record.TypeDecision, Scope: record.ScopeRepo,
			Title: id, Status: record.StatusAccepted, Body: "body",
			SourcePath: fmt.Sprintf("docs/decisions/%s.md", id),
		}
		if err := st.UpsertRecord(rec); err != nil {
			t.Fatal(err)
		}
		_ = i
	}
	got, err := st.GetRecordsByIDs([]string{"rec_a", "rec_missing", "rec_b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got["rec_a"] == nil || got["rec_b"] == nil {
		t.Fatalf("got %+v", got)
	}
}
