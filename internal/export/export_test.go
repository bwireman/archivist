package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
)

func TestRunNestsRecordsScopeThenType(t *testing.T) {
	st := openStore(t)
	mustUpsert(t, st, &record.Record{
		ID: "rec_rule", Slug: "no-fts", Type: record.TypeRule, Scope: record.ScopeGlobal,
		Title: "Never pass unsanitized text", Status: record.StatusAccepted,
		Severity: record.SeverityMust, Body: "Sanitize MATCH input.",
		SourcePath: "docs/global-decisions/no-fts.md", AppliesTo: []string{"internal/store/**"},
	})
	mustUpsert(t, st, &record.Record{
		ID: "rec_feat", Slug: "hybrid-search", Type: record.TypeFeature, Scope: record.ScopeGlobal,
		Title: "Hybrid search", Status: record.StatusAccepted, Body: "FTS plus vectors.",
		SourcePath: "docs/global-decisions/hybrid-search.md", Tags: []string{"search"},
	})
	mustUpsert(t, st, &record.Record{
		ID: "rec_dec", Slug: "plain-cli", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Plain CLI", Status: record.StatusAccepted, Body: "No TUI.",
		SourcePath: "docs/decisions/plain-cli.md",
	})

	out := t.TempDir()
	stale := filepath.Join(out, "records", "rule", "global", "no-fts.md")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "guides.md"), []byte("# leftover\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Run(st, nil, Options{OutDir: out}); err != nil {
		t.Fatal(err)
	}

	rulePath := filepath.Join(out, "records", "global", "rule", "no-fts.md")
	if _, err := os.Stat(rulePath); err != nil {
		t.Fatalf("typed copy missing: %v", err)
	}
	featPath := filepath.Join(out, "records", "global", "feature", "hybrid-search.md")
	if _, err := os.Stat(featPath); err != nil {
		t.Fatalf("feature copy missing: %v", err)
	}
	decPath := filepath.Join(out, "records", "repo", "decision", "plain-cli.md")
	if _, err := os.Stat(decPath); err != nil {
		t.Fatalf("decision copy missing: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatal("expected old records/<type>/<scope>/ layout to be removed")
	}

	index, err := os.ReadFile(filepath.Join(out, "INDEX.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(index)
	if !strings.Contains(got, "[Rules](rules.md)") || !strings.Contains(got, "[Features](features.md)") {
		t.Fatalf("index missing digest links: %s", got)
	}
	if !strings.Contains(got, "records/global/rule/no-fts.md") {
		t.Fatalf("index missing typed path: %s", got)
	}

	rules, err := os.ReadFile(filepath.Join(out, "rules.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(rules), "Sanitize MATCH input.") {
		t.Fatalf("rules digest missing body: %s", rules)
	}
	features, err := os.ReadFile(filepath.Join(out, "features.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(features), "FTS plus vectors.") {
		t.Fatalf("features digest missing body: %s", features)
	}
	if _, err := os.Stat(filepath.Join(out, "guides.md")); !os.IsNotExist(err) {
		t.Fatal("expected empty-type digest leftover to be removed")
	}
	if _, err := os.Stat(filepath.Join(out, "map.md")); err != nil {
		t.Fatalf("code map missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "maps.md")); !os.IsNotExist(err) {
		t.Fatal("maps.md should not exist without map-type records")
	}
}

func TestDigestFileKeepsCodeMapNameFree(t *testing.T) {
	if DigestFile(record.TypeMap) != "maps.md" {
		t.Fatal(DigestFile(record.TypeMap))
	}
	if DigestFile(record.TypeRule) != "rules.md" {
		t.Fatal(DigestFile(record.TypeRule))
	}
}

func TestRunWritesMapsDigestBesideCodeMap(t *testing.T) {
	st := openStore(t)
	mustUpsert(t, st, &record.Record{
		ID: "rec_map", Slug: "http-surface", Type: record.TypeMap, Scope: record.ScopeGlobal,
		Title: "HTTP surface", Status: record.StatusAccepted, Body: "Handlers live in internal/http.",
		SourcePath: "docs/global-decisions/http-surface.md",
	})
	out := t.TempDir()
	if err := Run(st, nil, Options{OutDir: out}); err != nil {
		t.Fatal(err)
	}
	maps, err := os.ReadFile(filepath.Join(out, "maps.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(maps), "Handlers live in internal/http.") {
		t.Fatalf("maps digest: %s", maps)
	}
	codeMap, err := os.ReadFile(filepath.Join(out, "map.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(codeMap), "# Code Map\n") {
		t.Fatalf("code map clobbered: %s", codeMap)
	}
}

func TestRunAlwaysWritesRulesDigest(t *testing.T) {
	st := openStore(t)
	out := t.TempDir()
	if err := Run(st, nil, Options{OutDir: out}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "rules.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(data), "# Rules\n") {
		t.Fatalf("rules.md: %s", data)
	}
}

func openStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func mustUpsert(t *testing.T, st *store.Store, rec *record.Record) {
	t.Helper()
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
}
