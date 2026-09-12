package store_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
	"github.com/bwireman/archivist/internal/version"
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
