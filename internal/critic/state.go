package critic

import "strings"

// State is the critic input: title, a blank line, and body, cut to 2000 bytes.
func State(title, body string) string {
	s := strings.TrimSpace(title) + "\n\n" + strings.TrimSpace(body)
	s = strings.TrimSpace(s)
	if len(s) > 2000 {
		s = s[:2000]
	}
	return s
}
