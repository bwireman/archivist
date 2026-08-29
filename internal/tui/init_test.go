package tui

import (
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/config"
)

func TestSplitJoinList(t *testing.T) {
	items := []string{".git", "vendor", "node_modules"}
	if got := SplitList(JoinList(items)); strings.Join(got, ",") != strings.Join(items, ",") {
		t.Fatalf("round-trip: %v", got)
	}
	got := SplitList(" *.pb.go , foo.go\n bar.go ")
	if strings.Join(got, ",") != "*.pb.go,foo.go,bar.go" {
		t.Fatalf("split mixed: %v", got)
	}
	if len(SplitList("")) != 0 {
		t.Fatalf("empty: %v", SplitList(""))
	}
}

func TestConfigFromFormRoundTrip(t *testing.T) {
	want := config.Default()
	got, err := ConfigFromForm(FormFromConfig(want))
	if err != nil {
		t.Fatal(err)
	}
	if got.Ollama.BaseURL != want.Ollama.BaseURL || got.Ollama.EmbedModel != want.Ollama.EmbedModel {
		t.Fatalf("ollama mismatch: %#v", got.Ollama)
	}
	if strings.Join(got.Index.SkipDirs, ",") != strings.Join(want.Index.SkipDirs, ",") {
		t.Fatalf("skip dirs: %v", got.Index.SkipDirs)
	}
	if got.Store.Path != want.Store.Path {
		t.Fatalf("store: %s", got.Store.Path)
	}
	if strings.Join(got.Index.ADR.Global, ",") != strings.Join(want.Index.ADR.Global, ",") {
		t.Fatalf("global adr: %v", got.Index.ADR.Global)
	}
}

func TestConfigFromFormPreservesGlobalPath(t *testing.T) {
	cfg := config.Default()
	cfg.Store.GlobalPath = "custom-global.db"
	got, err := ConfigFromForm(FormFromConfig(cfg))
	if err != nil {
		t.Fatal(err)
	}
	if got.Store.GlobalPath != "custom-global.db" {
		t.Fatalf("global_path: %s", got.Store.GlobalPath)
	}
}

func TestConfigFromFormRejectsBadTimeout(t *testing.T) {
	form := FormFromConfig(config.Default())
	form.EmbedTimeout = "nope"
	if _, err := ConfigFromForm(form); err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestConfigFromFormCustom(t *testing.T) {
	got, err := ConfigFromForm(InitForm{
		BaseURL:        "http://ollama.example:11434",
		EmbedModel:     "qwen3-embedding:0.6b",
		EmbedTimeout:   "5m",
		SkipDirs:       ".git, vendor",
		SkipGlobs:      "*.pb.go",
		ADRPaths:       "docs/decisions/**",
		GlobalADRPaths: "docs/global-decisions/**",
		StorePath:      ".archivist/custom.db",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Ollama.EmbedModel != "qwen3-embedding:0.6b" {
		t.Fatalf("model: %s", got.Ollama.EmbedModel)
	}
	if got.Store.Path != ".archivist/custom.db" {
		t.Fatalf("store: %s", got.Store.Path)
	}
	if strings.Join(got.Index.SkipGlobs, ",") != "*.pb.go" {
		t.Fatalf("globs: %v", got.Index.SkipGlobs)
	}
	if strings.Join(got.Index.ADR.Global, ",") != "docs/global-decisions/**" {
		t.Fatalf("global adr: %v", got.Index.ADR.Global)
	}
	if strings.Join(got.Index.ADR.Repo, ",") != "docs/decisions/**" {
		t.Fatalf("repo adr: %v", got.Index.ADR.Repo)
	}
}

func TestFormatInitSummary(t *testing.T) {
	cfg := config.Default()
	created := FormatInitSummary(cfg, false)
	if !strings.Contains(created, "Created .archivist.json") || !strings.Contains(created, "ollama pull nomic-embed-text") {
		t.Fatalf("created:\n%s", created)
	}
	updated := FormatInitSummary(cfg, true)
	if !strings.Contains(updated, "Updated .archivist.json") {
		t.Fatalf("updated:\n%s", updated)
	}
}
