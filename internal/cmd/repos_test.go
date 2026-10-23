package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/cmdlog"
	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
	"github.com/spf13/cobra"
)

func TestReposLinkRoundTrip(t *testing.T) {
	realHome, err := os.UserHomeDir()
	if err != nil || realHome == "" {
		t.Fatal(err)
	}
	realLinks := filepath.Join(realHome, ".archivist", "repo-links.json")
	before, beforeErr := os.ReadFile(realLinks)
	if beforeErr != nil && !os.IsNotExist(beforeErr) {
		t.Fatal(beforeErr)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	primary := t.TempDir()
	peer := t.TempDir()
	openCheckout(t, primary)
	openCheckout(t, peer)
	id1 := seedCheckout(t, peer, "rec_one")
	id2 := seedCheckout(t, peer, "rec_two")
	cfg := config.Default()
	cfg.LogCommands = true
	if err := config.Save(primary, cfg); err != nil {
		t.Fatal(err)
	}

	run(t, primary, "link", "--kind", "related", peer)
	list := listLinks(t, primary)
	if len(list) != 1 || list[0].Kind != "related" {
		t.Fatalf("related %#v", list)
	}
	run(t, primary, "unlink", "--kind", "related", peer)
	if list := listLinks(t, primary); len(list) != 0 {
		t.Fatalf("after unlink %#v", list)
	}

	run(t, primary, "link", "--kind", "depends-on", "--record", peer+"#"+id1, peer)
	list = listLinks(t, primary)
	if len(list) != 1 || len(list[0].Records) != 1 || list[0].Records[0].ID != id1 {
		t.Fatalf("first record %#v", list)
	}
	run(t, primary, "link", "--kind", "depends-on", "--record", peer+"#"+id2, peer)
	list = listLinks(t, primary)
	if len(list) != 1 || len(list[0].Records) != 1 || list[0].Records[0].ID != id2 {
		t.Fatalf("replaced %#v", list)
	}

	after, afterErr := os.ReadFile(realLinks)
	if os.IsNotExist(beforeErr) {
		if afterErr == nil {
			t.Fatal("created the real repo-links.json")
		}
		if !os.IsNotExist(afterErr) {
			t.Fatal(afterErr)
		}
	} else if afterErr != nil || !bytes.Equal(before, after) {
		t.Fatal("rewrote the real repo-links.json")
	}

	data, err := os.ReadFile(config.CommandsLogPath(primary))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var entry cmdlog.Entry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatal(err)
		}
		if entry.Dir == "in" && strings.HasSuffix(entry.Command, "repos link") {
			found = true
		}
	}
	if !found {
		t.Fatalf("log missing repos link\n%s", data)
	}

	info, err := os.Stat(config.RepoLinksPath())
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode %o", info.Mode().Perm())
	}
}

func TestReposLinkCommandIsLogged(t *testing.T) {
	root := NewRoot()
	var repos *cobra.Command
	for _, c := range root.Commands() {
		if c.Name() == "repos" {
			repos = c
			break
		}
	}
	if repos == nil {
		t.Fatal("missing repos")
	}
	var link *cobra.Command
	for _, c := range repos.Commands() {
		if c.Name() == "propose" {
			t.Fatal("propose is registered")
		}
		if c.Name() == "link" {
			link = c
		}
	}
	if link == nil {
		t.Fatal("missing link")
	}
	if !archiveCommand(link) {
		t.Fatal("link is not an archive command")
	}
}

func run(t *testing.T, primary string, args ...string) {
	t.Helper()
	cmd := NewRoot()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(append([]string{"--path", primary, "repos"}, args...))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
}

func listLinks(t *testing.T, primary string) []struct {
	Kind    string `json:"kind"`
	Records []struct {
		ID string `json:"id"`
	} `json:"records"`
} {
	t.Helper()
	cmd := NewRoot()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--path", primary, "repos", "list", "--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
	var got struct {
		Links []struct {
			Kind    string `json:"kind"`
			Records []struct {
				ID string `json:"id"`
			} `json:"records"`
		} `json:"links"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
	return got.Links
}

func openCheckout(t *testing.T, dir string) {
	t.Helper()
	db, err := store.Open(config.StorePath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
}

func seedCheckout(t *testing.T, dir, id string) string {
	t.Helper()
	db, err := store.Open(config.StorePath(dir))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rec := &record.Record{
		ID: id, Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: id, Slug: id, Status: record.StatusAccepted,
		Body: "body", SourcePath: "docs/" + id + ".md",
	}
	if err := db.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	return id
}
