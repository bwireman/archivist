package retrieve

import (
	"strings"
	"unicode/utf8"

	"github.com/bwireman/archivist/internal/store"
)

// Snippet returns a short passage from body centered on the earliest query term.
// The window is 240 runes. A match at rune 80 or later starts 80 runes earlier.
func Snippet(body, query string) string {
	const window = 240
	const lead = 80
	flat := strings.Join(strings.Fields(body), " ")
	if flat == "" {
		return ""
	}
	runes := []rune(flat)
	lower := strings.ToLower(flat)
	matchByte := -1
	for _, term := range store.SnippetTerms(query) {
		if i := strings.Index(lower, term); i >= 0 && (matchByte < 0 || i < matchByte) {
			matchByte = i
		}
	}
	match := -1
	if matchByte >= 0 {
		match = utf8.RuneCountInString(flat[:matchByte])
	}
	start := 0
	if match >= lead {
		start = match - lead
	}
	end := start + window
	if end > len(runes) {
		end = len(runes)
	}
	out := string(runes[start:end])
	if start > 0 {
		out = "..." + out
	}
	if end < len(runes) {
		out += "..."
	}
	return out
}
