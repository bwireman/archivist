package codemap

import (
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/bwireman/archivist/internal/store"
)

// Docs and data files are scanned by the indexer but should not be mapped
// from English words that look like declarations ("class of", "type of").
var genericSkipExt = map[string]bool{
	".md": true, ".markdown": true, ".txt": true, ".rst": true, ".adoc": true,
	".json": true, ".jsonc": true, ".yaml": true, ".yml": true, ".toml": true,
	".ini": true, ".cfg": true, ".conf": true, ".csv": true,
	".html": true, ".htm": true, ".xml": true, ".svg": true,
	".css": true, ".scss": true, ".sass": true, ".less": true,
	".lock": true, ".sum": true, ".mod": true, ".log": true,
}

var genericNameStop = map[string]bool{
	"a": true, "an": true, "and": true, "be": true, "for": true, "from": true,
	"if": true, "in": true, "is": true, "it": true, "not": true, "of": true,
	"or": true, "that": true, "the": true, "this": true, "to": true, "with": true,
}

const genericModifiers = `(?:pub(?:lic|\([^)]+\))?|private|protected|internal|export|default|async|static|final|abstract|override|sealed|open|virtual|partial|unsafe|extern|mut|crate|readonly|required|inline|data|actual|expect|suspend|tailrec|operator|infix|reified|inner|value|native)`

const genericKeywords = `function|defmodule|typedef|interface|companion|protocol|object|struct|class|trait|union|enum|macro|actor|record|impl|module|proc|defp|func|fun|fn|def|type|sub`

var (
	genericDeclRe = regexp.MustCompile(`(?i)^(?:` + genericModifiers + `\s+)*(` + genericKeywords + `)\s+([A-Za-z_][A-Za-z0-9_]*)`)
	genericImpRe  = regexp.MustCompile(`(?i)^(?:export\s+)?(?:import|use|require|include|from)\s+([^\s;(]+)`)
)

func extractGeneric(path, content string) (Result, error) {
	if genericSkipExt[strings.ToLower(filepath.Ext(path))] {
		return Result{}, nil
	}
	lines := strings.Split(content, "\n")
	res := Result{}
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || isGenericComment(trimmed) {
			continue
		}
		if m := genericDeclRe.FindStringSubmatch(trimmed); m != nil {
			name := m[2]
			if genericNameStop[strings.ToLower(name)] {
				continue
			}
			res.Symbols = append(res.Symbols, store.Symbol{
				FilePath: path,
				Name:     name,
				Kind:     genericKind(m[1]),
				Line:     i + 1,
				DocLine:  firstDocLine(lines, i+1),
				Exported: genericExported(trimmed, name),
			})
			continue
		}
		if m := genericImpRe.FindStringSubmatch(trimmed); m != nil {
			res.Edges = append(res.Edges, store.SymbolEdge{
				FromFile: path,
				ToPath:   cleanImport(m[1]),
				EdgeType: "import",
			})
		}
	}
	return res, nil
}

func isGenericComment(trimmed string) bool {
	return strings.HasPrefix(trimmed, "//") ||
		strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, "/*") ||
		strings.HasPrefix(trimmed, "*") ||
		strings.HasPrefix(trimmed, "--")
}

func genericKind(kw string) string {
	switch strings.ToLower(kw) {
	case "func", "fn", "fun", "def", "defp", "function", "proc", "sub", "macro":
		return "function"
	default:
		return strings.ToLower(kw)
	}
}

func genericExported(trimmed, name string) bool {
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "export ") || strings.HasPrefix(lower, "pub ") ||
		strings.HasPrefix(lower, "public ") {
		return true
	}
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(r)
}
