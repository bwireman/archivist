package codemap

import (
	"context"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/typescript/typescript"

	"github.com/bwireman/archivist/internal/store"
)

type languageSpec struct {
	lang      *sitter.Language
	symbols   []string
	imports   []string
	pkgNode   string
}

var languageSpecs = map[string]languageSpec{
	".go": {
		lang: golang.GetLanguage(),
		symbols: []string{"function_declaration", "method_declaration", "type_declaration", "type_spec", "interface_type"},
		imports: []string{"import_spec"},
		pkgNode: "package_clause",
	},
	".py": {
		lang: python.GetLanguage(),
		symbols: []string{"function_definition", "class_definition"},
		imports: []string{"import_statement", "import_from_statement"},
	},
	".js":  {lang: javascript.GetLanguage(), symbols: []string{"function_declaration", "class_declaration", "method_definition"}, imports: []string{"import_statement"}},
	".jsx": {lang: javascript.GetLanguage(), symbols: []string{"function_declaration", "class_declaration", "method_definition"}, imports: []string{"import_statement"}},
	".ts":  {lang: typescript.GetLanguage(), symbols: []string{"function_declaration", "class_declaration", "method_definition"}, imports: []string{"import_statement"}},
	".tsx": {lang: typescript.GetLanguage(), symbols: []string{"function_declaration", "class_declaration", "method_definition"}, imports: []string{"import_statement"}},
	".rs":  {lang: rust.GetLanguage(), symbols: []string{"function_item", "struct_item", "enum_item", "trait_item", "impl_item"}, imports: []string{"use_declaration"}},
	".java": {lang: java.GetLanguage(), symbols: []string{"method_declaration", "class_declaration", "interface_declaration"}, imports: []string{"import_declaration"}},
}

type Result struct {
	PackageName string
	Symbols     []store.Symbol
	Edges       []store.SymbolEdge
}

// Extract parses a source file and returns structural map data only.
func Extract(path, content string) (Result, error) {
	ext := strings.ToLower(filepath.Ext(path))
	spec, ok := languageSpecs[ext]
	if !ok {
		return extractGeneric(path, content)
	}

	parser := sitter.NewParser()
	parser.SetLanguage(spec.lang)
	tree, err := parser.ParseCtx(context.Background(), nil, []byte(content))
	if err != nil || tree == nil {
		return extractGeneric(path, content)
	}
	defer tree.Close()

	root := tree.RootNode()
	src := []byte(content)
	lines := strings.Split(content, "\n")
	res := Result{}

	if spec.pkgNode != "" {
		res.PackageName = findPackage(root, src, spec.pkgNode, ext)
	}

	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil {
			return
		}
		nt := node.Type()
		for _, want := range spec.symbols {
			if nt == want {
				name := symbolName(node, src)
				if name == "" {
					continue
				}
				line := int(node.StartPoint().Row) + 1
				exported := isExported(name, ext)
				res.Symbols = append(res.Symbols, store.Symbol{
					FilePath: path,
					Name:     name,
					Kind:     nt,
					Line:     line,
					DocLine:  firstDocLine(lines, line),
					Exported: exported,
				})
			}
		}
		for _, want := range spec.imports {
			if nt == want {
				imp := strings.TrimSpace(string(src[node.StartByte():node.EndByte()]))
				res.Edges = append(res.Edges, store.SymbolEdge{
					FromFile: path,
					ToPath:   cleanImport(imp),
					EdgeType: "import",
				})
			}
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}
	walk(root)
	return res, nil
}

func extractGeneric(path, content string) (Result, error) {
	lines := strings.Split(content, "\n")
	var syms []store.Symbol
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "func ") || strings.HasPrefix(trimmed, "def ") ||
			strings.HasPrefix(trimmed, "class ") || strings.HasPrefix(trimmed, "export ") {
			syms = append(syms, store.Symbol{
				FilePath: path,
				Name:     trimmed,
				Kind:     "line",
				Line:     i + 1,
				Exported: strings.HasPrefix(trimmed, "export "),
			})
		}
	}
	return Result{Symbols: syms}, nil
}

func findPackage(root *sitter.Node, src []byte, pkgNode, ext string) string {
	var name string
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil || name != "" {
			return
		}
		if node.Type() == pkgNode {
			text := string(src[node.StartByte():node.EndByte()])
			if ext == ".go" {
				text = strings.TrimPrefix(text, "package ")
				name = strings.Fields(text)[0]
			}
			return
		}
		for i := 0; i < int(node.ChildCount()); i++ {
			walk(node.Child(i))
		}
	}
	walk(root)
	return name
}

func symbolName(node *sitter.Node, src []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		t := child.Type()
		if t == "identifier" || t == "type_identifier" || t == "property_identifier" {
			return string(src[child.StartByte():child.EndByte()])
		}
	}
	return ""
}

func isExported(name, ext string) bool {
	if ext == ".go" && name != "" {
		r, _ := utf8.DecodeRuneInString(name)
		return unicode.IsUpper(r)
	}
	return true
}

func firstDocLine(lines []string, symLine int) string {
	for i := symLine - 2; i >= 0 && i >= symLine-6; i-- {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "//") {
			return strings.TrimSpace(strings.TrimPrefix(line, "//"))
		}
		if strings.HasPrefix(line, "#") {
			return strings.TrimSpace(strings.TrimPrefix(line, "#"))
		}
		if line != "" && !strings.HasPrefix(line, "/*") {
			break
		}
	}
	return ""
}

func cleanImport(imp string) string {
	imp = strings.TrimSpace(imp)
	imp = strings.TrimPrefix(imp, "import ")
	imp = strings.TrimPrefix(imp, "from ")
	imp = strings.Trim(imp, "\"'`")
	return imp
}

// IsBinary reports whether data looks binary.
func IsBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	if !utf8.Valid(data) {
		return true
	}
	for _, b := range data {
		if b == 0 {
			return true
		}
	}
	return false
}
