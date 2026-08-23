package chunk

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

type languageSpec struct {
	lang     *sitter.Language
	nodeTypes []string
}

var languageSpecs = map[string]languageSpec{
	".go":   {lang: golang.GetLanguage(), nodeTypes: []string{"function_declaration", "method_declaration", "type_declaration", "type_spec"}},
	".py":   {lang: python.GetLanguage(), nodeTypes: []string{"function_definition", "class_definition"}},
	".js":   {lang: javascript.GetLanguage(), nodeTypes: []string{"function_declaration", "class_declaration", "method_definition", "arrow_function"}},
	".jsx":  {lang: javascript.GetLanguage(), nodeTypes: []string{"function_declaration", "class_declaration", "method_definition", "arrow_function"}},
	".ts":   {lang: typescript.GetLanguage(), nodeTypes: []string{"function_declaration", "class_declaration", "method_definition", "arrow_function"}},
	".tsx":  {lang: typescript.GetLanguage(), nodeTypes: []string{"function_declaration", "class_declaration", "method_definition", "arrow_function"}},
	".rs":   {lang: rust.GetLanguage(), nodeTypes: []string{"function_item", "impl_item", "struct_item", "enum_item", "trait_item"}},
	".java": {lang: java.GetLanguage(), nodeTypes: []string{"method_declaration", "class_declaration", "interface_declaration", "enum_declaration"}},
}

// SplitWithTreeSitter parses source with tree-sitter and emits one chunk per
// semantic node. Falls back to generic splitting when parsing fails.
func SplitWithTreeSitter(path, content string, chunkType Type) []Chunk {
	ext := strings.ToLower(filepath.Ext(path))
	spec, ok := languageSpecs[ext]
	if !ok {
		return SplitGeneric(path, content, chunkType, defaultMaxChunkSize, defaultOverlap)
	}

	parser := sitter.NewParser()
	parser.SetLanguage(spec.lang)
	tree, err := parser.ParseCtx(context.Background(), nil, []byte(content))
	if err != nil || tree == nil {
		return SplitGeneric(path, content, chunkType, defaultMaxChunkSize, defaultOverlap)
	}
	defer tree.Close()

	root := tree.RootNode()
	src := []byte(content)
	var chunks []Chunk

	// file header context
	headerEnd := min(30, len(strings.Split(content, "\n")))
	header := strings.Join(strings.Split(content, "\n")[:headerEnd], "\n")

	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}
		nt := node.Type()
		for _, want := range spec.nodeTypes {
			if nt == want {
				startLine := int(node.StartPoint().Row) + 1
				endLine := int(node.EndPoint().Row) + 1
				body := string(src[node.StartByte():node.EndByte()])
				text := fmt.Sprintf("File: %s\n\n%s\n\n%s", path, header, body)
				chunks = append(chunks, NewChunk(path, chunkType, startLine, endLine, text, map[string]string{
					"node_type": nt,
				}))
				return
			}
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}
	walk(root)

	if len(chunks) == 0 {
		return SplitGeneric(path, content, chunkType, defaultMaxChunkSize, defaultOverlap)
	}
	return chunks
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
