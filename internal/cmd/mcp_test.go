package cmd

import (
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
)

func TestMCPOpenReadsExtraWithoutServing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	extra := t.TempDir()
	seedCheckoutRecord(t, extra, &record.Record{
		ID: "rec_mcp", Slug: "mcp-extra", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "MCP extra", Body: "readable", Status: record.StatusAccepted,
		SourcePath: "docs/decisions/mcp-extra.md",
	})
	cfg := config.Default()
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	ensureQueryStores(t, root)
	repo, homeDB, extras, err := openReadArchives(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	defer homeDB.Close()
	defer closeExtras(extras)
	if len(extras) != 0 {
		t.Fatalf("extras %d", len(extras))
	}
	n, err := repo.RecordCount()
	if err != nil || n != 1 {
		t.Fatalf("record count %d err=%v", n, err)
	}
}
