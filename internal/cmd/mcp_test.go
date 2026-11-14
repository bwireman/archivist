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
	db, err := openReadOnlyArchive(root)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	n, err := db.RecordCount()
	if err != nil || n != 1 {
		t.Fatalf("record count %d err=%v", n, err)
	}
}
