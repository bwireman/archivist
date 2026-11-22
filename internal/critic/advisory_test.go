package critic

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnabledReadsOnlyBeatLines(t *testing.T) {
	keepType, keepLasting := Enabled(PhaseReport())
	if !keepType || keepLasting {
		t.Fatalf("phase report type %v lasting %v", keepType, keepLasting)
	}
	keepType, keepLasting = Enabled("reeval type beat: yes\ntype beat: no\nlasting beat: yes\n")
	if keepType || !keepLasting {
		t.Fatalf("custom type %v lasting %v", keepType, keepLasting)
	}
}

func TestRememberAdvisoryOmitsOnErrorAndBadJSON(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			http.Error(w, "down", http.StatusInternalServerError)
			return
		}
		_, _ = io.WriteString(w, `{"message":{"content":"not-json"}}`)
	}))
	t.Cleanup(srv.Close)
	if adv := RememberAdvisory(context.Background(), srv.URL, "gemma4:e2b", "qwen3-embedding:8b", srv.Client(), "T", "B"); adv != nil {
		t.Fatalf("500 advisory %#v", adv)
	}
	if adv := RememberAdvisory(context.Background(), srv.URL, "gemma4:e2b", "qwen3-embedding:8b", srv.Client(), "T", "B"); adv != nil {
		t.Fatalf("bad json advisory %#v", adv)
	}
	if adv := RememberAdvisory(context.Background(), srv.URL, "", "qwen3-embedding:8b", srv.Client(), "T", "B"); adv != nil {
		t.Fatal("empty model called the critic")
	}
	if adv := RememberAdvisory(context.Background(), srv.URL, "qwen3-embedding:8b", "qwen3-embedding:8b", srv.Client(), "T", "B"); adv != nil {
		t.Fatal("embed model called the critic")
	}
	if calls != 2 {
		t.Fatalf("calls %d", calls)
	}
}
