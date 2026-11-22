package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/store"
)

func TestDecideInsertsNoRow(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"message":{"content":"{\"type\":\"rule\",\"lasting\":false,\"reason\":\"constraint\"}"}}`)
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
	before, err := db.RecordCount()
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	cmd := NewRoot()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--path", root, "decide", "--title", "Candidate", "--body", "A must that later work should follow."})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("%v\n%s", err, buf.String())
	}
	var adv map[string]any
	if err := json.Unmarshal(buf.Bytes(), &adv); err != nil {
		t.Fatalf("json %v %s", err, buf.String())
	}
	if adv["type"] != "rule" {
		t.Fatalf("advisory %#v", adv)
	}
	opened, err := store.Open(config.StorePath(root))
	if err != nil {
		t.Fatal(err)
	}
	defer opened.Close()
	after, err := opened.RecordCount()
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("rows %d to %d", before, after)
	}
}

func TestDecideEmptyAndErrorPrintEmptyObject(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	cfg := config.Default()
	cfg.Ollama.BaseURL = srv.URL
	cfg.Ollama.EmbedModel = "qwen3-embedding:8b"
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	cmd := NewRoot()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--path", root, "decide", "--title", "Candidate", "--body", "text"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "{}\n" {
		t.Fatalf("empty model %q", buf.String())
	}
	cfg.Ollama.CriticModel = "gemma4:e2b"
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	cmd = NewRoot()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--path", root, "decide", "--title", "Candidate", "--body", "text"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "{}\n" {
		t.Fatalf("error %q", buf.String())
	}
}
