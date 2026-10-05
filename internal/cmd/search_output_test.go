package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/retrieve"
	"github.com/bwireman/archivist/internal/store"
)

func TestSearchCommandFullFlag(t *testing.T) {
	if newSearchCmd().Flags().Lookup("full") == nil {
		t.Fatal("missing --full")
	}
}

func TestReadmeDocumentsCompactSearch(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(b)
	if !strings.Contains(text, "--full") {
		t.Fatal("README missing --full")
	}
	var searchRow string
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "| `search` |") {
			searchRow = line
			break
		}
	}
	if searchRow == "" || !strings.Contains(searchRow, "full") {
		t.Fatalf("MCP search row: %s", searchRow)
	}
}

func TestSearchJSONCompactFullAndLog(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	orig := repoPath
	t.Cleanup(func() { repoPath = orig })

	cfg := config.Default()
	cfg.LogCommands = true
	cfg.Ollama.BaseURL = "http://127.0.0.1:1"
	cfg.Ollama.EmbedTimeout = "1s"
	if err := applyInit(root, cfg); err != nil {
		t.Fatal(err)
	}
	repoDB, err := store.Open(config.StorePath(root))
	if err != nil {
		t.Fatal(err)
	}
	rec := &record.Record{
		ID:         record.NewID(),
		Slug:       "zqxcard-fixture",
		Type:       record.TypeFeature,
		Scope:      record.ScopeRepo,
		Title:      "Zqxcard fixture",
		Status:     record.StatusAccepted,
		Body:       "BODYTOKEN lives here",
		SourcePath: "docs/decisions/zqxcard-fixture.md",
	}
	if err := repoDB.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
	if err := repoDB.Close(); err != nil {
		t.Fatal(err)
	}

	compact := runSearch(t, root, "--json", "Zqxcard")
	if strings.Contains(compact, "BODYTOKEN") || strings.Contains(compact, "ContentHash") {
		t.Fatalf("compact json: %s", compact)
	}
	logText := readCommandsLog(t, root)
	if !strings.Contains(logText, "Zqxcard fixture") || strings.Contains(logText, "ContentHash") {
		t.Fatalf("default search log: %s", logText)
	}

	full := runSearch(t, root, "--json", "--full", "Zqxcard")
	if !strings.Contains(full, "BODYTOKEN") || !strings.Contains(full, "ContentHash") {
		t.Fatalf("full json: %s", full)
	}
	logText = readCommandsLog(t, root)
	if !strings.Contains(logText, "ContentHash") {
		t.Fatalf("full search log: %s", logText)
	}

	homeDB, err := store.Open(config.HomeStorePath())
	if err != nil {
		t.Fatal(err)
	}
	defer homeDB.Close()
	repoDB, err = store.Open(config.StorePath(root))
	if err != nil {
		t.Fatal(err)
	}
	defer repoDB.Close()
	results, err := (&retrieve.Engine{Repo: repoDB, Home: homeDB}).Search(context.Background(), nil, retrieve.Options{Query: "Zqxcard"})
	if err != nil {
		t.Fatal(err)
	}
	want := retrieve.FormatResults(results)
	if !strings.Contains(want, "Zqxcard fixture") {
		t.Fatalf("format: %s", want)
	}
	text := runSearch(t, root, "Zqxcard")
	textFull := runSearch(t, root, "--full", "Zqxcard")
	if text != want || textFull != want {
		t.Fatalf("text %q full %q want %q", text, textFull, want)
	}
}

func runSearch(t *testing.T, root string, args ...string) string {
	t.Helper()
	cmd := NewRoot()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(append([]string{"--path", root, "search"}, args...))
	if err := cmd.Execute(); err != nil {
		t.Fatalf("search %v: %v\n%s", args, err, stderr.String())
	}
	return stdout.String()
}

func readCommandsLog(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(config.CommandsLogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
