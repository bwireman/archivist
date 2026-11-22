package config_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
			Repo:      "docs/decisions",
			Global:    "docs/global-decisions",
			Dev:       "notes",
			Export:    "docs/archive",
			WriteDocs: true,
		},
		Publish: config.PublishConfig{
			Destinations: map[string]config.PublishDestination{
				"wiki": {Command: []string{"./scripts/push.sh", "{{bundle}}"}},
			},
		},
		LogCommands: true,
	}
}

func TestDefaultConfig(t *testing.T) {
	t.Setenv("OLLAMA_HOST", "")
	cfg := config.Default()
	if cfg.Ollama.BaseURL != "" {
		t.Fatalf("base_url: %q want empty", cfg.Ollama.BaseURL)
	}
	if cfg.Ollama.ResolvedBaseURL() != config.DefaultOllamaURL {
		t.Fatalf("resolved: %q", cfg.Ollama.ResolvedBaseURL())
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
	if cfg.Ollama.EmbedNumCtx != 0 {
		t.Fatalf("embed_num_ctx: %d want 0", cfg.Ollama.EmbedNumCtx)
	}
	if len(cfg.Index.SkipGlobs) != 0 {
		t.Fatalf("skip_globs: %v", cfg.Index.SkipGlobs)
	}
	if cfg.Records.Repo != config.DefaultDecisionsDir {
		t.Fatalf("records.repo: %q", cfg.Records.Repo)
	}
	if cfg.Records.Global != "" {
		t.Fatalf("records.global: %q want empty home default", cfg.Records.Global)
	}
	if cfg.Records.Export != config.DefaultArchiveDir {
		t.Fatalf("records.export: %q", cfg.Records.Export)
	}
	if cfg.Records.WriteDocs {
		t.Fatal("records.write_docs should default off")
	}
	if cfg.LogCommands {
		t.Fatal("log_commands should default off")
	}
}

func TestLoadFullConfigFromDisk(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
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
	if _, err := os.Stat(filepath.Join(dir, config.DefaultConfigName)); !os.IsNotExist(err) {
		t.Fatal("legacy config should be removed")
	}
	if _, err := os.Stat(config.GlobalConfigPath()); err != nil {
		t.Fatal(err)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
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
	if _, err := os.Stat(filepath.Join(dir, config.DefaultConfigName)); !os.IsNotExist(err) {
		t.Fatal("save should not leave .archivist.json in the checkout")
	}
	raw, err := os.ReadFile(config.GlobalConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"ollama", "index", "records", "publish", "log_commands"} {
		if _, ok := keys[key]; !ok {
			t.Fatalf("missing key %q", key)
		}
	}
}

func TestLoadMissingConfigUsesDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
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
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, config.DefaultConfigName), []byte("{invalid"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(dir); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadEmbedNumCtx(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	raw := []byte(`{"ollama": {"embed_num_ctx": 32768}}`)
	if err := os.WriteFile(filepath.Join(dir, config.DefaultConfigName), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Ollama.EmbedNumCtx != 32768 {
		t.Fatalf("embed_num_ctx: %d", loaded.Ollama.EmbedNumCtx)
	}
}

func TestLoadOmitsRecordsUsesDefaults(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
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
	if loaded.Records.Global != "" {
		t.Fatalf("global: %q want empty home default", loaded.Records.Global)
	}
	if loaded.Records.WriteDocs {
		t.Fatal("omitted write_docs should stay off")
	}
	if loaded.Ollama.EmbedModel != "nomic-embed-text" {
		t.Fatalf("model: %q", loaded.Ollama.EmbedModel)
	}
	if loaded.Ollama.BaseURL != "" {
		t.Fatalf("base_url: %q", loaded.Ollama.BaseURL)
	}
}

func TestLoadWriteDocsTrue(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	raw := []byte(`{"records": {"write_docs": true}}`)
	if err := os.WriteFile(filepath.Join(dir, config.DefaultConfigName), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.Records.WriteDocs {
		t.Fatal("expected write_docs true")
	}
	if loaded.Records.Export != config.DefaultArchiveDir {
		t.Fatalf("export path: %q", loaded.Records.Export)
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

func TestIsHomeGlobalPath(t *testing.T) {
	if !config.IsHomeGlobalPath("global/001.md") {
		t.Fatal("expected home global path")
	}
	if config.IsHomeGlobalPath("docs/global-decisions/001.md") {
		t.Fatal("in-repo global is not a home-global virtual path")
	}
	if config.IsHomeGlobalPath("user/001.md") {
		t.Fatal("dev path is not home-global")
	}
}

func TestGlobalDirDefaultsToArchivistHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	var r config.RecordsConfig
	got := r.GlobalDir("/repo")
	want := filepath.Join(home, ".archivist")
	if got != want {
		t.Fatalf("empty global: %q want %q", got, want)
	}
	if r.GlobalInRepo() {
		t.Fatal("empty global is not in-repo")
	}
	r.Global = "docs/global-decisions"
	if !r.GlobalInRepo() {
		t.Fatal("relative global is in-repo")
	}
	if got := r.GlobalDir("/repo"); got != filepath.Join("/repo", "docs/global-decisions") {
		t.Fatalf("in-repo dir: %q", got)
	}
	r.Global = "~/.archivist"
	if r.GlobalInRepo() {
		t.Fatal("~/ path is not in-repo")
	}
	if got := r.GlobalDir("/repo"); got != want {
		t.Fatalf("~/.archivist: %q want %q", got, want)
	}
}

func TestStorePaths(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	got := config.StorePath(dir)
	want := filepath.Join(home, ".archivist", "archive.db")
	if got != want {
		t.Fatalf("store: %q want %q", got, want)
	}
	if legacy := config.LegacyIndexPath(dir); legacy != filepath.Join(dir, ".archivist", "index.db") {
		t.Fatalf("legacy: %q", legacy)
	}
	if got := config.CommandsLogPath(dir); got != filepath.Join(config.CheckoutDir(dir), "commands.log") {
		t.Fatalf("commands log: %q", got)
	}
	if home == "" {
		t.Skip("no home")
	}
	if got := config.HomeStorePath(); got != want {
		t.Fatalf("home store: %q", got)
	}
}

func TestLoadEmptyBaseURLStaysEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	for _, raw := range []string{
		`{"ollama": {"base_url": ""}}`,
		`{"ollama": {"base_url": "   "}}`,
	} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, config.DefaultConfigName), []byte(raw), 0o644); err != nil {
			t.Fatal(err)
		}
		loaded, err := config.Load(dir)
		if err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(loaded.Ollama.BaseURL) != "" {
			t.Fatalf("stored base_url %q for %s", loaded.Ollama.BaseURL, raw)
		}
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

func TestCanonicalResolvesSymlink(t *testing.T) {
	real := t.TempDir()
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}
	got, err := config.Canonical(link)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(real)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("canonical %q want %q", got, want)
	}
}

func TestCanonicalMissingPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "new-repo")
	got, err := config.Canonical(missing)
	if err != nil {
		t.Fatal(err)
	}
	if got != missing {
		t.Fatalf("canonical %q want %q", got, missing)
	}
}

func TestDataDir(t *testing.T) {
	dir := t.TempDir()
	if got := config.DataDir(dir); got != filepath.Join(dir, ".archivist") {
		t.Fatalf("data dir: %q", got)
	}
}

func TestLoadIgnoresArchivesKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	primary := t.TempDir()
	other := filepath.Join(t.TempDir(), "not-opened")
	raw := []byte(`{"archives":["` + other + `"],"ollama":{"embed_model":"qwen3-embedding:0.6b"}}`)
	if err := os.WriteFile(filepath.Join(primary, config.DefaultConfigName), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(primary); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(other); !os.IsNotExist(err) {
		t.Fatal("archives path was created")
	}
}

func mustCanon(t *testing.T, path string) string {
	t.Helper()
	got, err := config.Canonical(path)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestLoadPromotesLegacyHomeAndCheckout(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	canon := mustCanon(t, root)
	if err := os.MkdirAll(filepath.Dir(config.GlobalConfigPath()), 0o755); err != nil {
		t.Fatal(err)
	}
	homeBody := []byte("{\n  \"base_url\": \"http://localhost:11434\",\n  \"embed_model\": \"qwen3-embedding:8b\",\n  \"embed_timeout\": \"5m\",\n  \"embed_num_ctx\": 32768\n}\n")
	if err := os.WriteFile(config.GlobalConfigPath(), homeBody, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(config.CheckoutDir(canon), 0o755); err != nil {
		t.Fatal(err)
	}
	checkoutPath := filepath.Join(config.CheckoutDir(canon), "config.json")
	checkoutBody := []byte(`{
	  "ollama": {"base_url": "", "embed_model": "", "embed_num_ctx": 8192},
	  "index": {"skip_globs": ["testdata/efficacy-fixture/**"]},
	  "records": {"repo": "docs/decisions", "global": "docs/global-decisions", "export": "docs/archive"},
	  "log_commands": true
	}`)
	if err := os.WriteFile(checkoutPath, checkoutBody, 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := config.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Ollama.EmbedModel != "qwen3-embedding:8b" || loaded.Ollama.EmbedNumCtx != 32768 {
		t.Fatalf("ollama %#v", loaded.Ollama)
	}
	if !loaded.LogCommands || loaded.Records.Global != "docs/global-decisions" {
		t.Fatalf("records %#v log=%v", loaded.Records, loaded.LogCommands)
	}
	if len(loaded.Index.SkipGlobs) != 1 || loaded.Index.SkipGlobs[0] != "testdata/efficacy-fixture/**" {
		t.Fatalf("globs %#v", loaded.Index.SkipGlobs)
	}
	if _, err := os.Stat(checkoutPath); !os.IsNotExist(err) {
		t.Fatal("checkout config should be removed")
	}
	raw, err := os.ReadFile(config.GlobalConfigPath())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"log_commands"`) || !strings.Contains(string(raw), `"ollama"`) {
		t.Fatalf("global file %s", raw)
	}

	other := t.TempDir()
	otherCanon := mustCanon(t, other)
	if err := os.MkdirAll(config.CheckoutDir(otherCanon), 0o755); err != nil {
		t.Fatal(err)
	}
	otherPath := filepath.Join(config.CheckoutDir(otherCanon), "config.json")
	otherBody := []byte(`{"ollama":{"base_url":"","embed_model":""},"records":{"repo":"docs/decisions","export":"docs/archive"}}`)
	if err := os.WriteFile(otherPath, otherBody, 0o644); err != nil {
		t.Fatal(err)
	}
	again, err := config.Load(other)
	if err != nil {
		t.Fatal(err)
	}
	if again.Ollama.EmbedModel != "qwen3-embedding:8b" || !again.LogCommands || again.Records.Global != "docs/global-decisions" {
		t.Fatalf("second checkout clobbered %#v log=%v", again.Ollama, again.LogCommands)
	}
	if _, err := os.Stat(otherPath); !os.IsNotExist(err) {
		t.Fatal("second checkout config should be removed")
	}
}

func TestFinishLoadLeavesCriticModelEmpty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := config.GlobalConfigPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"ollama":{"embed_model":""}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Ollama.CriticModel != "" {
		t.Fatalf("critic_model %q", loaded.Ollama.CriticModel)
	}
	if loaded.Ollama.EmbedModel != config.DefaultEmbedModel {
		t.Fatalf("embed_model %q", loaded.Ollama.EmbedModel)
	}
}

func TestNormalizeOllamaHostWithoutScheme(t *testing.T) {
	cfg := config.OllamaConfig{BaseURL: "localhost:11434"}
	if got := cfg.ResolvedBaseURL(); got != "http://localhost:11434" {
		t.Fatalf("got %q", got)
	}
}
