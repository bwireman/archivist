package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/bwireman/archivist/internal/config"
)

func fullConfig() *config.Config {
	return &config.Config{
		Ollama: config.OllamaConfig{
			BaseURL:      "http://ollama.example:11434",
			EmbedModel:   "mxbai-embed-large",
			EmbedTimeout: "5m",
		},
		Index: config.IndexConfig{
			SkipGlobs: []string{"*.min.js", "*.pb.go"},
		},
		Records: config.RecordsConfig{
			Repo:   "docs/decisions",
			Global: "docs/global-decisions",
			Dev:    "notes",
			Export: "docs/archive",
		},
		Publish: config.PublishConfig{
			Destinations: map[string]config.PublishDestination{
				"wiki": {Command: []string{"./scripts/push.sh", "{{bundle}}"}},
			},
		},
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := config.Default()
	if cfg.Ollama.BaseURL != config.DefaultOllamaURL {
		t.Fatalf("base_url: %q", cfg.Ollama.BaseURL)
	}
	if cfg.Ollama.EmbedModel != config.DefaultEmbedModel {
		t.Fatalf("embed_model: %q", cfg.Ollama.EmbedModel)
	}
	if cfg.Ollama.EmbedTimeout != config.DefaultEmbedTimeoutStr {
		t.Fatalf("embed_timeout: %q", cfg.Ollama.EmbedTimeout)
	}
	if cfg.Ollama.EmbedTimeoutDuration() != config.DefaultEmbedTimeout {
		t.Fatalf("timeout duration: %s", cfg.Ollama.EmbedTimeoutDuration())
	}
	if len(cfg.Index.SkipGlobs) != 0 {
		t.Fatalf("skip_globs: %v", cfg.Index.SkipGlobs)
	}
	if cfg.Records.Repo != config.DefaultDecisionsDir {
		t.Fatalf("records.repo: %q", cfg.Records.Repo)
	}
	if cfg.Records.Global != config.DefaultGlobalDecisionsDir {
		t.Fatalf("records.global: %q", cfg.Records.Global)
	}
	if cfg.Records.Export != config.DefaultArchiveDir {
		t.Fatalf("records.export: %q", cfg.Records.Export)
	}
}

func TestLoadFullConfigFromDisk(t *testing.T) {
	dir := t.TempDir()
	data, err := os.ReadFile(filepath.Join("testdata", "full.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, config.DefaultConfigName), data, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := fullConfig()
	if !reflect.DeepEqual(loaded, want) {
		t.Fatalf("want %#v\ngot %#v", want, loaded)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := fullConfig()
	if err := config.Save(dir, want); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, want) {
		t.Fatalf("want %#v\ngot %#v", want, loaded)
	}
	raw, err := os.ReadFile(filepath.Join(dir, config.DefaultConfigName))
	if err != nil {
		t.Fatal(err)
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"ollama", "index", "records", "publish"} {
		if _, ok := keys[key]; !ok {
			t.Fatalf("missing key %q", key)
		}
	}
}

func TestLoadMissingConfigUsesDefaults(t *testing.T) {
	dir := t.TempDir()
	loaded, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, config.Default()) {
		t.Fatal("expected defaults")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.DefaultConfigName), []byte("{invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(dir); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadOmitsRecordsUsesDefaults(t *testing.T) {
	dir := t.TempDir()
	raw := []byte(`{"ollama": {"embed_model": "nomic-embed-text"}}`)
	if err := os.WriteFile(filepath.Join(dir, config.DefaultConfigName), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Records.Repo != config.DefaultDecisionsDir {
		t.Fatalf("repo: %q", loaded.Records.Repo)
	}
	if loaded.Ollama.EmbedModel != "nomic-embed-text" {
		t.Fatalf("model: %q", loaded.Ollama.EmbedModel)
	}
}

func TestPathUnder(t *testing.T) {
	if !config.PathUnder("docs/decisions/001.md", "docs/decisions") {
		t.Fatal("expected under")
	}
	if config.PathUnder("docs/other/001.md", "docs/decisions") {
		t.Fatal("not under")
	}
}

func TestVirtualUserADRPath(t *testing.T) {
	if got := config.VirtualUserADRPath("001-foo.md"); got != "user/001-foo.md" {
		t.Fatalf("got %q", got)
	}
}

func TestIsUserGlobalPath(t *testing.T) {
	if !config.IsUserGlobalPath("user/001.md") {
		t.Fatal("expected user path")
	}
	if config.IsUserGlobalPath("docs/global-decisions/001.md") {
		t.Fatal("in-repo global is not a user path")
	}
}

func TestStorePaths(t *testing.T) {
	dir := t.TempDir()
	got := config.StorePath(dir)
	want := filepath.Join(dir, ".archivist", "index.db")
	if got != want {
		t.Fatalf("store: %q want %q", got, want)
	}
	home := config.ArchivistHome()
	if home == "" {
		t.Skip("no home")
	}
	if got := config.HomeStorePath(); got != filepath.Join(home, "archive.db") {
		t.Fatalf("home store: %q", got)
	}
}

func TestResolvedBaseURLEnv(t *testing.T) {
	t.Setenv("OLLAMA_HOST", "127.0.0.1:11434")
	empty := config.OllamaConfig{}
	if got := empty.ResolvedBaseURL(); got != "http://127.0.0.1:11434" {
		t.Fatalf("env: %q", got)
	}
	set := config.OllamaConfig{BaseURL: "http://ollama.example:11434"}
	if got := set.ResolvedBaseURL(); got != "http://ollama.example:11434" {
		t.Fatalf("explicit: %q", got)
	}
}

func TestDataDir(t *testing.T) {
	dir := t.TempDir()
	if got := config.DataDir(dir); got != filepath.Join(dir, ".archivist") {
		t.Fatalf("data dir: %q", got)
	}
}

func TestNormalizeOllamaHostWithoutScheme(t *testing.T) {
	cfg := config.OllamaConfig{BaseURL: "localhost:11434"}
	if got := cfg.ResolvedBaseURL(); got != "http://localhost:11434" {
		t.Fatalf("got %q", got)
	}
}
