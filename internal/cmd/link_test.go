package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
	"github.com/spf13/cobra"
)

func TestLinkDoesNotCreateRepoLinksJSON(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	primary := t.TempDir()
	peer := t.TempDir()
	seedCheckoutRecord(t, primary, &record.Record{
		ID: "rec_local", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "rec_local", Slug: "rec_local", Status: record.StatusAccepted,
		Body: "body", SourcePath: "docs/rec_local.md",
	})
	seedCheckoutRecord(t, peer, &record.Record{
		ID: "rec_remote", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "rec_remote", Slug: "rec_remote", Status: record.StatusAccepted,
		Body: "body", SourcePath: "docs/rec_remote.md",
	})
	cfg := config.Default()
	cfg.LogCommands = true
	if err := config.Save(primary, cfg); err != nil {
		t.Fatal(err)
	}

	runLink(t, primary, "link", "--kind", "record", "--from", "rec_local", "--to", peer+"#rec_remote")
	list := listRecordLinks(t, primary)
	if len(list) != 1 || list[0].Kind != "record" || list[0].FromID != "rec_local" || list[0].ToID != "rec_remote" {
		t.Fatalf("record %#v", list)
	}
	runLink(t, primary, "link", "--kind", "pin", "--to", peer+"#rec_remote")
	runLink(t, primary, "link", "--kind", "repo", "--from", "rec_local", "--to", peer)
	if list := listRecordLinks(t, primary); len(list) != 3 {
		t.Fatalf("three %#v", list)
	}
	runLink(t, primary, "unlink", "--kind", "pin", "--to", peer+"#rec_remote")
	if list := listRecordLinks(t, primary); len(list) != 2 {
		t.Fatalf("after unlink %#v", list)
	}

	path := filepath.Join(home, ".archivist", "repo-links.json")
	if _, err := os.Stat(path); err == nil {
		t.Fatal("created repo-links.json")
	} else if !os.IsNotExist(err) {
		t.Fatal(err)
	}

	data, err := os.ReadFile(config.CommandsLogPath(primary))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "archivist link") {
		t.Fatalf("log missing archivist link\n%s", data)
	}
}

func TestLinkCommandsAreRegistered(t *testing.T) {
	root := NewRoot()
	var link, unlink *cobra.Command
	for _, c := range root.Commands() {
		switch c.Name() {
		case "repos":
			t.Fatal("repos is registered")
		case "link":
			link = c
		case "unlink":
			unlink = c
		}
	}
	if link == nil || unlink == nil {
		t.Fatal("missing link or unlink")
	}
	var list *cobra.Command
	for _, c := range link.Commands() {
		if c.Name() == "list" {
			list = c
		}
	}
	if list == nil {
		t.Fatal("missing list")
	}
	if !archiveCommand(link) || !archiveCommand(unlink) || !archiveCommand(list) {
		t.Fatal("link commands are not logged")
	}
}

func runLink(t *testing.T, primary string, args ...string) {
	t.Helper()
	cmd := NewRoot()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(append([]string{"--path", primary}, args...))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
}

func listRecordLinks(t *testing.T, primary string) []store.Link {
	t.Helper()
	cmd := NewRoot()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--path", primary, "link", "list", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
	var got struct {
		Links []store.Link `json:"links"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
	return got.Links
}
