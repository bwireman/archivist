package publish

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
)

func TestPublishUnknownDestination(t *testing.T) {
	err := Publish(t.TempDir(), config.Default(), nil, nil, "missing")
	if err == nil {
		t.Fatal("expected unknown destination error")
	}
}

func TestPublishRunsDestinationCommand(t *testing.T) {
	repo := t.TempDir()
	st, err := store.Open(filepath.Join(t.TempDir(), "repo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.UpsertRecord(&record.Record{
		ID: "rec_pub", Slug: "pub", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Publish test", Status: record.StatusAccepted, Body: "bundle me",
		SourcePath: "docs/decisions/pub.md",
	}); err != nil {
		t.Fatal(err)
	}

	copied := filepath.Join(t.TempDir(), "out")
	cfg := config.Default()
	cfg.Publish.Destinations = map[string]config.PublishDestination{
		"copy": {Command: []string{"cp", "-r", "{{bundle}}", copied}},
	}
	if err := Publish(repo, cfg, st, nil, "copy"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(copied, "INDEX.md")); err != nil {
		t.Fatalf("bundle INDEX.md missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(copied, "rules.md")); err != nil {
		t.Fatalf("bundle rules.md missing: %v", err)
	}
}

func TestPublishEmptyCommand(t *testing.T) {
	cfg := config.Default()
	cfg.Publish.Destinations = map[string]config.PublishDestination{
		"noop": {Command: nil},
	}
	if err := Publish(t.TempDir(), cfg, nil, nil, "noop"); err == nil {
		t.Fatal("expected empty command error")
	}
}
