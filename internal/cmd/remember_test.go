package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/store"
)

func TestRememberArchiveWritesOnlyExtra(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	extra := t.TempDir()
	db, err := store.Open(config.StorePath(extra))
	if err != nil {
		t.Fatal(err)
	}
	extraCanon, err := config.Canonical(extra)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RegisterCheckout(extraCanon, config.CheckoutHex(extraCanon)); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	cmd := NewRoot()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{
		"--path", root, "remember",
		"--archive", extra,
		"--type", "decision", "--scope", "repo",
		"--title", "Extra only", "--body", "lives there",
	})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
	primary, err := store.Open(config.StorePath(root))
	if err != nil {
		t.Fatal(err)
	}
	defer primary.Close()
	rec, ok, err := primary.GetRecordBySlug("extra-only")
	if err != nil || !ok || rec.Checkout != extraCanon {
		t.Fatalf("remembered checkout ok=%v err=%v rec=%+v", ok, err, rec)
	}
}

func TestRememberMissingArchive(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	missing := t.TempDir()
	cfg := config.Default()
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	cmd := NewRoot()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{
		"--path", root, "remember",
		"--archive", missing,
		"--type", "decision", "--scope", "repo",
		"--title", "Nope", "--body", "no",
	})
	err := cmd.Execute()
	canon := mustCanonCmd(t, missing)
	if err == nil || (!strings.Contains(err.Error(), missing) && !strings.Contains(err.Error(), canon)) {
		t.Fatalf("err=%v", err)
	}
	if _, statErr := os.Stat(filepath.Join(missing, ".archivist")); !os.IsNotExist(statErr) {
		t.Fatal("remember created .archivist")
	}
}

func TestRememberFlagDocumented(t *testing.T) {
	if newRememberCmd().Flags().Lookup("archive") == nil {
		t.Fatal("missing --archive")
	}
}

func TestRememberCriticDisagreementKeepsCallerType(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"message":{"content":"{\"type\":\"rule\",\"lasting\":false,\"reason\":\"looks like a constraint\"}"}}`)
	}))
	t.Cleanup(srv.Close)
	cfg := config.Default()
	cfg.Ollama.BaseURL = srv.URL
	cfg.Ollama.EmbedModel = "qwen3-embedding:8b"
	cfg.Ollama.CriticModel = "gemma4:e2b"
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(config.StorePath(root))
	if err != nil {
		t.Fatal(err)
	}
	canon, err := config.Canonical(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RegisterCheckout(canon, config.CheckoutHex(canon)); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	cmd := NewRoot()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--path", root, "remember", "--json", "--type", "decision", "--scope", "repo", "--title", "Keep caller type", "--body", "The caller chose decision."})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
	var payload struct {
		ID       string         `json:"id"`
		Advisory map[string]any `json:"advisory"`
	}
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatalf("json %v %s", err, buf.String())
	}
	if payload.ID == "" || payload.Advisory["type"] != "rule" {
		t.Fatalf("payload %#v", payload)
	}
	if _, ok := payload.Advisory["lasting"]; ok {
		t.Fatalf("lasting was not wired: %#v", payload.Advisory)
	}
	opened, err := store.Open(config.StorePath(root))
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	rec, ok, err := opened.GetRecordByID(payload.ID)
	if err != nil || !ok {
		t.Fatalf("get %v %v", ok, err)
	}
	if rec.Type != "decision" {
		t.Fatalf("stored type %s", rec.Type)
	}
}

func TestRememberCriticErrorOmitsAdvisory(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		code int
	}{
		{name: "http", code: http.StatusInternalServerError, body: "down"},
		{name: "json", code: http.StatusOK, body: `{"message":{"content":"nope"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			root := t.TempDir()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.code)
				_, _ = io.WriteString(w, tc.body)
			}))
			t.Cleanup(srv.Close)
			cfg := config.Default()
			cfg.Ollama.BaseURL = srv.URL
			cfg.Ollama.EmbedModel = "qwen3-embedding:8b"
			cfg.Ollama.CriticModel = "gemma4:e2b"
			if err := config.Save(root, cfg); err != nil {
				t.Fatal(err)
			}
			db, err := store.Open(config.StorePath(root))
			if err != nil {
				t.Fatal(err)
			}
			canon, err := config.Canonical(root)
			if err != nil {
				t.Fatal(err)
			}
			if err := db.RegisterCheckout(canon, config.CheckoutHex(canon)); err != nil {
				t.Fatal(err)
			}
			_ = db.Close()
			var buf bytes.Buffer
			cmd := NewRoot()
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs([]string{"--path", root, "remember", "--json", "--type", "rule", "--scope", "repo", "--title", "Still written " + tc.name, "--body", "kept"})
			if err := cmd.Execute(); err != nil {
				t.Fatalf("%v\n%s", err, buf.String())
			}
			var payload map[string]any
			if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
				t.Fatalf("json %v %s", err, buf.String())
			}
			if payload["id"] == "" {
				t.Fatalf("payload %#v", payload)
			}
			if _, ok := payload["advisory"]; ok {
				t.Fatalf("advisory present %#v", payload)
			}
		})
	}
}

func TestRememberEmptyCriticModelOmitsAdvisory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("critic was called")
	}))
	t.Cleanup(srv.Close)
	cfg := config.Default()
	cfg.Ollama.BaseURL = srv.URL
	cfg.Ollama.EmbedModel = "qwen3-embedding:8b"
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	db, err := store.Open(config.StorePath(root))
	if err != nil {
		t.Fatal(err)
	}
	canon, err := config.Canonical(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.RegisterCheckout(canon, config.CheckoutHex(canon)); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	var buf bytes.Buffer
	cmd := NewRoot()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--path", root, "remember", "--json", "--type", "decision", "--scope", "repo", "--title", "No critic", "--body", "plain"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["id"] == "" {
		t.Fatalf("payload %#v", payload)
	}
	if _, ok := payload["advisory"]; ok {
		t.Fatalf("advisory %#v", payload)
	}
}
