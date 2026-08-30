package chunk_test

import (
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/chunk"
)

func TestSplitGenericMarkdown(t *testing.T) {
	content := "# Title\n\nParagraph one.\n\n## Section\n\nParagraph two."
	chunks := chunk.SplitGeneric("docs/readme.md", content, chunk.TypeDoc, 500, 0)
	if len(chunks) != 1 {
		t.Fatalf("small markdown should stay one chunk, got %d", len(chunks))
	}
	if chunks[0].Type != chunk.TypeDoc {
		t.Fatalf("expected doc type, got %s", chunks[0].Type)
	}
	if !strings.Contains(chunks[0].Content, "File: docs/readme.md") {
		t.Fatalf("expected file annotation:\n%s", chunks[0].Content)
	}
	if !strings.Contains(chunks[0].Content, "Paragraph two.") {
		t.Fatalf("expected packed sections:\n%s", chunks[0].Content)
	}
}

func TestSplitGenericMarkdownSplitsWhenLarge(t *testing.T) {
	var b strings.Builder
	b.WriteString("# Title\n\n")
	b.WriteString(strings.Repeat("alpha ", 40))
	b.WriteString("\n\n## Section\n\n")
	b.WriteString(strings.Repeat("bravo ", 40))
	chunks := chunk.SplitGeneric("docs/readme.md", b.String(), chunk.TypeDoc, 200, 0)
	if len(chunks) < 2 {
		t.Fatalf("large markdown should split, got %d", len(chunks))
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
	if !strings.Contains(chunks[0].Content, "func main()") {
		t.Fatalf("expected surrounding code in comment chunk: %s", chunks[0].Content)
	}
	if !strings.Contains(chunks[0].Content, "File: main.go") {
		t.Fatalf("expected file annotation: %s", chunks[0].Content)
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
			if !strings.Contains(c.Content, "File: sample.go") {
				t.Fatalf("expected file annotation:\n%s", c.Content)
			}
			if !strings.Contains(c.Content, "package sample") {
				t.Fatalf("expected file preamble:\n%s", c.Content)
			}
		}
	}
	if !foundHello {
		t.Fatal("expected Hello function chunk")
	}
}

func TestSplitGoMethodIncludesParentAndDoc(t *testing.T) {
	content := `package sample

type Greeter struct{}

// Hello greets.
func (g *Greeter) Hello() string {
	return "hi"
}
`
	chunks := chunk.SplitFile("sample.go", content, nil, nil)
	var hello chunk.Chunk
	found := false
	for _, c := range chunks {
		if strings.Contains(c.Content, "func (g *Greeter) Hello()") {
			hello = c
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Hello method chunk, got %#v", chunks)
	}
	if hello.Metadata["parent"] != "Greeter" {
		t.Fatalf("parent metadata: %v", hello.Metadata)
	}
	if !strings.Contains(hello.Content, "Parent: Greeter") {
		t.Fatalf("expected parent in content:\n%s", hello.Content)
	}
	if !strings.Contains(hello.Content, "Hello greets.") {
		t.Fatalf("expected leading doc comment:\n%s", hello.Content)
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

func TestSplitADRStaysTogether(t *testing.T) {
	content := "# Decision\n\nUse SQLite.\n\n## Consequences\n\nSimple ops.\n"
	chunks := chunk.SplitGeneric("docs/decisions/001.md", content, chunk.TypeADR, 0, 0)
	if len(chunks) != 1 {
		t.Fatalf("small ADR should be one chunk, got %d", len(chunks))
	}
	if chunks[0].Type != chunk.TypeADR {
		t.Fatalf("expected adr chunks, got %s", chunks[0].Type)
	}
	if !strings.Contains(chunks[0].Content, "Use SQLite.") || !strings.Contains(chunks[0].Content, "Simple ops.") {
		t.Fatalf("expected packed ADR:\n%s", chunks[0].Content)
	}
	if !strings.Contains(chunks[0].Content, "File: docs/decisions/001.md") {
		t.Fatalf("expected file annotation:\n%s", chunks[0].Content)
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
