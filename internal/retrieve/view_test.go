package retrieve

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/record"
)

func TestProjectSearchNilIsEmptyArray(t *testing.T) {
	data, err := json.Marshal(ProjectSearch(nil))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "[]" {
		t.Fatalf("nil search JSON %s", data)
	}
}

func TestProjectSearchCardKeys(t *testing.T) {
	cards := ProjectSearch([]Result{{
		Record: &record.Record{
			ID: "rec_1", Slug: "one", Type: record.TypeFeature, Scope: record.ScopeGlobal,
			Title: "Card", Status: record.StatusAccepted, Body: "BODYTOKEN",
			ContentHash: "abc", SourcePath: "docs/x.md",
		},
		Score:  0.5,
		Source: SourceHybrid,
	}})
	data, err := json.Marshal(cards)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, banned := range []string{"body", "Body", "ContentHash", "SourcePath", "created_at", "provenance"} {
		if strings.Contains(text, banned) {
			t.Fatalf("card contains %q: %s", banned, text)
		}
	}
	var hits []map[string]any
	if err := json.Unmarshal(data, &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 {
		t.Fatalf("hits: %s", text)
	}
	assertKeys(t, hits[0], "id", "slug", "type", "scope", "title", "status", "score", "source")
}

func TestProjectRecordKeys(t *testing.T) {
	card := ProjectRecord(&record.Record{
		ID: "rec_1", Slug: "one", Type: record.TypeRule, Scope: record.ScopeRepo,
		Title: "Rule", Status: record.StatusAccepted, Body: "keep it",
	})
	data, err := json.Marshal(card)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	assertKeys(t, got, "id", "slug", "type", "scope", "title", "status", "severity", "body", "tags", "applies_to", "superseded_by")
	if got["severity"] != "" || got["superseded_by"] != "" {
		t.Fatalf("empty fields: %#v", got)
	}
}

func assertKeys(t *testing.T, m map[string]any, want ...string) {
	t.Helper()
	var got []string
	for k := range m {
		got = append(got, k)
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Fatalf("keys %v, want %v", got, want)
	}
}
