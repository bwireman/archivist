package retrieve

import (
	"strings"
	"testing"
)

func TestSnippetCentersOnLateTerm(t *testing.T) {
	body := strings.Repeat("á", 300) + " retrieve"
	got := Snippet(body, "retrieve")
	if !strings.HasPrefix(got, "...") {
		t.Fatalf("snippet %q", got)
	}
	if !strings.Contains(got, "retrieve") {
		t.Fatalf("missing term %q", got)
	}
	rest := []rune(strings.TrimPrefix(got, "..."))
	idx := indexRunes(rest, []rune("retrieve"))
	if idx != 80 {
		t.Fatalf("retrieve rune index %d in %q", idx, got)
	}
}
