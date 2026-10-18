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

func TestSearchScopeFlagUnknownIndexScopeRemains(t *testing.T) {
	cmd := NewRoot()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"search", "--scope", "repo", "q"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(strings.ToLower(err.Error()), "unknown") {
		t.Fatalf("search --scope: err=%v stderr=%s", err, stderr.String())
	}

	cmd = NewRoot()
	stdout.Reset()
	stderr.Reset()
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"index", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("index --help: %v\n%s", err, stderr.String())
	}
	help := stdout.String()
	if !strings.Contains(help, "--scope") || !strings.Contains(help, "subdirectory") {
		t.Fatalf("index help: %s", help)
	}
}

func TestSearchCommandFullFlag(t *testing.T) {
	if newSearchCmd().Flags().Lookup("full") == nil {
		t.Fatal("missing --full")
	}
}

func TestMCPDocDocumentsCompactSearch(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "docs", "mcp.md"))
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
	if searchRow == "" || !strings.Contains(searchRow, "full") || !strings.Contains(searchRow, "snippet") {
		t.Fatalf("MCP search row: %s", searchRow)
	}
	var statusRow string
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, "| `status` |") {
			statusRow = line
			break
		}
	}
	if statusRow == "" || !strings.Contains(statusRow, "missing_vectors") {
		t.Fatalf("MCP status row: %s", statusRow)
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
	if !strings.Contains(compact, "BODYTOKEN") || strings.Contains(compact, "ContentHash") || strings.Contains(compact, `"body"`) {
		t.Fatalf("compact json: %s", compact)
	}
	logText := readCommandsLog(t, root)
	if !strings.Contains(logText, "Zqxcard fixture") || !strings.Contains(logText, "BODYTOKEN") || strings.Contains(logText, "ContentHash") {
		t.Fatalf("default search log: %s", logText)
	}
	textOut := runSearch(t, root, "Zqxcard")
	if !strings.Contains(textOut, "BODYTOKEN") {
		t.Fatalf("text search missing snippet: %s", textOut)
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

func TestSearchReadsListedArchive(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	extra := t.TempDir()
	seedCheckoutRecord(t, extra, &record.Record{
		ID: "rec_listed", Slug: "listed-hit", Type: record.TypeDecision, Scope: record.ScopeRepo,
		Title: "Listedhit token", Body: "from the extra", Status: record.StatusAccepted,
		SourcePath: "docs/decisions/listed-hit.md",
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
	ensureQueryStores(t, root)
	cmd := NewRoot()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--path", root, "search", "--json", "Listedhit"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "rec_listed") || !strings.Contains(stdout.String(), extra) && !strings.Contains(stdout.String(), mustCanonCmd(t, extra)) {
		t.Fatalf("search json: %s", stdout.String())
	}
}

func TestSearchMissingStoreAsksForInit(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	if err := config.Save(root, config.Default()); err != nil {
		t.Fatal(err)
	}
	cmd := NewRoot()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--path", root, "search", "q"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "archivist init") {
		t.Fatalf("err=%v\n%s", err, buf.String())
	}
	if _, statErr := os.Stat(config.StorePath(root)); !os.IsNotExist(statErr) {
		t.Fatal("search created the checkout database")
	}
}

func TestSearchReadsNonWritableStores(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	cfg := config.Default()
	cfg.LogCommands = true
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	repo, err := store.Open(config.StorePath(root))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertRecord(&record.Record{
		ID: "rec_repo_frozen", Slug: "repo-frozen", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "RepoFrozenToken", Body: "checkout", Status: record.StatusAccepted,
		SourcePath: "docs/features/repo-frozen.md",
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Close(); err != nil {
		t.Fatal(err)
	}
	homeDB, err := store.Open(config.HomeStorePath())
	if err != nil {
		t.Fatal(err)
	}
	if err := homeDB.UpsertRecord(&record.Record{
		ID: "rec_home_frozen", Slug: "home-frozen", Type: record.TypeDecision, Scope: record.ScopeGlobal,
		Title: "HomeFrozenToken", Body: "home", Status: record.StatusAccepted,
		SourcePath: "user/home-frozen.md",
	}); err != nil {
		t.Fatal(err)
	}
	if err := homeDB.Close(); err != nil {
		t.Fatal(err)
	}

	homeDir := filepath.Dir(config.HomeStorePath())
	t.Cleanup(func() { _ = os.Chmod(homeDir, 0o755) })
	if err := os.Chmod(homeDir, 0o555); err != nil {
		t.Fatal(err)
	}
	homeHit := runSearch(t, root, "--json", "HomeFrozenToken")
	if !strings.Contains(homeHit, "HomeFrozenToken") {
		t.Fatalf("home search: %s", homeHit)
	}
	if err := os.Chmod(homeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	dataDir := config.DataDir(root)
	if err := os.Remove(config.CommandsLogPath(root)); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dataDir, 0o755) })
	if err := os.Chmod(dataDir, 0o555); err != nil {
		t.Fatal(err)
	}
	repoHit := runSearch(t, root, "--json", "RepoFrozenToken")
	if !strings.Contains(repoHit, "RepoFrozenToken") {
		t.Fatalf("repo search: %s", repoHit)
	}
	if _, err := os.Stat(config.CommandsLogPath(root)); !os.IsNotExist(err) {
		t.Fatal("query wrote commands.log on a read-only checkout")
	}
}

func TestSearchMissingExtraErrors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	missing := t.TempDir()
	cfg := config.Default()
	cfg.Archives = []string{missing}
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	ensureQueryStores(t, root)
	cmd := NewRoot()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--path", root, "search", "anything"})
	err := cmd.Execute()
	canon := mustCanonCmd(t, missing)
	if err == nil || (!strings.Contains(err.Error(), missing) && !strings.Contains(err.Error(), canon)) {
		t.Fatalf("err=%v", err)
	}
	if _, statErr := os.Stat(filepath.Join(missing, ".archivist")); !os.IsNotExist(statErr) {
		t.Fatal("search created .archivist")
	}
}

func mustCanonCmd(t *testing.T, path string) string {
	t.Helper()
	got, err := config.Canonical(path)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func ensureQueryStores(t *testing.T, root string) {
	t.Helper()
	repo, home, err := openStores(root)
	if err != nil {
		t.Fatal(err)
	}
	err = repo.Close()
	if cerr := home.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		t.Fatal(err)
	}
}

func seedCheckoutRecord(t *testing.T, dir string, rec *record.Record) {
	t.Helper()
	db, err := store.Open(config.StorePath(dir))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
}

func readCommandsLog(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(config.CommandsLogPath(root))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
