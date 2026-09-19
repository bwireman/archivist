package version

import "testing"

func TestRevisionPrefersLdflagsCommit(t *testing.T) {
	orig := Commit
	t.Cleanup(func() { Commit = orig })
	Commit = "abc123"
	if got := Revision(); got != "abc123" {
		t.Fatalf("Revision() = %q", got)
	}
}
