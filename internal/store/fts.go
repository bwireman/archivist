package store

import (
	"strings"
	"unicode"
)

// fts5Terms turns a natural-language or path-like string into quoted FTS5
// terms. Punctuation is tokenized away so it cannot be parsed as operators,
// column filters, or phrases. Each remaining token is quoted so AND/OR/NOT/NEAR
// in the input are terms, not syntax.
func fts5Terms(q string) []string {
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
	return tokens
}

var ftsStopwords = map[string]bool{
	"a": true, "an": true, "and": true, "are": true, "as": true, "at": true, "be": true,
	"before": true, "after": true, "by": true, "for": true, "from": true, "how": true,
	"in": true, "into": true, "is": true, "it": true, "its": true, "of": true, "on": true,
	"or": true, "that": true, "the": true, "this": true, "to": true, "was": true,
	"what": true, "when": true, "where": true, "which": true, "who": true, "why": true,
	"will": true, "with": true, "without": true, "do": true, "does": true, "not": true,
	"no": true, "we": true, "our": true, "you": true, "your": true, "i": true,
}

// searchTerms splits a code-map query into lowercase words. Stopwords are
// dropped. When every word is a stopword, or the query has no words, the
// trimmed query is one term so a search for "%" or "_" stays literal.
func searchTerms(query string) []string {
	raw := strings.TrimSpace(query)
	if raw == "" {
		return nil
	}
	quoted := contentTerms(fts5Terms(raw))
	if len(quoted) == 0 {
		return []string{raw}
	}
	out := make([]string, len(quoted))
	for i, t := range quoted {
		out[i] = strings.ToLower(strings.Trim(t, `"`))
	}
	return out
}

// SnippetTerms returns the unquoted lowercase query words used to center a
// search snippet. It does not stem them and it does not change SearchFTS.
func SnippetTerms(query string) []string {
	quoted := contentTerms(fts5Terms(query))
	out := make([]string, 0, len(quoted))
	for _, t := range quoted {
		out = append(out, strings.ToLower(strings.Trim(t, `"`)))
	}
	return out
}

func contentTerms(terms []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range terms {
		word := strings.ToLower(strings.Trim(t, `"`))
		if ftsStopwords[word] || seen[word] {
			continue
		}
		seen[word] = true
		out = append(out, t)
	}
	return out
}

// minShouldMatch is how many of n content terms a record must contain to count
// as a keyword hit after the all-terms match came back empty: two thirds,
// and at least two when the query has two or more.
func minShouldMatch(n int) int {
	if n <= 2 {
		return n
	}
	return (2*n + 2) / 3
}
