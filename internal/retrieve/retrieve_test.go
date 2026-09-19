package retrieve_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/retrieve"
	"github.com/bwireman/archivist/internal/store"
)

type fixedEmbedder struct {
	vec []float32
}

func (f fixedEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	return f.vec, nil
}

func TestSearchFiltersByType(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "repo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	rule := &record.Record{
		ID: "rec_rule", Slug: "sanitize", Type: record.TypeRule, Scope: record.ScopeRepo,
		Title: "Sanitize MATCH", Status: record.StatusAccepted, Body: "fts sanitize",
		SourcePath: "docs/global-decisions/sanitize.md",
	}
	dec := &record.Record{
		ID: "rec_dec", Slug: "sqlite", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Use sqlite", Status: record.StatusAccepted, Body: "fts sanitize backup",
		SourcePath: "docs/decisions/sqlite.md",
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
	if err := st.SetRecordVector(dec.ID, "test", []float32{0.9, 0.1}); err != nil {
		t.Fatal(err)
	}

	engine := &retrieve.Engine{Repo: st}
	results, err := engine.Search(context.Background(), fixedEmbedder{vec: []float32{1, 0}}, retrieve.Options{
		Query: "sanitize",
		Type:  record.TypeRule,
		TopK:  5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Record.ID != rule.ID {
		t.Fatalf("filtered results: %+v", results)
	}
}

func TestSearchOverlayPrefersRepoScope(t *testing.T) {
	repo, err := store.Open(filepath.Join(t.TempDir(), "repo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	home, err := store.Open(filepath.Join(t.TempDir(), "home.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer home.Close()

	repoRec := &record.Record{
		ID: "rec_repo", Slug: "billing", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Repo billing", Status: record.StatusAccepted, Body: "billing policy",
		SourcePath: "docs/decisions/billing.md",
	}
	globalRec := &record.Record{
		ID: "rec_global", Slug: "billing", Type: record.TypeDecision, Scope: record.ScopeGlobal,
		Title: "Global billing", Status: record.StatusAccepted, Body: "billing policy",
		SourcePath: "global/billing.md",
	}
	if err := repo.UpsertRecord(repoRec); err != nil {
		t.Fatal(err)
	}
	if err := home.UpsertRecord(globalRec); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetRecordVector(repoRec.ID, "test", []float32{1, 0}); err != nil {
		t.Fatal(err)
	}
	if err := home.SetRecordVector(globalRec.ID, "test", []float32{1, 0}); err != nil {
		t.Fatal(err)
	}

	engine := &retrieve.Engine{Repo: repo, Home: home}
	results, err := engine.Search(context.Background(), fixedEmbedder{vec: []float32{1, 0}}, retrieve.Options{
		Query: "billing",
		TopK:  5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Record.ID != repoRec.ID {
		t.Fatalf("overlay results: %+v", results)
	}
}

// The scope overlay collapses same-topic records. Records of different types
// that happen to slugify the same are different topics and must both survive.
func TestSearchKeepsSameSlugAcrossTypes(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "repo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	feature := &record.Record{
		ID: "rec_feature", Slug: "archive-export", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "Archive export", Status: record.StatusAccepted, Body: "how export works",
		SourcePath: "docs/decisions/archive-export-feature.md",
	}
	rule := &record.Record{
		ID: "rec_rule", Slug: "archive-export", Type: record.TypeRule, Scope: record.ScopeRepo,
		Title: "Archive export", Status: record.StatusAccepted, Body: "never hand-edit export",
		SourcePath: "docs/decisions/archive-export-rule.md",
	}
	for _, rec := range []*record.Record{feature, rule} {
		if err := st.UpsertRecord(rec); err != nil {
			t.Fatal(err)
		}
	}

	engine := &retrieve.Engine{Repo: st}
	results, err := engine.Search(context.Background(), nil, retrieve.Options{Query: "export", TopK: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("both types should survive the overlay, got %+v", results)
	}
}
