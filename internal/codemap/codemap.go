package codemap

import (
	"context"
	"fmt"
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

// Version is the code-map extractor contract. Bump it when extractors change
// so the next full index remaps files whose content hashes have not changed.
const Version = 2

type languageSpec struct {
	lang    *sitter.Language
	symbols []string
	imports []string
	pkgNode string
}

var languageSpecs = map[string]languageSpec{
	".go": {
		lang:    golang.GetLanguage(),
		symbols: []string{"function_declaration", "method_declaration", "type_declaration", "type_spec", "interface_type"},
		imports: []string{"import_spec"},
		pkgNode: "package_clause",
	},
	".py": {
		lang:    python.GetLanguage(),
		symbols: []string{"function_definition", "class_definition"},
		imports: []string{"import_statement", "import_from_statement"},
	},
	".js":   {lang: javascript.GetLanguage(), symbols: []string{"function_declaration", "class_declaration", "method_definition"}, imports: []string{"import_statement"}},
	".jsx":  {lang: javascript.GetLanguage(), symbols: []string{"function_declaration", "class_declaration", "method_definition"}, imports: []string{"import_statement"}},
	".mjs":  {lang: javascript.GetLanguage(), symbols: []string{"function_declaration", "class_declaration", "method_definition"}, imports: []string{"import_statement"}},
	".cjs":  {lang: javascript.GetLanguage(), symbols: []string{"function_declaration", "class_declaration", "method_definition"}, imports: []string{"import_statement"}},
	".ts":   {lang: typescript.GetLanguage(), symbols: []string{"function_declaration", "class_declaration", "method_definition"}, imports: []string{"import_statement"}},
	".tsx":  {lang: typescript.GetLanguage(), symbols: []string{"function_declaration", "class_declaration", "method_definition"}, imports: []string{"import_statement"}},
	".mts":  {lang: typescript.GetLanguage(), symbols: []string{"function_declaration", "class_declaration", "method_definition"}, imports: []string{"import_statement"}},
	".cts":  {lang: typescript.GetLanguage(), symbols: []string{"function_declaration", "class_declaration", "method_definition"}, imports: []string{"import_statement"}},
	".rs":   {lang: rust.GetLanguage(), symbols: []string{"function_item", "struct_item", "enum_item", "trait_item", "impl_item"}, imports: []string{"use_declaration"}},
	".java": {lang: java.GetLanguage(), symbols: []string{"method_declaration", "class_declaration", "interface_declaration"}, imports: []string{"import_declaration"}},
}

type Result struct {
	PackageName string
	Symbols     []store.Symbol
	Edges       []store.SymbolEdge
}

// Extract parses a source file and returns structural map data only.
// Tree-sitter or Gleam extractors run first; a language-agnostic line
// matcher fills in when they error or return no symbols or imports.
func Extract(path, content string) (Result, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if spec, ok := languageSpecs[ext]; ok {
		res, err := extractTreeSitter(path, content, spec, ext)
		return withGenericBackup(path, content, res, err)
	}
	if ext == ".gleam" {
		res, err := extractGleam(path, content)
		return withGenericBackup(path, content, res, err)
	}
	return extractGeneric(path, content)
}

func withGenericBackup(path, content string, res Result, err error) (Result, error) {
	if err != nil {
		return extractGeneric(path, content)
	}
	if len(res.Symbols) > 0 || len(res.Edges) > 0 {
		return res, nil
	}
	gen, gerr := extractGeneric(path, content)
	if gerr != nil || (len(gen.Symbols) == 0 && len(gen.Edges) == 0) {
		return res, nil
	}
	gen.PackageName = res.PackageName
	return gen, nil
}

func extractTreeSitter(path, content string, spec languageSpec, ext string) (Result, error) {
	parser := sitter.NewParser()
	parser.SetLanguage(spec.lang)
	tree, err := parser.ParseCtx(context.Background(), nil, []byte(content))
	if err != nil || tree == nil {
		return Result{}, fmt.Errorf("parse %s", path)
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
	for i := symLine - 2; i >= 0 && i >= symLine-8; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "@") {
			continue
		}
		if strings.HasPrefix(line, "//") {
			doc := strings.TrimSpace(strings.TrimPrefix(line, "//"))
			return strings.TrimSpace(strings.TrimPrefix(doc, "/"))
		}
		if strings.HasPrefix(line, "#") {
			return strings.TrimSpace(strings.TrimPrefix(line, "#"))
		}
		if !strings.HasPrefix(line, "/*") {
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
