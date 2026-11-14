package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRelocateCheckoutFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".archivist"), 0o755); err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(root, ".archivist.json")
	body := []byte("{\n  \"ollama\": {\"embed_model\": \"qwen3-embedding:8b\", \"embed_num_ctx\": 32768},\n  \"log_commands\": true\n}\n")
	if err := os.WriteFile(legacy, body, 0o644); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(root, ".archivist", "commands.log")
	if err := os.WriteFile(logPath, []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RelocateCheckoutFiles(root); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(legacy); !os.IsNotExist(err) {
		t.Fatalf("legacy json still present: %v", err)
	}
	canon, err := Canonical(root)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(canon)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Ollama.EmbedModel != "qwen3-embedding:8b" || cfg.Ollama.EmbedNumCtx != 32768 || !cfg.LogCommands {
		t.Fatalf("loaded %#v", cfg.Ollama)
	}
	moved := filepath.Join(CheckoutDir(canon), "commands.log")
	got, err := os.ReadFile(moved)
	if err != nil || string(got) != "old\n" {
		t.Fatalf("log %q err=%v", got, err)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Fatal("old log remains")
	}
}
