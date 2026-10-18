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

func TestSearchReturnsEveryScope(t *testing.T) {
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

	// Three slugs, so overlayWins does not collapse them.
	records := []*record.Record{
		{
			ID: "rec_alpha", Slug: "alpha", Type: record.TypeDecision, Scope: record.ScopeRepo,
			Title: "Alpha", Status: record.StatusAccepted, Body: "zqxscope", SourcePath: "docs/decisions/alpha.md",
		},
		{
			ID: "rec_beta", Slug: "beta", Type: record.TypeDecision, Scope: record.ScopeGlobal,
			Title: "Beta", Status: record.StatusAccepted, Body: "zqxscope", SourcePath: "global/beta.md",
		},
		{
			ID: "rec_gamma", Slug: "gamma", Type: record.TypeDecision, Scope: record.ScopeDev,
			Title: "Gamma", Status: record.StatusAccepted, Body: "zqxscope", SourcePath: "user/gamma.md",
		},
	}
	if err := repo.UpsertRecord(records[0]); err != nil {
		t.Fatal(err)
	}
	if err := home.UpsertRecord(records[1]); err != nil {
		t.Fatal(err)
	}
	if err := home.UpsertRecord(records[2]); err != nil {
		t.Fatal(err)
	}

	results, err := (&retrieve.Engine{Repo: repo, Home: home}).Search(
		context.Background(), nil, retrieve.Options{Query: "zqxscope", TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, r := range results {
		got[r.Record.ID] = true
	}
	for _, id := range []string{"rec_alpha", "rec_beta", "rec_gamma"} {
		if !got[id] {
			t.Fatalf("missing %s in %+v", id, results)
		}
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

func TestSearchAnyTermOnlyBoostsVectorHits(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "repo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	boosted := &record.Record{
		ID: "rec_boosted", Slug: "review", Type: record.TypeRule, Scope: record.ScopeRepo,
		Title: "Review every commit", Status: record.StatusAccepted, Body: "review the diff",
		SourcePath: "docs/decisions/review.md",
	}
	plain := &record.Record{
		ID: "rec_plain", Slug: "plain", Type: record.TypeRule, Scope: record.ScopeRepo,
		Title: "Unrelated close vector", Status: record.StatusAccepted, Body: "nothing shared",
		SourcePath: "docs/decisions/plain.md",
	}
	noise := &record.Record{
		ID: "rec_noise", Slug: "noise", Type: record.TypeRule, Scope: record.ScopeRepo,
		Title: "Commit noise", Status: record.StatusAccepted, Body: "commit appears here",
		SourcePath: "docs/decisions/noise.md",
	}
	for _, rec := range []*record.Record{boosted, plain, noise} {
		if err := st.UpsertRecord(rec); err != nil {
			t.Fatal(err)
		}
	}
	// {1,3} against query {1,0} is about 0.32, under the floor.
	vecs := map[string][]float32{
		boosted.ID: {0.9, 0.1},
		plain.ID:   {1, 0},
		noise.ID:   {1, 3},
	}
	for id, vec := range vecs {
		if err := st.SetRecordVector(id, "test", vec); err != nil {
			t.Fatal(err)
		}
	}

	engine := &retrieve.Engine{Repo: st}
	query := "review commit subagent"
	results, err := engine.Search(context.Background(), fixedEmbedder{vec: []float32{1, 0}}, retrieve.Options{Query: query, TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].Record.ID != boosted.ID || results[0].Source != "hybrid" || results[1].Record.ID != plain.ID {
		t.Fatalf("any-term should boost the vector hit and drop the weak one: %+v", results)
	}

	// boosted has 2 of 3 terms (a keyword hit); noise has 1 (any-term only).
	keywordOnly, err := engine.Search(context.Background(), nil, retrieve.Options{Query: query, TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(keywordOnly) != 1 || keywordOnly[0].Record.ID != boosted.ID {
		t.Fatalf("keyword-only should keep the 2-of-3 hit and drop any-term noise: %+v", keywordOnly)
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

func TestSearchExtraKeepsSameSlug(t *testing.T) {
	repo, err := store.Open(filepath.Join(t.TempDir(), "repo.db"))
	if err != nil {
		t.Fatal(err)
	}
	extraPath := filepath.Join(t.TempDir(), "extra.db")
	extra, err := store.Open(extraPath)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	defer extra.Close()
	primary := &record.Record{
		ID: "rec_primary", Slug: "shared-slug", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Sharedslug primary", Status: record.StatusAccepted, Body: "sharedslug token",
		SourcePath: "docs/decisions/shared-slug.md",
	}
	other := &record.Record{
		ID: "rec_extra", Slug: "shared-slug", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Sharedslug extra", Status: record.StatusAccepted, Body: "sharedslug token",
		SourcePath: "docs/decisions/shared-slug.md",
	}
	if err := repo.UpsertRecord(primary); err != nil {
		t.Fatal(err)
	}
	if err := extra.UpsertRecord(other); err != nil {
		t.Fatal(err)
	}
	beforeRepo, _ := repo.RecordCount()
	beforeExtra, _ := extra.RecordCount()
	root := filepath.Dir(filepath.Dir(extraPath))
	engine := &retrieve.Engine{Repo: repo, Extras: []retrieve.Extra{{Root: root, DB: extra}}}
	results, err := engine.Search(context.Background(), nil, retrieve.Options{Query: "sharedslug", TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, r := range results {
		got[r.Record.ID] = r.Archive
	}
	if _, ok := got["rec_primary"]; !ok {
		t.Fatalf("missing primary: %+v", results)
	}
	if got["rec_primary"] != "" {
		t.Fatalf("primary archive %q", got["rec_primary"])
	}
	if got["rec_extra"] != root {
		t.Fatalf("extra archive %q results %+v", got["rec_extra"], results)
	}
	cards := retrieve.ProjectSearch(results)
	for _, card := range cards {
		if card.ID == "rec_primary" && card.Archive != "" {
			t.Fatalf("primary card archive %q", card.Archive)
		}
		if card.ID == "rec_extra" && card.Archive != root {
			t.Fatalf("extra card %+v", card)
		}
	}
	empty := retrieve.ProjectSearch([]retrieve.Result{{
		Record: &record.Record{
			ID: "rec_1", Slug: "one", Type: record.TypeFeature, Scope: record.ScopeGlobal,
			Title: "Card", Status: record.StatusAccepted,
		},
		Source: retrieve.SourceHybrid,
	}})
	if empty[0].Archive != "" {
		t.Fatal("empty archive should stay unset")
	}
	afterRepo, _ := repo.RecordCount()
	afterExtra, _ := extra.RecordCount()
	if beforeRepo != afterRepo || beforeExtra != afterExtra {
		t.Fatalf("counts changed %d/%d -> %d/%d", beforeRepo, beforeExtra, afterRepo, afterExtra)
	}
}

func TestExtraArchiveSortsAfterWeakerCheckoutHit(t *testing.T) {
	repo, err := store.Open(filepath.Join(t.TempDir(), "repo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	extra, err := store.Open(filepath.Join(t.TempDir(), "extra.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer extra.Close()
	first := &record.Record{
		ID: "rec_first", Slug: "first", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "First zqxband", Status: record.StatusAccepted, Body: "alpha",
		SourcePath: "docs/decisions/first.md",
	}
	second := &record.Record{
		ID: "rec_second", Slug: "second", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Second", Status: record.StatusAccepted, Body: "zqxband",
		SourcePath: "docs/decisions/second.md",
	}
	outside := &record.Record{
		ID: "rec_outside", Slug: "outside", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Outside zqxband", Status: record.StatusAccepted, Body: "only here",
		SourcePath: "docs/decisions/outside.md",
	}
	if err := repo.UpsertRecord(first); err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertRecord(second); err != nil {
		t.Fatal(err)
	}
	if err := extra.UpsertRecord(outside); err != nil {
		t.Fatal(err)
	}
	results, err := (&retrieve.Engine{Repo: repo, Extras: []retrieve.Extra{{Root: "/extra", DB: extra}}}).Search(
		context.Background(), nil, retrieve.Options{Query: "zqxband", TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 || results[2].Record.ID != outside.ID || results[2].Band != 1 {
		t.Fatalf("extra should sort last: %+v", results)
	}
	for _, r := range results[:2] {
		if r.Band != 0 {
			t.Fatalf("checkout band: %+v", results)
		}
	}
}

func TestEqualRRFSortsByCosine(t *testing.T) {
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
	high := &record.Record{
		ID: "rec_z", Slug: "high", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "High zqxequal", Status: record.StatusAccepted, Body: "zqxequal",
		SourcePath: "docs/decisions/high.md",
	}
	low := &record.Record{
		ID: "rec_a", Slug: "low", Type: record.TypeDecision, Scope: record.ScopeGlobal,
		Title: "Low zqxequal", Status: record.StatusAccepted, Body: "zqxequal",
		SourcePath: "global/low.md",
	}
	if err := repo.UpsertRecord(high); err != nil {
		t.Fatal(err)
	}
	if err := home.UpsertRecord(low); err != nil {
		t.Fatal(err)
	}
	if err := repo.SetRecordVector(high.ID, "test", []float32{1, 0}); err != nil {
		t.Fatal(err)
	}
	if err := home.SetRecordVector(low.ID, "test", []float32{0.6, 0.8}); err != nil {
		t.Fatal(err)
	}
	results, err := (&retrieve.Engine{Repo: repo, Home: home}).Search(
		context.Background(), fixedEmbedder{vec: []float32{1, 0}}, retrieve.Options{Query: "zqxequal", TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 || results[0].Record.ID != high.ID || results[0].Cosine <= results[1].Cosine {
		t.Fatalf("cosine tie-break: %+v", results)
	}
}

type recordingEmbedder struct {
	got string
	vec []float32
}

func (r *recordingEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	r.got = text
	if r.vec == nil {
		return []float32{1, 0}, nil
	}
	return r.vec, nil
}

func TestSearchEmbedsQwenQueryPrefix(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "repo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rec := &record.Record{
		ID: "rec_q", Slug: "q", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "Hybrid", Status: record.StatusAccepted, Body: "hybrid search",
		SourcePath: "docs/global-decisions/q.md",
	}
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	emb := &recordingEmbedder{}
	if _, err := (&retrieve.Engine{Repo: st, QueryModel: "qwen3-embedding:8b"}).Search(
		context.Background(), emb, retrieve.Options{Query: "hybrid", TopK: 5}); err != nil {
		t.Fatal(err)
	}
	if emb.got != retrieve.QueryText("qwen3-embedding:8b", "hybrid") {
		t.Fatalf("prefixed query %q", emb.got)
	}
	emb.got = ""
	if _, err := (&retrieve.Engine{Repo: st}).Search(
		context.Background(), emb, retrieve.Options{Query: "hybrid", TopK: 5}); err != nil {
		t.Fatal(err)
	}
	if emb.got != "hybrid" {
		t.Fatalf("raw query %q", emb.got)
	}
}

func TestSearchOmitsVectorOnlyWrongModel(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "repo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	queryVec := []float32{1, 0}
	kept := &record.Record{
		ID: "rec_kept", Slug: "kept", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "Kept", Status: record.StatusAccepted, Body: "unrelated kept",
		SourcePath: "docs/global-decisions/kept.md",
	}
	other := &record.Record{
		ID: "rec_other", Slug: "other", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "Other", Status: record.StatusAccepted, Body: "unrelated other",
		SourcePath: "docs/global-decisions/other.md",
	}
	if err := st.UpsertRecord(kept); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertRecord(other); err != nil {
		t.Fatal(err)
	}
	if err := st.SetRecordVector(kept.ID, "qwen3-embedding:8b", queryVec); err != nil {
		t.Fatal(err)
	}
	if err := st.SetRecordVector(other.ID, "other", queryVec); err != nil {
		t.Fatal(err)
	}
	before, err := st.QueueDepth()
	if err != nil {
		t.Fatal(err)
	}
	results, err := (&retrieve.Engine{Repo: st, QueryModel: "qwen3-embedding:8b"}).Search(
		context.Background(), fixedEmbedder{vec: queryVec}, retrieve.Options{Query: "zqxnomatch", TopK: 10})
	if err != nil {
		t.Fatal(err)
	}
	after, err := st.QueueDepth()
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("search changed queue %d -> %d", before, after)
	}
	if len(results) != 1 || results[0].Record.ID != kept.ID || results[0].Source != retrieve.SourceVector {
		t.Fatalf("results %+v", results)
	}
}

func TestSearchWrongDimStaysKeywordSource(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "repo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	rec := &record.Record{
		ID: "rec_dim", Slug: "dim", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "Alpha term", Status: record.StatusAccepted, Body: "alpha term",
		SourcePath: "docs/global-decisions/dim.md",
	}
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	if err := st.SetRecordVector(rec.ID, "qwen3-embedding:8b", []float32{1, 0, 0, 0}); err != nil {
		t.Fatal(err)
	}
	results, err := (&retrieve.Engine{Repo: st, QueryModel: "qwen3-embedding:8b"}).Search(
		context.Background(), fixedEmbedder{vec: []float32{1, 0}}, retrieve.Options{Query: "alpha", TopK: 5})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Source != retrieve.SourceFTS {
		t.Fatalf("results %+v", results)
	}
}
