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
	chunks := chunk.SplitFile("sample.go", content, nil, nil)
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

func TestClassifyADRDoesNotMatchSiblingPrefix(t *testing.T) {
	typ := chunk.ClassifyFile("docs/decisions-old/001.md", []string{"docs/decisions/**"})
	if typ == chunk.TypeADR {
		t.Fatalf("sibling prefix should not be an ADR, got %s", typ)
	}
	if typ != chunk.TypeDoc {
		t.Fatalf("expected doc, got %s", typ)
	}
}

func TestClassifyADRGlob(t *testing.T) {
	if chunk.ClassifyFile("docs/ADR-001.md", []string{"**/ADR*.md"}) != chunk.TypeADR {
		t.Fatal("expected nested ADR*.md to match")
	}
}

func TestClassifyGlobalWins(t *testing.T) {
	typ, scope := chunk.Classify(
		"docs/global-decisions/001.md",
		[]string{"docs/decisions/**", "docs/global-decisions/**"},
		[]string{"docs/global-decisions/**"},
	)
	if typ != chunk.TypeADR || scope != chunk.ScopeGlobal {
		t.Fatalf("got %s %s", typ, scope)
	}
}

func TestClassifyRepoADR(t *testing.T) {
	typ, scope := chunk.Classify(
		"docs/decisions/001.md",
		[]string{"docs/decisions/**"},
		[]string{"docs/global-decisions/**"},
	)
	if typ != chunk.TypeADR || scope != chunk.ScopeRepo {
		t.Fatalf("got %s %s", typ, scope)
	}
}

func TestSplitFileStampsADRScope(t *testing.T) {
	content := "# Decision\n\nUse SQLite.\n"
	chunks := chunk.SplitFile(
		"docs/decisions/001.md",
		content,
		[]string{"docs/decisions/**"},
		[]string{"docs/global-decisions/**"},
	)
	if len(chunks) == 0 {
		t.Fatal("expected chunks")
	}
	for _, c := range chunks {
		if c.Type != chunk.TypeADR {
			t.Fatalf("type %s", c.Type)
		}
		if c.Metadata[chunk.MetaADRScope] != chunk.ScopeRepo {
			t.Fatalf("scope %v", c.Metadata)
		}
	}

	global := chunk.SplitFile(
		"docs/global-decisions/001.md",
		content,
		[]string{"docs/decisions/**"},
		[]string{"docs/global-decisions/**"},
	)
	if global[0].Metadata[chunk.MetaADRScope] != chunk.ScopeGlobal {
		t.Fatalf("global scope %v", global[0].Metadata)
	}
}

func TestStampOrigin(t *testing.T) {
	chunks := []chunk.Chunk{{Type: chunk.TypeADR, Metadata: map[string]string{chunk.MetaADRScope: chunk.ScopeGlobal}}}
	chunk.StampOrigin(chunks, chunk.OriginRepo, "/tmp/repo")
	if chunks[0].Metadata[chunk.MetaOrigin] != chunk.OriginRepo {
		t.Fatalf("origin %v", chunks[0].Metadata)
	}
	if chunks[0].Metadata[chunk.MetaOriginRoot] != "/tmp/repo" {
		t.Fatalf("origin_root %v", chunks[0].Metadata)
	}
}

func TestClassifyMDCAsDoc(t *testing.T) {
	if chunk.ClassifyFile(".cursor/rules/foo.mdc", nil) != chunk.TypeDoc {
		t.Fatal("expected .mdc to be classified as doc")
	}
}

func TestSplitADRUsesMarkdownHeadings(t *testing.T) {
	content := "# Decision\n\nUse SQLite.\n\n## Consequences\n\nSimple ops.\n"
	chunks := chunk.SplitGeneric("docs/decisions/001.md", content, chunk.TypeADR, 500, 0)
	if len(chunks) < 2 {
		t.Fatalf("expected heading split for ADR, got %d", len(chunks))
	}
	for _, c := range chunks {
		if c.Type != chunk.TypeADR {
			t.Fatalf("expected adr chunks, got %s", c.Type)
		}
	}
}

func TestMatchAnyPatternSkipGlobs(t *testing.T) {
	if !chunk.MatchAnyPattern("pkg/api.pb.go", []string{"*.pb.go"}) {
		t.Fatal("*.pb.go should match nested files")
	}
	if chunk.MatchAnyPattern("pkg/api.go", []string{"*.pb.go"}) {
		t.Fatal("*.pb.go should not match api.go")
	}
}

func TestExtractCommentChunksIgnoresAnnotation(t *testing.T) {
	content := "package main\n\n// TODO: fix this\n// ANNOTATION: keep\nfunc main() {}\n"
	chunks := chunk.ExtractCommentChunks("main.go", content)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 comment chunk, got %d (%v)", len(chunks), chunks)
	}
}
