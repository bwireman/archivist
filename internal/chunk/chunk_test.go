package chunk_test

import (
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/chunk"
)

func TestSplitGenericMarkdown(t *testing.T) {
	content := "# Title\n\nParagraph one.\n\n## Section\n\nParagraph two."
	chunks := chunk.SplitGeneric("docs/readme.md", content, chunk.TypeDoc, 500, 0)
	if len(chunks) < 2 {
		t.Fatalf("expected multiple chunks, got %d", len(chunks))
	}
	if chunks[0].Type != chunk.TypeDoc {
		t.Fatalf("expected doc type, got %s", chunks[0].Type)
	}
}

func TestExtractCommentChunks(t *testing.T) {
	content := "package main\n\n// TODO: fix this\nfunc main() {}\n"
	chunks := chunk.ExtractCommentChunks("main.go", content)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 comment chunk, got %d", len(chunks))
	}
	if !strings.Contains(chunks[0].Content, "TODO") {
		t.Fatalf("expected TODO in content: %s", chunks[0].Content)
	}
}

func TestSplitGoFile(t *testing.T) {
	content := `package sample

func Hello() string {
	return "hi"
}

func World() string {
	return "world"
}
`
	chunks := chunk.SplitFile("sample.go", content, nil)
	if len(chunks) == 0 {
		t.Fatal("expected tree-sitter chunks for Go file")
	}
	foundHello := false
	for _, c := range chunks {
		if strings.Contains(c.Content, "Hello") {
			foundHello = true
		}
	}
	if !foundHello {
		t.Fatal("expected Hello function chunk")
	}
}

func TestClassifyADR(t *testing.T) {
	typ := chunk.ClassifyFile("docs/decisions/001-use-sqlite.md", []string{"docs/decisions/**"})
	if typ != chunk.TypeADR {
		t.Fatalf("expected adr, got %s", typ)
	}
}
