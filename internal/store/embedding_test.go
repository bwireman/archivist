package store

import (
	"path/filepath"
	"testing"

	"github.com/bwireman/archivist/internal/record"
)

func TestSearchFTSFiltersType(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	rule := &record.Record{
		ID: "rec_rule", Slug: "rule-one", Type: record.TypeRule, Scope: record.ScopeRepo,
		Title: "Must sanitize", Status: record.StatusAccepted, Body: "sanitize fts",
		SourcePath: "docs/global-decisions/rule-one.md",
	}
	dec := &record.Record{
		ID: "rec_dec", Slug: "dec-one", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Use sqlite", Status: record.StatusAccepted, Body: "sqlite fts",
		SourcePath: "docs/decisions/dec-one.md",
	}
	if err := st.UpsertRecord(rule); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertRecord(dec); err != nil {
		t.Fatal(err)
	}
	if err := st.SetRecordVector(rule.ID, "test", []float32{1, 0}); err != nil {
		t.Fatal(err)
	}
	if err := st.SetRecordVector(dec.ID, "test", []float32{0, 1}); err != nil {
		t.Fatal(err)
	}

	hits, err := st.SearchFTS("sanitize", 10, RecordFilter{Type: record.TypeRule})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].RecordID != rule.ID {
		t.Fatalf("rule fts hits: %+v", hits)
	}
}

func TestRankEmbeddingsDoesNotNeedDecodedRows(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	rule := &record.Record{
		ID: "rec_rule", Slug: "rule-one", Type: record.TypeRule, Scope: record.ScopeRepo,
		Title: "Must sanitize", Status: record.StatusAccepted, Body: "sanitize fts",
		SourcePath: "docs/global-decisions/rule-one.md",
	}
	dec := &record.Record{
		ID: "rec_dec", Slug: "dec-one", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Use sqlite", Status: record.StatusAccepted, Body: "sqlite fts",
		SourcePath: "docs/decisions/dec-one.md",
	}
	if err := st.UpsertRecord(rule); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertRecord(dec); err != nil {
		t.Fatal(err)
	}
	if err := st.SetRecordVector(rule.ID, "test", []float32{1, 0}); err != nil {
		t.Fatal(err)
	}
	if err := st.SetRecordVector(dec.ID, "test", []float32{0, 1}); err != nil {
		t.Fatal(err)
	}

	ranked, err := st.RankEmbeddings([]float32{1, 0}, "", RecordFilter{Type: record.TypeRule}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(ranked) != 1 || ranked[0].RecordID != rule.ID {
		t.Fatalf("ranked %+v", ranked)
	}

	// A vector against itself is cosine 1 whether or not the candidate side
	// was ever decoded out of its blob.
	query := []float32{0, 1}
	got := cosineSimilarityEncoded(query, vectorNorm(query), encodeEmbedding(query))
	if got != 1 {
		t.Fatalf("encoded cosine %v want 1", got)
	}
}

func TestRankEmbeddingsEmptyModelKeepsLengthMismatch(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rec := &record.Record{
		ID: "rec_short", Slug: "short", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "Short", Status: record.StatusAccepted, Body: "body",
		SourcePath: "docs/global-decisions/short.md",
	}
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	if err := st.SetRecordVector(rec.ID, "test", []float32{1, 0}); err != nil {
		t.Fatal(err)
	}
	ranked, err := st.RankEmbeddings([]float32{1, 0, 0, 0}, "", RecordFilter{}, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(ranked) != 1 || ranked[0].RecordID != rec.ID {
		t.Fatalf("ranked %+v", ranked)
	}
}

func TestUpsertDeletesVectorWhenEmbedTextChanges(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rec := &record.Record{
		ID: "rec_edit", Slug: "edit", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "Edit", Status: record.StatusAccepted, Body: "before",
		SourcePath: "docs/global-decisions/edit.md",
	}
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	if err := st.SetRecordVector(rec.ID, "test", []float32{1, 0}); err != nil {
		t.Fatal(err)
	}
	rec.Body = "after"
	rec.ContentHash = ""
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	if _, _, ok, err := st.GetRecordVector(rec.ID); err != nil || ok {
		t.Fatalf("vector ok %v err %v", ok, err)
	}
}

func TestUnchangedUpsertRequeuesGiveup(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rec := &record.Record{
		ID: "rec_gave", Slug: "gave", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "Gave", Status: record.StatusAccepted, Body: "same",
		SourcePath: "docs/global-decisions/gave.md",
	}
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	if err := st.DropQueueItem(rec.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.RecordGiveup(rec.ID, 5, "boom"); err != nil {
		t.Fatal(err)
	}
	rec.ContentHash = ""
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	depth, err := st.QueueDepth()
	if err != nil || depth != 1 {
		t.Fatalf("queue %d err %v", depth, err)
	}
	gave, err := st.EmbedGiveups()
	if err != nil || len(gave) != 0 {
		t.Fatalf("giveups %+v err %v", gave, err)
	}
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	depth, err = st.QueueDepth()
	if err != nil || depth != 1 {
		t.Fatalf("second upsert queue %d err %v", depth, err)
	}
}

func TestUpsertStatusKeepsVector(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rec := &record.Record{
		ID: "rec_status", Slug: "status", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "Status", Status: record.StatusAccepted, Body: "same body",
		SourcePath: "docs/global-decisions/status.md",
	}
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	if err := st.SetRecordVector(rec.ID, "test", []float32{1, 0}); err != nil {
		t.Fatal(err)
	}
	rec.Status = record.StatusProposed
	rec.ContentHash = ""
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	if _, _, ok, err := st.GetRecordVector(rec.ID); err != nil || !ok {
		t.Fatalf("vector ok %v err %v", ok, err)
	}
}

func TestReadOnlyGiveupMissingTable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.Exec(`DROP TABLE embed_giveup`); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	ro, err := OpenReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	got, err := ro.EmbedGiveups()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("giveups %+v", got)
	}
}

func TestOpenExistingEnsuresGiveup(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.db.Exec(`DROP TABLE embed_giveup`); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}
	st, err = OpenExisting(path)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rec := &record.Record{
		ID: "rec_up", Slug: "up", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "Up", Status: record.StatusAccepted, Body: "body",
		SourcePath: "docs/global-decisions/up.md",
	}
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
}
