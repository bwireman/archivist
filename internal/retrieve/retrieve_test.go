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

func TestSearchDropsWeakVectorOnlyAndRetired(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "repo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	hybrid := &record.Record{
		ID: "rec_hybrid", Slug: "alpha", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "Alpha feature", Status: record.StatusAccepted, Body: "alpha term",
		SourcePath: "docs/global-decisions/alpha.md",
	}
	weak := &record.Record{
		ID: "rec_weak", Slug: "weak", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "Unrelated weak", Status: record.StatusAccepted, Body: "nothing shared",
		SourcePath: "docs/global-decisions/weak.md",
	}
	strong := &record.Record{
		ID: "rec_strong", Slug: "strong", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "Unrelated strong", Status: record.StatusAccepted, Body: "nothing shared either",
		SourcePath: "docs/global-decisions/strong.md",
	}
	retired := &record.Record{
		ID: "rec_old", Slug: "old-alpha", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Old alpha", Status: record.StatusSuperseded, Body: "alpha term retired",
		SourcePath: "docs/global-decisions/old-alpha.md",
	}
	for _, rec := range []*record.Record{hybrid, weak, strong, retired} {
		if err := st.UpsertRecord(rec); err != nil {
			t.Fatal(err)
		}
	}
	// {1,3} against query {1,0} is about 0.32, under the floor. {1,0} is 1.
	vecs := map[string][]float32{
		hybrid.ID:  {1, 3},
		weak.ID:    {1, 3},
		strong.ID:  {1, 0},
		retired.ID: {1, 0},
	}
	for id, vec := range vecs {
		if err := st.SetRecordVector(id, "test", vec); err != nil {
			t.Fatal(err)
		}
	}

	engine := &retrieve.Engine{Repo: st}
	results, err := engine.Search(context.Background(), fixedEmbedder{vec: []float32{1, 0}}, retrieve.Options{
		Query: "alpha",
		TopK:  10,
	})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, r := range results {
		got[r.Record.ID] = r.Source
	}
	if got[hybrid.ID] != "hybrid" || got[strong.ID] != "vector" || len(got) != 2 {
		t.Fatalf("results: %+v", results)
	}

	onlyOld, err := engine.Search(context.Background(), nil, retrieve.Options{
		Query:  "alpha",
		Status: record.StatusSuperseded,
		TopK:   10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(onlyOld) != 1 || onlyOld[0].Record.ID != retired.ID {
		t.Fatalf("status filter: %+v", onlyOld)
	}

	if _, err := engine.Search(context.Background(), nil, retrieve.Options{Query: "alpha", Status: "nope"}); err == nil {
		t.Fatal("expected invalid status")
	}
}
