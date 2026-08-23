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
		Provider: "cursor",
		Ollama: config.OllamaConfig{
			BaseURL:         "http://ollama.example:11434",
			EmbedModel:      "mxbai-embed-large",
			GenerateModel:   "qwen2.5:7b",
			EmbedTimeout:    "5m",
			GenerateTimeout: "45m",
		},
		Cursor: config.CursorConfig{
			APIKey:          "cursor_test_key",
			BaseURL:         "https://api.cursor.example",
			Model:           "composer-2.5",
			GenerateTimeout: "45m",
			PollInterval:    "3s",
		},
		Docs: config.DocsConfig{
			Root: "documentation",
			Pages: map[string]string{
				"cmd/**":      "documentation/cli.md",
				"internal/**": "documentation/internals.md",
				"pkg/**":      "documentation/packages.md",
			},
		},
		Index: config.IndexConfig{
			SkipDirs: []string{
				".git", "vendor", "node_modules", ".archivist", "dist", "build",
			},
			SkipGlobs: []string{"*.min.js", "*.pb.go"},
			ADRPaths: []string{
				"**/adr/**",
				"docs/decisions/**",
				"**/ADR*.md",
				"architecture/decisions/**",
			},
		},
		Store: config.StoreConfig{
			Path: ".archivist/custom-index.db",
		},
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := config.Default()

	if cfg.Provider != config.DefaultProvider {
		t.Fatalf("provider: got %q", cfg.Provider)
	}
	if cfg.Ollama.BaseURL != "http://localhost:11434" {
		t.Fatalf("ollama.base_url: got %q", cfg.Ollama.BaseURL)
	}
	if cfg.Ollama.EmbedModel != "nomic-embed-text" {
		t.Fatalf("ollama.embed_model: got %q", cfg.Ollama.EmbedModel)
	}
	if cfg.Ollama.GenerateModel != "llama3.1" {
		t.Fatalf("ollama.generate_model: got %q", cfg.Ollama.GenerateModel)
	}
	if cfg.Ollama.EmbedTimeout != config.DefaultEmbedTimeoutStr {
		t.Fatalf("ollama.embed_timeout: got %q", cfg.Ollama.EmbedTimeout)
	}
	if cfg.Ollama.GenerateTimeout != config.DefaultGenerateTimeoutStr {
		t.Fatalf("ollama.generate_timeout: got %q", cfg.Ollama.GenerateTimeout)
	}
	if cfg.Ollama.GenerateTimeoutDuration() != config.DefaultGenerateTimeout {
		t.Fatalf("generate timeout duration: got %s", cfg.Ollama.GenerateTimeoutDuration())
	}
	if cfg.Cursor.BaseURL != config.DefaultCursorBaseURL {
		t.Fatalf("cursor.base_url: got %q", cfg.Cursor.BaseURL)
	}
	if cfg.Cursor.Model != config.DefaultCursorModel {
		t.Fatalf("cursor.model: got %q", cfg.Cursor.Model)
	}
	if cfg.Docs.Root != "docs" {
		t.Fatalf("docs.root: got %q", cfg.Docs.Root)
	}
	if cfg.Docs.Pages == nil {
		t.Fatal("docs.pages: expected non-nil map")
	}
	if len(cfg.Docs.Pages) != 0 {
		t.Fatalf("docs.pages: expected empty map, got %v", cfg.Docs.Pages)
	}
	if !reflect.DeepEqual(cfg.Index.SkipDirs, []string{".git", "vendor", "node_modules", ".archivist"}) {
		t.Fatalf("index.skip_dirs: got %v", cfg.Index.SkipDirs)
	}
	if len(cfg.Index.SkipGlobs) != 0 {
		t.Fatalf("index.skip_globs: expected empty slice, got %v", cfg.Index.SkipGlobs)
	}
	if !reflect.DeepEqual(cfg.Index.ADRPaths, []string{"**/adr/**", "docs/decisions/**", "**/ADR*.md"}) {
		t.Fatalf("index.adr_paths: got %v", cfg.Index.ADRPaths)
	}
	if cfg.Store.Path != filepath.Join(config.DefaultDataDir, config.DefaultIndexDB) {
		t.Fatalf("store.path: got %q", cfg.Store.Path)
	}
}

func TestLoadFullConfigFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "full.json"))
	if err != nil {
		t.Fatal(err)
	}

	var got config.Config
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}

	want := fullConfig()
	if !reflect.DeepEqual(&got, want) {
		t.Fatalf("loaded config mismatch:\nwant: %#v\ngot:  %#v", want, &got)
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
		t.Fatalf("loaded config mismatch:\nwant: %#v\ngot:  %#v", want, loaded)
	}
}

func TestSaveLoadRoundTripAllKeys(t *testing.T) {
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
		t.Fatalf("round-trip config mismatch:\nwant: %#v\ngot:  %#v", want, loaded)
	}

	raw, err := os.ReadFile(filepath.Join(dir, config.DefaultConfigName))
	if err != nil {
		t.Fatal(err)
	}

	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatal(err)
	}

	for _, key := range []string{"provider", "ollama", "cursor", "docs", "index", "store"} {
		if _, ok := keys[key]; !ok {
			t.Fatalf("saved config missing top-level key %q", key)
		}
	}

	var cursor map[string]json.RawMessage
	if err := json.Unmarshal(keys["cursor"], &cursor); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"base_url", "model", "generate_timeout", "poll_interval"} {
		if _, ok := cursor[key]; !ok {
			t.Fatalf("saved config missing cursor.%s", key)
		}
	}

	var ollama map[string]json.RawMessage
	if err := json.Unmarshal(keys["ollama"], &ollama); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"base_url", "embed_model", "generate_model", "embed_timeout", "generate_timeout"} {
		if _, ok := ollama[key]; !ok {
			t.Fatalf("saved config missing ollama.%s", key)
		}
	}

	var docs map[string]json.RawMessage
	if err := json.Unmarshal(keys["docs"], &docs); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"root", "pages"} {
		if _, ok := docs[key]; !ok {
			t.Fatalf("saved config missing docs.%s", key)
		}
	}

	var index map[string]json.RawMessage
	if err := json.Unmarshal(keys["index"], &index); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"skip_dirs", "skip_globs", "adr_paths"} {
		if _, ok := index[key]; !ok {
			t.Fatalf("saved config missing index.%s", key)
		}
	}

	var store map[string]json.RawMessage
	if err := json.Unmarshal(keys["store"], &store); err != nil {
		t.Fatal(err)
	}
	if _, ok := store["path"]; !ok {
		t.Fatal("saved config missing store.path")
	}
}

func TestLoadMissingConfigUsesDefaults(t *testing.T) {
	dir := t.TempDir()
	loaded, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded, config.Default()) {
		t.Fatalf("expected defaults when config file is missing")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.DefaultConfigName), []byte("{invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(dir); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestStorePath(t *testing.T) {
	dir := t.TempDir()
	cfg := fullConfig()

	rel := config.StorePath(dir, cfg)
	if rel != filepath.Join(dir, ".archivist/custom-index.db") {
		t.Fatalf("relative store path: got %q", rel)
	}

	cfg.Store.Path = "/var/lib/archivist/index.db"
	abs := config.StorePath(dir, cfg)
	if abs != "/var/lib/archivist/index.db" {
		t.Fatalf("absolute store path: got %q", abs)
	}
}

func TestDataDir(t *testing.T) {
	dir := t.TempDir()
	if got := config.DataDir(dir); got != filepath.Join(dir, ".archivist") {
		t.Fatalf("data dir: got %q", got)
	}
}

func TestLoadEmptyStorePathFallsBackToDefault(t *testing.T) {
	dir := t.TempDir()
	raw := []byte(`{
  "ollama": {
    "base_url": "http://localhost:11434",
    "embed_model": "nomic-embed-text",
    "generate_model": "llama3.1"
  },
  "docs": {
    "root": "docs",
    "pages": {}
  },
  "index": {
    "skip_dirs": [".git"],
    "skip_globs": [],
    "adr_paths": ["**/adr/**"]
  },
  "store": {
    "path": ""
  }
}`)
	if err := os.WriteFile(filepath.Join(dir, config.DefaultConfigName), raw, 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Store.Path != filepath.Join(config.DefaultDataDir, config.DefaultIndexDB) {
		t.Fatalf("store.path fallback: got %q", loaded.Store.Path)
	}
}
