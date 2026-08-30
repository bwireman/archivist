package chunk

import (
	"context"
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
	lang      *sitter.Language
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

const fileHeaderLines = 50

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

	lines := strings.Split(content, "\n")

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
				header := filePreamble(lines, startLine)
				lead := leadingCommentBlock(lines, startLine)
				var parts []string
				if header != "" {
					parts = append(parts, header)
				}
				if lead != "" {
					parts = append(parts, lead)
				}
				parts = append(parts, body)
				text := strings.Join(parts, "\n\n")
				parent := semanticParent(node, src)
				meta := map[string]string{"node_type": nt}
				if parent != "" {
					meta["parent"] = parent
				}
				annotated := annotateContent(path, chunkType, text, "Node", nt, "Parent", parent)
				if len(annotated) <= defaultMaxChunkSize {
					chunks = append(chunks, NewChunk(path, chunkType, startLine, endLine, annotated, meta))
				} else {
					pieces := splitBySize(path, annotated, chunkType, defaultMaxChunkSize, defaultOverlap, false, meta)
					for i := range pieces {
						pieces[i].StartLine = startLine
						pieces[i].EndLine = endLine
					}
					chunks = append(chunks, pieces...)
				}
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

func filePreamble(lines []string, startLine int) string {
	end := min(fileHeaderLines, len(lines))
	if startLine-1 < end {
		end = startLine - 1
	}
	if end <= 0 {
		return ""
	}
	i := end - 1
	for i >= 0 && strings.TrimSpace(lines[i]) == "" {
		i--
	}
	for i >= 0 && isDocCommentLine(lines[i]) {
		i--
	}
	end = i + 1
	if end <= 0 {
		return ""
	}
	if end >= fileHeaderLines {
		snap := end
		for snap > 0 && strings.TrimSpace(lines[snap-1]) != "" {
			snap--
		}
		if snap > 0 {
			end = snap
		}
	}
	if end <= 0 {
		return ""
	}
	return strings.TrimRight(strings.Join(lines[:end], "\n"), "\n")
}

func leadingCommentBlock(lines []string, startLine int) string {
	i := startLine - 2
	if i < 0 {
		return ""
	}
	for i >= 0 && strings.TrimSpace(lines[i]) == "" {
		i--
	}
	end := i
	if end < 0 || !isDocCommentLine(lines[end]) {
		return ""
	}
	for i >= 0 && isDocCommentLine(lines[i]) {
		i--
	}
	return strings.TrimRight(strings.Join(lines[i+1:end+1], "\n"), "\n")
}

func isDocCommentLine(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "//") ||
		strings.HasPrefix(t, "*") ||
		strings.HasPrefix(t, "/*") ||
		strings.HasPrefix(t, "#") ||
		strings.HasPrefix(t, "--")
}

func semanticParent(node *sitter.Node, src []byte) string {
	if node.Type() == "method_declaration" {
		if recv := node.ChildByFieldName("receiver"); recv != nil {
			if n := firstIdent(recv, src); n != "" {
				return n
			}
		}
	}
	for p := node.Parent(); p != nil; p = p.Parent() {
		switch p.Type() {
		case "class_declaration", "class_definition", "impl_item", "trait_item",
			"interface_declaration", "struct_item", "enum_item", "type_declaration",
			"type_spec":
			if n := nodeName(p, src); n != "" {
				return n
			}
		}
	}
	return ""
}

func nodeName(node *sitter.Node, src []byte) string {
	if node == nil {
		return ""
	}
	if n := node.ChildByFieldName("name"); n != nil {
		return nodeText(n, src)
	}
	return firstIdent(node, src)
}

func firstIdent(node *sitter.Node, src []byte) string {
	if found := findNodeType(node, src, "type_identifier"); found != "" {
		return found
	}
	if found := findNodeType(node, src, "identifier"); found != "" {
		return found
	}
	if found := findNodeType(node, src, "property_identifier"); found != "" {
		return found
	}
	return findNodeType(node, src, "field_identifier")
}

func findNodeType(node *sitter.Node, src []byte, typ string) string {
	if node == nil {
		return ""
	}
	if node.Type() == typ {
		return nodeText(node, src)
	}
	for i := 0; i < int(node.ChildCount()); i++ {
		if n := findNodeType(node.Child(i), src, typ); n != "" {
			return n
		}
	}
	return ""
}

func nodeText(node *sitter.Node, src []byte) string {
	if node == nil {
		return ""
	}
	return string(src[node.StartByte():node.EndByte()])
}
