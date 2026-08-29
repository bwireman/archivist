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
			SkipDirs: []string{
				".git", "vendor", "node_modules", ".archivist", "dist", "build",
			},
			SkipGlobs: []string{"*.min.js", "*.pb.go"},
			ADR: config.ADRConfig{
				Repo: []string{
					"docs/decisions/**",
					"**/adr/**",
					"**/ADR*.md",
					"architecture/decisions/**",
				},
				Global: []string{
					"docs/global-decisions/**",
					"architecture/global-decisions/**",
				},
			},
		},
		Store: config.StoreConfig{
			Path:       ".archivist/custom-index.db",
			GlobalPath: "custom-global.db",
		},
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := config.Default()

	if cfg.Ollama.BaseURL != "http://localhost:11434" {
		t.Fatalf("ollama.base_url: got %q", cfg.Ollama.BaseURL)
	}
	if cfg.Ollama.EmbedModel != "nomic-embed-text" {
		t.Fatalf("ollama.embed_model: got %q", cfg.Ollama.EmbedModel)
	}
	if cfg.Ollama.EmbedTimeout != config.DefaultEmbedTimeoutStr {
		t.Fatalf("ollama.embed_timeout: got %q", cfg.Ollama.EmbedTimeout)
	}
	if cfg.Ollama.EmbedTimeoutDuration() != config.DefaultEmbedTimeout {
		t.Fatalf("embed timeout duration: got %s", cfg.Ollama.EmbedTimeoutDuration())
	}
	if !reflect.DeepEqual(cfg.Index.SkipDirs, []string{".git", "vendor", "node_modules", ".archivist"}) {
		t.Fatalf("index.skip_dirs: got %v", cfg.Index.SkipDirs)
	}
	if len(cfg.Index.SkipGlobs) != 0 {
		t.Fatalf("index.skip_globs: expected empty slice, got %v", cfg.Index.SkipGlobs)
	}
	if !reflect.DeepEqual(cfg.Index.ADR, config.Default().Index.ADR) {
		t.Fatalf("index.adr: got %#v", cfg.Index.ADR)
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

	for _, key := range []string{"ollama", "index", "store"} {
		if _, ok := keys[key]; !ok {
			t.Fatalf("saved config missing top-level key %q", key)
		}
	}

	var ollama map[string]json.RawMessage
	if err := json.Unmarshal(keys["ollama"], &ollama); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"base_url", "embed_model", "embed_timeout"} {
		if _, ok := ollama[key]; !ok {
			t.Fatalf("saved config missing ollama.%s", key)
		}
	}

	var index map[string]json.RawMessage
	if err := json.Unmarshal(keys["index"], &index); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"skip_dirs", "skip_globs", "adr"} {
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
	if _, ok := store["global_path"]; !ok {
		t.Fatal("saved config missing store.global_path")
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

func TestLoadOmitsADRUsesDefault(t *testing.T) {
	dir := t.TempDir()
	raw := []byte(`{
  "ollama": {
    "base_url": "http://localhost:11434",
    "embed_model": "nomic-embed-text"
  },
  "index": {
    "skip_dirs": [".git"],
    "skip_globs": []
  },
  "store": {
    "path": ".archivist/index.db"
  }
}`)
	if err := os.WriteFile(filepath.Join(dir, config.DefaultConfigName), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded.Index.ADR, config.Default().Index.ADR) {
		t.Fatalf("omitted index.adr: got %#v", loaded.Index.ADR)
	}
}

func TestLoadLegacyADRPaths(t *testing.T) {
	dir := t.TempDir()
	raw := []byte(`{
  "index": {
    "skip_dirs": [".git"],
    "skip_globs": [],
    "adr_paths": ["docs/decisions/**"],
    "global_adr_paths": ["docs/global-decisions/**"]
  },
  "store": { "path": ".archivist/index.db" }
}`)
	if err := os.WriteFile(filepath.Join(dir, config.DefaultConfigName), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(loaded.Index.ADR.Repo, []string{"docs/decisions/**"}) {
		t.Fatalf("legacy adr_paths: %v", loaded.Index.ADR.Repo)
	}
	if !reflect.DeepEqual(loaded.Index.ADR.Global, []string{"docs/global-decisions/**"}) {
		t.Fatalf("legacy global_adr_paths: %v", loaded.Index.ADR.Global)
	}
}

func TestVirtualUserADRPath(t *testing.T) {
	if got := config.VirtualUserADRPath("001-foo.md"); got != "user/001-foo.md" {
		t.Fatalf("got %q", got)
	}
	if got := config.VirtualUserADRPath("/nested/a.md"); got != "user/nested/a.md" {
		t.Fatalf("got %q", got)
	}
}

func TestIsUserGlobalPath(t *testing.T) {
	if !config.IsUserGlobalPath("user/001.md") {
		t.Fatal("expected user/001.md")
	}
	if config.IsUserGlobalPath("docs/global-decisions/001.md") {
		t.Fatal("in-repo global ADR is not a user path")
	}
}

func TestGlobalStorePath(t *testing.T) {
	cfg := config.Default()
	cfg.Store.GlobalPath = "/var/lib/archivist/global.db"
	if got := config.GlobalStorePath(cfg); got != "/var/lib/archivist/global.db" {
		t.Fatalf("absolute: got %q", got)
	}

	cfg.Store.GlobalPath = "custom-global.db"
	home := config.ArchivistHome()
	if home == "" {
		t.Skip("no home dir")
	}
	want := filepath.Join(home, "custom-global.db")
	if got := config.GlobalStorePath(cfg); got != want {
		t.Fatalf("relative: got %q want %q", got, want)
	}

	cfg.Store.GlobalPath = ""
	want = filepath.Join(home, config.DefaultGlobalDB)
	if got := config.GlobalStorePath(cfg); got != want {
		t.Fatalf("default: got %q want %q", got, want)
	}

	cfg.Store.GlobalPath = "../outside.db"
	if got := config.GlobalStorePath(cfg); got != "" {
		t.Fatalf("relative path escaping ~/.archivist should be empty, got %q", got)
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
    "embed_model": "nomic-embed-text"
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
