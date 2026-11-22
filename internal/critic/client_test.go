package critic

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestStateCutsTo2000Bytes(t *testing.T) {
	short := State("Title", "Body")
	if short != "Title\n\nBody" {
		t.Fatalf("short %q", short)
	}
	got := State("T", strings.Repeat("x", 3000))
	if len(got) != 2000 {
		t.Fatalf("len %d", len(got))
	}
}

func TestJudgeRequestAndParse(t *testing.T) {
	var got chatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read: %v", err)
		}
		if !bytes.Contains(body, SchemaJSON()) {
			t.Errorf("format bytes missing schema")
		}
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("decode: %v", err)
		}
		_, _ = io.WriteString(w, `{"message":{"content":"{\"type\":\"rule\",\"lasting\":false,\"reason\":\"`+strings.Repeat("r", 250)+`\"}"}}`)
	}))
	t.Cleanup(srv.Close)

	c := &Client{BaseURL: srv.URL, Model: "gemma4:e2b", HTTP: srv.Client(), Prompt: PromptText(), Schema: SchemaJSON()}
	res, err := c.Judge(context.Background(), "Title", "Body")
	if err != nil {
		t.Fatal(err)
	}
	if got.Stream || got.Think || got.KeepAlive != 0 || got.Model != "gemma4:e2b" {
		t.Fatalf("request %#v", got)
	}
	if got.Options.Temperature != 0 {
		t.Fatalf("temperature %v", got.Options.Temperature)
	}
	if !res.OK || res.Type != "rule" || res.Lasting {
		t.Fatalf("result %#v", res)
	}
	if len([]rune(res.Reason)) != 200 {
		t.Fatalf("reason runes %d", len([]rune(res.Reason)))
	}
}

func TestJudgeOmitsBadContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"message":{"content":"not json","confidence":0.9}}`)
	}))
	t.Cleanup(srv.Close)
	c := &Client{BaseURL: srv.URL, Model: "gemma4:e2b", HTTP: srv.Client(), Prompt: "p", Schema: SchemaJSON()}
	res, err := c.Judge(context.Background(), "T", "B")
	if err != nil {
		t.Fatal(err)
	}
	if res.OK {
		t.Fatalf("expected omit, got %#v", res)
	}
}

func TestMissingModelWritesNoReport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, `{"models":[{"name":"qwen3-embedding:8b"}]}`)
	}))
	t.Cleanup(srv.Close)
	path := t.TempDir() + "/report.md"
	c := &Client{BaseURL: srv.URL, Model: "gemma4:e2b", HTTP: srv.Client()}
	err := Run(context.Background(), c, "qwen3-embedding:8b", nil, path)
	if err == nil {
		t.Fatal("expected error")
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("report exists: %v", statErr)
	}
}
