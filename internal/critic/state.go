package critic

import (
	"strings"
	"unicode/utf8"
)

// State is the critic input: title, a blank line, and body, cut to 2000 bytes
// on a rune boundary.
func State(title, body string) string {
	s := strings.TrimSpace(title) + "\n\n" + strings.TrimSpace(body)
	s = strings.TrimSpace(s)
	if len(s) <= 2000 {
		return s
	}
	s = s[:2000]
	for s != "" && !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}
