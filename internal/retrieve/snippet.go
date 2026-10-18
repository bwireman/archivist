package retrieve

import (
	"strings"

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
	lower := []rune(strings.ToLower(flat))
	match := -1
	for _, term := range store.SnippetTerms(query) {
		if term == "" {
			continue
		}
		if i := indexRunes(lower, []rune(term)); i >= 0 && (match < 0 || i < match) {
			match = i
		}
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

func indexRunes(hay, needle []rune) int {
	if len(needle) == 0 || len(needle) > len(hay) {
		return -1
	}
	for i := 0; i+len(needle) <= len(hay); i++ {
		if equalRunes(hay[i:i+len(needle)], needle) {
			return i
		}
	}
	return -1
}

func equalRunes(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
