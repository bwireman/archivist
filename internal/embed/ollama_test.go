package embed

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOllamaClientEmbedAndHealth(t *testing.T) {
	var gotPrompt string
	var gotReq embedRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/tags":
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"models":[]}`)
		case "/api/embeddings":
			var req embedRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Errorf("decode: %v", err)
			}
			gotPrompt = req.Prompt
			gotReq = req
			_ = json.NewEncoder(w).Encode(embedResponse{Embedding: []float32{1, 0, 0}})
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	client := NewOllamaClientWithTimeout(srv.URL, "test-model", time.Second)
	if err := client.Healthy(context.Background()); err != nil {
		t.Fatal(err)
	}
	vec, err := client.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatal(err)
	}
	if gotPrompt != "hello" {
		t.Fatalf("prompt %q", gotPrompt)
	}
	if len(vec) != 3 || vec[0] != 1 {
		t.Fatalf("vec %v", vec)
	}
	if gotReq.Options != nil {
		t.Fatalf("expected no options, got %#v", gotReq.Options)
	}
}

func TestOllamaClientEmbedNumCtx(t *testing.T) {
	var gotReq embedRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/embeddings" {
			http.NotFound(w, r)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&gotReq); err != nil {
			t.Errorf("decode: %v", err)
		}
		_ = json.NewEncoder(w).Encode(embedResponse{Embedding: []float32{1}})
	}))
	t.Cleanup(srv.Close)

	client := NewOllamaClientWithTimeout(srv.URL, "test-model", time.Second)
	client.numCtx = 32768
	if _, err := client.Embed(context.Background(), "hello"); err != nil {
		t.Fatal(err)
	}
	if gotReq.Options == nil || gotReq.Options.NumCtx != 32768 {
		t.Fatalf("options: %#v", gotReq.Options)
	}
}

func TestOllamaClientEmptyVector(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(embedResponse{})
	}))
	t.Cleanup(srv.Close)
	client := NewOllamaClientWithTimeout(srv.URL, "test-model", time.Second)
	_, err := client.Embed(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected empty vector error")
	}
}
