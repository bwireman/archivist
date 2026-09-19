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

	ranked, err := st.RankEmbeddings([]float32{1, 0}, RecordFilter{Type: record.TypeRule}, 5)
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
