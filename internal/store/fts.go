package store

import (
	"strings"
	"unicode"
)

// fts5Query turns a natural-language or path-like string into an FTS5 MATCH
// expression. Punctuation is tokenized away so it cannot be parsed as
// operators, column filters, or phrases. Each remaining token is quoted so
// AND/OR/NOT/NEAR in the input are terms, not syntax. Tokens are combined
// with implicit AND.
func fts5Query(q string) string {
	var tokens []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() == 0 {
			return
		}
		tok := strings.ReplaceAll(cur.String(), `"`, `""`)
		cur.Reset()
		tokens = append(tokens, `"`+tok+`"`)
	}
	for _, r := range q {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			cur.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return strings.Join(tokens, " ")
}
