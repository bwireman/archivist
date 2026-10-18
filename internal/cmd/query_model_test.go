package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/retrieve"
)

func TestCLISearchAndCheckSendQwenPrefix(t *testing.T) {
	var mu sync.Mutex
	var prompts []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"models":[]}`)
		case "/api/embeddings":
			var body struct {
				Prompt string `json:"prompt"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			mu.Lock()
			prompts = append(prompts, body.Prompt)
			mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			_, _ = io.WriteString(w, `{"embedding":[1,0]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	ensureQueryStores(t, root)
	cfg := config.Default()
	cfg.Ollama.EmbedModel = "qwen3-embedding:8b"
	cfg.Ollama.BaseURL = srv.URL
	cfg.Ollama.EmbedTimeout = "2s"
	if err := config.Save(root, cfg); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) {
		t.Helper()
		cmd := NewRoot()
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SetArgs(append([]string{"--path", root}, args...))
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
	}
	run("search", "hybrid")
	run("check", "hybrid rules")

	mu.Lock()
	defer mu.Unlock()
	want := []string{
		retrieve.QueryText("qwen3-embedding:8b", "hybrid"),
		retrieve.QueryText("qwen3-embedding:8b", "hybrid rules"),
	}
	if len(prompts) != len(want) {
		t.Fatalf("prompts %q", prompts)
	}
	for i := range want {
		if prompts[i] != want[i] {
			t.Fatalf("prompt %d %q want %q", i, prompts[i], want[i])
		}
	}
}
