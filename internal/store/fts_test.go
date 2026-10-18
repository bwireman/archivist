package store

import (
	"path/filepath"
	"strings"
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
		if got := strings.Join(fts5Terms(tt.in), " "); got != tt.want {
			t.Errorf("fts5Terms(%q) = %q, want %q", tt.in, got, tt.want)
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

func TestSearchFTSFallsBackToAnyTerm(t *testing.T) {
	st, rec := openFTSStore(t)
	hits, err := st.SearchFTS("billing subagent", 10, RecordFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].RecordID != rec.ID || !hits[0].AnyTerm {
		t.Fatalf("fallback hits = %+v, want %s marked AnyTerm", hits, rec.ID)
	}

	hits, err = st.SearchFTS("subagent", 10, RecordFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("single missing term hits = %+v", hits)
	}
}

func TestMinShouldMatch(t *testing.T) {
	for n, want := range map[int]int{0: 0, 1: 1, 2: 2, 3: 2, 4: 3, 5: 4, 6: 4} {
		if got := minShouldMatch(n); got != want {
			t.Errorf("minShouldMatch(%d) = %d, want %d", n, got, want)
		}
	}
}

func TestSearchFTSTwoThirdsOfContentTerms(t *testing.T) {
	st, rec := openFTSStore(t)
	hits, err := st.SearchFTS("how billing works with records", 10, RecordFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].RecordID != rec.ID || hits[0].AnyTerm {
		t.Fatalf("2 of 3 content terms should be a hit: %+v", hits)
	}

	hits, err = st.SearchFTS("billing widget subagent", 10, RecordFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || !hits[0].AnyTerm {
		t.Fatalf("1 of 3 content terms should stay AnyTerm: %+v", hits)
	}
}

func TestSearchFTSStems(t *testing.T) {
	st, rec := openFTSStore(t)
	hits, err := st.SearchFTS("decision", 10, RecordFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].RecordID != rec.ID {
		t.Fatalf("decision should match decisions: %+v", hits)
	}
}

func TestOpenRebuildsUnstemmedFTS(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	rec := &record.Record{
		ID: "rec_old", Slug: "old", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Billing decisions", Status: record.StatusAccepted, Body: "body",
		SourcePath: "docs/decisions/old.md", Tags: []string{"payments"},
	}
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`DROP TABLE records_fts`,
		`CREATE VIRTUAL TABLE records_fts USING fts5(record_id UNINDEXED, title, body, tags)`,
	} {
		if _, err := st.db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.SetMeta(MetaFTSTokenizer, ""); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	st, err = Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	for _, q := range []string{"decision", "payments"} {
		hits, err := st.SearchFTS(q, 10, RecordFilter{})
		if err != nil {
			t.Fatal(err)
		}
		if len(hits) != 1 || hits[0].RecordID != rec.ID {
			t.Fatalf("rebuilt fts %q hits = %+v", q, hits)
		}
	}
}

func TestSearchFTSPrefersAllTerms(t *testing.T) {
	st, rec := openFTSStore(t)
	other := &record.Record{
		ID:         "rec_other",
		Slug:       "other",
		Type:       record.TypeDecision,
		Scope:      record.ScopeRepo,
		Title:      "Billing retries",
		Status:     record.StatusAccepted,
		Body:       "Unrelated.",
		SourcePath: "docs/decisions/other.md",
	}
	if err := st.UpsertRecord(other); err != nil {
		t.Fatal(err)
	}
	hits, err := st.SearchFTS("billing qwen3", 10, RecordFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].RecordID != rec.ID || hits[0].AnyTerm {
		t.Fatalf("all-terms hits = %+v, want only %s", hits, rec.ID)
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

func TestSearchFTSMatchesAppliesTo(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	rec := &record.Record{
		ID: "rec_path", Slug: "path", Type: record.TypeRule, Scope: record.ScopeRepo,
		Title: "Plain title", Status: record.StatusAccepted, Body: "plain body",
		AppliesTo:  []string{"pkg/zqxpath"},
		SourcePath: "docs/global-decisions/path.md",
	}
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	hits, err := st.SearchFTS("zqxpath", 10, RecordFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].RecordID != rec.ID {
		t.Fatalf("applies_to hits %+v", hits)
	}
}

func TestOpenRebuildsAppliesToColumn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	writeOldAppliesFTS(t, path)
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	hits, err := st.SearchFTS("zqxpath", 10, RecordFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("rebuilt hits %+v", hits)
	}
}

func TestOpenExistingRebuildsAppliesToColumn(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	writeOldAppliesFTS(t, path)
	st, err := OpenExisting(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	rec := &record.Record{
		ID: "rec_new", Slug: "new", Type: record.TypeRule, Scope: record.ScopeRepo,
		Title: "Also plain", Status: record.StatusAccepted, Body: "still plain",
		AppliesTo:  []string{"other/zqxpath"},
		SourcePath: "docs/global-decisions/new.md",
	}
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	hits, err := st.SearchFTS("zqxpath", 10, RecordFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 2 {
		t.Fatalf("hits %+v", hits)
	}
}

func TestOpenReadOnlyDoesNotRebuildFTS(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	writeOldAppliesFTS(t, path)
	st, err := OpenReadOnly(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	have, _, err := st.GetMeta(MetaFTSTokenizer)
	if err != nil {
		t.Fatal(err)
	}
	if have != ftsTokenizer {
		t.Fatalf("meta %q", have)
	}
	hits, err := st.SearchFTS("zqxpath", 10, RecordFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("read-only rebuilt: %+v", hits)
	}
}

func writeOldAppliesFTS(t *testing.T, path string) {
	t.Helper()
	st, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	rec := &record.Record{
		ID: "rec_oldpath", Slug: "oldpath", Type: record.TypeRule, Scope: record.ScopeRepo,
		Title: "Plain title", Status: record.StatusAccepted, Body: "plain body",
		AppliesTo:  []string{"pkg/zqxpath"},
		SourcePath: "docs/global-decisions/oldpath.md",
	}
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`DROP TABLE records_fts`,
		`CREATE VIRTUAL TABLE records_fts USING fts5(record_id UNINDEXED, title, body, tags)`,
		`INSERT INTO records_fts(record_id, title, body, tags) VALUES ('rec_oldpath', 'Plain title', 'plain body', '')`,
	} {
		if _, err := st.db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	if err := st.SetMeta(MetaFTSTokenizer, ftsTokenizer); err != nil {
		t.Fatal(err)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
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
