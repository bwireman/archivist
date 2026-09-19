package store

import (
	"path/filepath"
	"testing"

	"github.com/bwireman/archivist/internal/record"
)

func TestFTS5Query(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"   ", ""},
		{"/", ""},
		{"***", ""},
		{"gitignore", `"gitignore"`},
		{"docs/decisions/foo.md", `"docs" "decisions" "foo" "md"`},
		{"records.global", `"records" "global"`},
		{"qwen3-embedding:0.6b", `"qwen3" "embedding" "0" "6b"`},
		{"foo AND bar", `"foo" "AND" "bar"`},
		{`unmatched " quote`, `"unmatched" "quote"`},
		{"NEAR(foo, bar)", `"NEAR" "foo" "bar"`},
		{"internal/**", `"internal"`},
		{"honor gitignore", `"honor" "gitignore"`},
	}
	for _, tt := range tests {
		if got := fts5Query(tt.in); got != tt.want {
			t.Errorf("fts5Query(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSearchFTSAcceptsPunctuation(t *testing.T) {
	st, rec := openFTSStore(t)
	queries := []string{
		"docs/decisions/billing.md",
		"records.global",
		"qwen3-embedding:0.6b",
		`unmatched " quote`,
		"foo AND bar",
		"NEAR(foo, bar)",
		"internal/**",
		"*",
		"...",
		"~/.archivist",
		"config.json:embed_model",
	}
	for _, q := range queries {
		if _, err := st.SearchFTS(q, 10, RecordFilter{}); err != nil {
			t.Errorf("SearchFTS(%q): %v", q, err)
		}
	}

	hits, err := st.SearchFTS("docs/decisions/billing.md", 10, RecordFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].RecordID != rec.ID {
		t.Fatalf("path query hits = %+v, want %s", hits, rec.ID)
	}

	hits, err = st.SearchFTS("records.global", 10, RecordFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].RecordID != rec.ID {
		t.Fatalf("dotted query hits = %+v, want %s", hits, rec.ID)
	}

	hits, err = st.SearchFTS("/", 10, RecordFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("punctuation-only query hits = %+v", hits)
	}
}

func TestSearchFTSDoesNotTreatSQLAsStatements(t *testing.T) {
	st, rec := openFTSStore(t)
	payloads := []string{
		"'; DROP TABLE records; --",
		"foo OR bar",
		`title:Billing`,
		"NOT gitignore",
	}
	for _, q := range payloads {
		if _, err := st.SearchFTS(q, 10, RecordFilter{}); err != nil {
			t.Errorf("SearchFTS(%q): %v", q, err)
		}
	}
	n, err := st.RecordCount()
	if err != nil || n != 1 {
		t.Fatalf("fts payload dropped records count=%d err=%v", n, err)
	}
	got, ok, err := st.GetRecordByID(rec.ID)
	if err != nil || !ok || got.ID != rec.ID {
		t.Fatalf("fixture lost after fts payloads")
	}
}

func openFTSStore(t *testing.T) (*Store, *record.Record) {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	rec := &record.Record{
		ID:         "rec_fts",
		Slug:       "fts-fixture",
		Type:       record.TypeDecision,
		Scope:      record.ScopeRepo,
		Title:      "Billing lives in docs/decisions/billing.md",
		Status:     record.StatusAccepted,
		Body:       "Also mentions records.global and qwen3-embedding:0.6b.",
		SourcePath: "docs/decisions/fts-fixture.md",
	}
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	return st, rec
}
