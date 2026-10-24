package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
)

func TestLinkFixtureRepos(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	primary := t.TempDir()
	peer := t.TempDir()
	fixture := filepath.Join("..", "..", "testdata", "cli-fixture", "steps", "01-pantry")
	copyFixture(t, fixture, primary)
	copyFixture(t, fixture, peer)
	ensureQueryStores(t, primary)

	seedCheckoutRecord(t, primary, &record.Record{
		ID: "rec_pantry", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "Pantry shelf", Slug: "pantry-shelf", Status: record.StatusAccepted,
		Body: "local pantry checkout", SourcePath: "docs/pantry.md",
	})
	seedCheckoutRecord(t, peer, &record.Record{
		ID: "rec_seal", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "Jar seal", Slug: "jar-seal", Status: record.StatusAccepted,
		Body: "brimstone wax stamp", SourcePath: "docs/seal.md",
		AppliesTo: []string{"pantry/jar/jar.go"},
	})
	if err := config.Save(primary, config.Default()); err != nil {
		t.Fatal(err)
	}

	runLink(t, primary, "link", "--kind", "record", "--from", "rec_pantry", "--to", peer+"#rec_seal")
	runLink(t, primary, "link", "--kind", "pin", "--to", peer+"#rec_seal")
	runLink(t, primary, "link", "--kind", "repo", "--from", "rec_pantry", "--to", peer)

	var env struct {
		Results []struct {
			ID string `json:"id"`
		} `json:"results"`
		Links []struct {
			Kind  string `json:"kind"`
			Files []struct {
				Path string `json:"path"`
				Text string `json:"text"`
			} `json:"files"`
			Hits []struct {
				ID string `json:"id"`
			} `json:"hits"`
		} `json:"links"`
	}
	if err := json.Unmarshal([]byte(runSearch(t, primary, "--json", "pantry")), &env); err != nil {
		t.Fatal(err)
	}
	if len(env.Results) == 0 || env.Results[0].ID != "rec_pantry" {
		t.Fatalf("results %#v", env.Results)
	}
	var recordFiles, pinFiles int
	var sawRepo bool
	for _, link := range env.Links {
		switch link.Kind {
		case "record", "pin":
			if len(link.Files) != 1 || link.Files[0].Path != "pantry/jar/jar.go" || !strings.Contains(link.Files[0].Text, "func Seal") {
				t.Fatalf("%s files %#v", link.Kind, link.Files)
			}
			if link.Kind == "record" {
				recordFiles++
			} else {
				pinFiles++
			}
		case "repo":
			sawRepo = true
			if len(link.Hits) != 1 || link.Hits[0].ID != "rec_seal" {
				t.Fatalf("repo hits %#v", link.Hits)
			}
			if len(link.Files) != 1 || link.Files[0].Path != "pantry/jar/jar.go" || !strings.Contains(link.Files[0].Text, "func Seal") {
				t.Fatalf("repo files %#v", link.Files)
			}
		}
	}
	if recordFiles != 1 || pinFiles != 1 || !sawRepo {
		t.Fatalf("links %#v", env.Links)
	}
}

func copyFixture(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.CopyFS(dst, os.DirFS(src)); err != nil {
		t.Fatal(err)
	}
}
