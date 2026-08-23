package embed_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/embed"
)

func TestCursorClientGenerate(t *testing.T) {
	var archived bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/me":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"email":"test@example.com"}`))
		case r.Method == http.MethodPost && r.URL.Path == "/v1/agents":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"agent": map[string]string{"id": "bc-test"},
				"run":   map[string]string{"id": "run-test", "status": "RUNNING"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/agents/bc-test/runs/run-test":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":     "run-test",
				"status": "FINISHED",
				"result": "# Generated Doc\n\nFrom Cursor API.",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/agents/bc-test/archive":
			archived = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	client, err := embed.NewCursorClient(config.CursorConfig{
		APIKey:       "test-key",
		BaseURL:      srv.URL,
		Model:        "composer-2.5",
		PollInterval: "1ms",
	})
	if err != nil {
		t.Fatal(err)
	}

	out, err := client.Generate(context.Background(), "write docs")
	if err != nil {
		t.Fatal(err)
	}
	if out != "# Generated Doc\n\nFrom Cursor API." {
		t.Fatalf("unexpected output: %q", out)
	}
	if !archived {
		t.Fatal("expected agent archive call")
	}
}

func TestNewProvidersCursor(t *testing.T) {
	t.Setenv(config.CursorAPIKeyEnv, "test-key")
	cfg := config.Default()
	cfg.Provider = "cursor"

	providers, err := embed.NewProviders(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if providers.Embedder == nil || providers.Generator == nil {
		t.Fatal("expected embedder and generator")
	}
}

func TestNewProvidersCursorMissingKey(t *testing.T) {
	t.Setenv(config.CursorAPIKeyEnv, "")
	cfg := config.Default()
	cfg.Provider = "cursor"

	if _, err := embed.NewProviders(cfg); err == nil {
		t.Fatal("expected error without cursor api key")
	}
}

func TestCursorClientHealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/me" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		username, _, ok := r.BasicAuth()
		if !ok || username != "secret" {
			t.Fatalf("expected basic auth, got %q", username)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client, err := embed.NewCursorClient(config.CursorConfig{
		APIKey:  "secret",
		BaseURL: srv.URL,
		Model:   "composer-2.5",
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Healthy(ctx); err != nil {
		t.Fatal(err)
	}
}
