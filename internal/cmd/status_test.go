package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
)

func TestStatusListsExtraSeparatelyWhenReadOnly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	extra := t.TempDir()
	seedCheckoutRecord(t, extra, &record.Record{
		ID: "rec_status", Slug: "status-extra", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Status extra", Body: "counted apart", Status: record.StatusAccepted,
		SourcePath: "docs/decisions/status-extra.md",
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
	cmd := NewRoot()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--path", root, "status", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, stderr.String())
	}
	var got struct {
		RecordCount int `json:"record_count"`
		Extras      []struct {
			Root        string `json:"root"`
			RecordCount int    `json:"record_count"`
		} `json:"extras"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("%v\n%s", err, stdout.String())
	}
	if got.RecordCount != 0 {
		t.Fatalf("record_count %d includes the extra", got.RecordCount)
	}
	canon := mustCanonCmd(t, extra)
	if len(got.Extras) != 1 || got.Extras[0].Root != canon || got.Extras[0].RecordCount != 1 {
		t.Fatalf("extras %+v", got.Extras)
	}
}
