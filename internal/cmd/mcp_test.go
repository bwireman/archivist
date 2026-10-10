package cmd

import (
	"os"
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
	path := config.StorePath(extra)
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })
	cfg := config.Default()
	cfg.Archives = []string{extra}
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	repo, homeDB, extras, err := openReadArchives(root, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer repo.Close()
	defer homeDB.Close()
	defer closeExtras(extras)
	if len(extras) != 1 {
		t.Fatalf("extras %d", len(extras))
	}
	n, err := extras[0].DB.RecordCount()
	if err != nil || n != 1 {
		t.Fatalf("extra count %d err=%v", n, err)
	}
}
