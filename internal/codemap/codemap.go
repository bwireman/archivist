// Package codemap extracts symbols and imports from source files.
package codemap

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"
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
const Version = 5

type languageSpec struct {
	lang    *sitter.Language
	symbols []string
	imports []string
	pkgNode string
}

var (
	jsSpec = languageSpec{
		lang:    javascript.GetLanguage(),
		symbols: []string{"function_declaration", "class_declaration", "method_definition"},
		imports: []string{"import_statement"},
	}
	tsSpec = languageSpec{
		lang:    typescript.GetLanguage(),
		symbols: []string{"function_declaration", "class_declaration", "method_definition"},
		imports: []string{"import_statement"},
	}
	languageSpecs = map[string]languageSpec{
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
		".js": jsSpec, ".jsx": jsSpec, ".mjs": jsSpec, ".cjs": jsSpec,
		".ts": tsSpec, ".tsx": tsSpec, ".mts": tsSpec, ".cts": tsSpec,
		".rs":   {lang: rust.GetLanguage(), symbols: []string{"function_item", "struct_item", "enum_item", "trait_item", "impl_item"}, imports: []string{"use_declaration"}},
		".java": {lang: java.GetLanguage(), symbols: []string{"method_declaration", "class_declaration", "interface_declaration"}, imports: []string{"import_declaration"}},
	}
)

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
		return withGenericBackup(path, content, extractGleam(path, content), nil)
	}
	return extractGeneric(path, content), nil
}

func withGenericBackup(path, content string, res Result, err error) (Result, error) {
	if err != nil {
		return extractGeneric(path, content), nil
	}
	if len(res.Symbols) > 0 || len(res.Edges) > 0 {
		return res, nil
	}
	gen := extractGeneric(path, content)
	if len(gen.Symbols) == 0 && len(gen.Edges) == 0 {
		return res, nil
	}
	gen.PackageName = res.PackageName
	return gen, nil
}

var (
	parserMu sync.Mutex
	parsers  = map[*sitter.Language]*sitter.Parser{}
)

func parserFor(lang *sitter.Language) *sitter.Parser {
	parserMu.Lock()
	defer parserMu.Unlock()
	if p, ok := parsers[lang]; ok {
		return p
	}
	p := sitter.NewParser()
	p.SetLanguage(lang)
	parsers[lang] = p
	return p
}

func extractTreeSitter(path, content string, spec languageSpec, ext string) (Result, error) {
	parser := parserFor(spec.lang)
	tree, err := parser.ParseCtx(context.Background(), nil, []byte(content))
	if err != nil {
		return Result{}, fmt.Errorf("parse %s: %w", path, err)
	}
	defer tree.Close()

	root := tree.RootNode()
	src := []byte(content)
	lines := strings.Split(content, "\n")
	res := Result{}

	if spec.pkgNode != "" {
		res.PackageName = findPackage(root, src, spec.pkgNode)
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
				full := ""
				if ext == ".py" {
					full = pythonNodeDoc(node, src)
				}
				if full == "" {
					full = docBlock(lines, line)
				}
				summary, body := splitDoc(full)
				res.Symbols = append(res.Symbols, store.Symbol{
					FilePath: path,
					Name:     name,
					Kind:     nt,
					Line:     line,
					DocLine:  summary,
					Doc:      body,
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

func findPackage(root *sitter.Node, src []byte, pkgNode string) string {
	var name string
	var walk func(node *sitter.Node)
	walk = func(node *sitter.Node) {
		if node == nil || name != "" {
			return
		}
		if node.Type() == pkgNode {
			text := strings.TrimPrefix(string(src[node.StartByte():node.EndByte()]), "package ")
			fields := strings.Fields(text)
			if len(fields) > 0 {
				name = fields[0]
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

// symbolName is the declared name of node. Children are scanned in order so
// that a Go method's field_identifier wins over the type_identifier of its
// return value, which appears later in the same child list.
func symbolName(node *sitter.Node, src []byte) string {
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child == nil {
			continue
		}
		switch child.Type() {
		case "identifier", "field_identifier", "type_identifier", "property_identifier":
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

// maxDocLines caps a stored comment. Search matches this text; the summary
// line is still what doc_line shows.
const maxDocLines = 24

// docBlock is the comment on this symbol. A block above the declaration starts
// at its first line, not the line closest to the symbol. Python docstrings sit
// on the following line, including when the quotes are alone. Later lines stay
// in the block so search can see them. The summary is the first line.
func docBlock(lines []string, symLine int) string {
	if doc := commentBlock(lines, symLine); doc != "" {
		return doc
	}
	return docstringBelow(lines, symLine)
}

func docSummary(full string) string {
	full = strings.TrimSpace(full)
	if i := strings.IndexByte(full, '\n'); i >= 0 {
		return strings.TrimSpace(full[:i])
	}
	return full
}

// splitDoc returns the summary line and, when the comment continues, the whole
// block. The block is what search stores.
func splitDoc(full string) (summary, block string) {
	summary = docSummary(full)
	full = strings.TrimSpace(full)
	if full != summary {
		block = full
	}
	return summary, block
}

func capDoc(s string) string {
	s = strings.TrimSpace(s)
	lines := strings.Split(s, "\n")
	if len(lines) > maxDocLines {
		s = strings.Join(lines[:maxDocLines], "\n")
	}
	return strings.TrimSpace(s)
}

func commentBlock(lines []string, symLine int) string {
	i := symLine - 2
	floor := symLine - 8
	if floor < 0 {
		floor = 0
	}
	if i >= len(lines) {
		i = len(lines) - 1
	}
	for i >= floor {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "@") {
			i--
			continue
		}
		break
	}
	if i < floor {
		return ""
	}
	line := strings.TrimSpace(lines[i])
	switch {
	case strings.HasPrefix(line, "//"), strings.HasPrefix(line, "#"):
		prefix := "//"
		if strings.HasPrefix(line, "#") {
			prefix = "#"
		}
		start := i
		for start > 0 && strings.HasPrefix(strings.TrimSpace(lines[start-1]), prefix) {
			start--
		}
		var parts []string
		for n := start; n <= i; n++ {
			text := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(lines[n]), prefix))
			if prefix == "//" {
				text = strings.TrimSpace(strings.TrimPrefix(text, "/"))
			}
			if text != "" {
				parts = append(parts, text)
			}
		}
		return capDoc(strings.Join(parts, "\n"))
	case strings.Contains(line, "*/"):
		start := i
		for start > 0 && !strings.Contains(lines[start], "/*") {
			start--
		}
		if !strings.Contains(lines[start], "/*") {
			return ""
		}
		return capDoc(cleanBlockComment(lines[start : i+1]))
	default:
		return ""
	}
}

func cleanBlockComment(block []string) string {
	text := strings.Join(block, "\n")
	start := strings.Index(text, "/*")
	if start < 0 {
		return ""
	}
	text = text[start+2:]
	if end := strings.Index(text, "*/"); end >= 0 {
		text = text[:end]
	}
	var parts []string
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		if line != "" {
			parts = append(parts, line)
		}
	}
	return strings.Join(parts, "\n")
}

// pythonNodeDoc reads the docstring from the function or class body. A line
// scan misses it when the header is split across lines.
func pythonNodeDoc(node *sitter.Node, src []byte) string {
	block := childOfType(node, "block")
	if block == nil {
		return ""
	}
	for i := 0; i < int(block.ChildCount()); i++ {
		ch := block.Child(i)
		if ch == nil || ch.Type() == "comment" {
			continue
		}
		if ch.Type() != "expression_statement" {
			return ""
		}
		str := childOfType(ch, "string")
		if str == nil {
			return ""
		}
		return pythonDocText(string(src[str.StartByte():str.EndByte()]))
	}
	return ""
}

func childOfType(node *sitter.Node, typ string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		ch := node.Child(i)
		if ch != nil && ch.Type() == typ {
			return ch
		}
	}
	return nil
}

func pythonDocText(text string) string {
	text = strings.TrimSpace(text)
	quote := ""
	switch {
	case strings.HasPrefix(text, `"""`):
		quote = `"""`
	case strings.HasPrefix(text, `'''`):
		quote = `'''`
	default:
		return ""
	}
	text = strings.TrimPrefix(text, quote)
	text = strings.TrimSuffix(strings.TrimSpace(text), quote)
	var parts []string
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			parts = append(parts, line)
		}
	}
	return capDoc(strings.Join(parts, "\n"))
}

func docstringBelow(lines []string, symLine int) string {
	end := symLine + maxDocLines
	if end > len(lines) {
		end = len(lines)
	}
	for i := symLine; i < end; i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "@") {
			continue
		}
		quote := ""
		switch {
		case strings.HasPrefix(line, `"""`):
			quote = `"""`
		case strings.HasPrefix(line, `'''`):
			quote = `'''`
		default:
			return ""
		}
		var parts []string
		rest := strings.TrimPrefix(line, quote)
		if before, _, ok := strings.Cut(rest, quote); ok {
			if t := strings.TrimSpace(before); t != "" {
				parts = append(parts, t)
			}
			return capDoc(strings.Join(parts, "\n"))
		}
		if t := strings.TrimSpace(rest); t != "" {
			parts = append(parts, t)
		}
		for j := i + 1; j < end; j++ {
			next := strings.TrimSpace(lines[j])
			if before, _, ok := strings.Cut(next, quote); ok {
				if t := strings.TrimSpace(before); t != "" {
					parts = append(parts, t)
				}
				return capDoc(strings.Join(parts, "\n"))
			}
			if next != "" {
				parts = append(parts, next)
			}
		}
		return capDoc(strings.Join(parts, "\n"))
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
	return slices.Contains(data, 0)
}
