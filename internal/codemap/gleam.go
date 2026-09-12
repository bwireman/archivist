package codemap

import (
	"regexp"
	"strings"

	"github.com/bwireman/archivist/internal/store"
)

var (
	gleamImport = regexp.MustCompile(`^import\s+([A-Za-z0-9_/]+)`)
	gleamFn     = regexp.MustCompile(`^(?:pub\s+)?fn\s+([A-Za-z_][A-Za-z0-9_]*)`)
	gleamType   = regexp.MustCompile(`^(?:pub\s+)?(?:opaque\s+)?type\s+([A-Za-z_][A-Za-z0-9_]*)`)
	gleamConst  = regexp.MustCompile(`^(?:pub\s+)?const\s+([A-Za-z_][A-Za-z0-9_]*)`)
	gleamCtor   = regexp.MustCompile(`^([A-Z][A-Za-z0-9_]*)\s*(?:\(|$)`)
)

// extractGleam maps top-level Gleam declarations. go-tree-sitter does not
// ship a Gleam grammar, so this runs before the language-agnostic fallback.
func extractGleam(path, content string) (Result, error) {
	lines := strings.Split(content, "\n")
	res := Result{}
	inType := false
	inPubType := false
	depth := 0

	for i, line := range lines {
		lineNo := i + 1
		trimmed := strings.TrimSpace(line)
		braces := strings.Count(line, "{") - strings.Count(line, "}")
		indented := len(line) > 0 && (line[0] == ' ' || line[0] == '\t')

		if inType {
			depth += braces
			if trimmed != "" && !strings.HasPrefix(trimmed, "//") {
				if m := gleamCtor.FindStringSubmatch(trimmed); m != nil {
					res.Symbols = append(res.Symbols, gleamSym(path, m[1], "constructor", lineNo, inPubType, lines))
				}
			}
			if depth <= 0 {
				inType = false
				inPubType = false
				depth = 0
			}
			continue
		}

		if trimmed == "" || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "@") || indented {
			continue
		}

		if m := gleamImport.FindStringSubmatch(trimmed); m != nil {
			res.Edges = append(res.Edges, store.SymbolEdge{
				FromFile: path,
				ToPath:   m[1],
				EdgeType: "import",
			})
			continue
		}
		if m := gleamFn.FindStringSubmatch(trimmed); m != nil {
			res.Symbols = append(res.Symbols, gleamSym(path, m[1], "function", lineNo, strings.HasPrefix(trimmed, "pub "), lines))
			continue
		}
		if m := gleamType.FindStringSubmatch(trimmed); m != nil {
			pub := strings.HasPrefix(trimmed, "pub ")
			res.Symbols = append(res.Symbols, gleamSym(path, m[1], "type", lineNo, pub, lines))
			if strings.Contains(trimmed, "{") {
				inType = true
				inPubType = pub
				depth = braces
				if depth <= 0 {
					inType = false
					inPubType = false
					depth = 0
				}
			}
			continue
		}
		if m := gleamConst.FindStringSubmatch(trimmed); m != nil {
			res.Symbols = append(res.Symbols, gleamSym(path, m[1], "constant", lineNo, strings.HasPrefix(trimmed, "pub "), lines))
		}
	}
	return res, nil
}

func gleamSym(path, name, kind string, line int, exported bool, lines []string) store.Symbol {
	return store.Symbol{
		FilePath: path,
		Name:     name,
		Kind:     kind,
		Line:     line,
		DocLine:  firstDocLine(lines, line),
		Exported: exported,
	}
}
