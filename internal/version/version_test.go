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

func TestBuiltPrefersLdflagsDate(t *testing.T) {
	orig := Date
	t.Cleanup(func() { Date = orig })
	Date = "2026-10-11T19:17:00Z"
	if got := Built(); got != "2026-10-11T19:17:00Z" {
		t.Fatalf("Built() = %q", got)
	}
}

func TestFormatOmitsUnknownHashAndDate(t *testing.T) {
	if got := format("0.1.0", "", ""); got != "0.1.0" {
		t.Fatalf("format empty = %q", got)
	}
	if got := format("0.1.0", "abc", ""); got != "0.1.0 abc" {
		t.Fatalf("format hash = %q", got)
	}
	if got := format("0.1.0", "", "2026-10-11T19:17:00Z"); got != "0.1.0 2026-10-11T19:17:00Z" {
		t.Fatalf("format date = %q", got)
	}
	if got := format("0.1.0", "abc", "2026-10-11T19:17:00Z"); got != "0.1.0 abc 2026-10-11T19:17:00Z" {
		t.Fatalf("format both = %q", got)
	}
}

func TestReportUsesStampedHashAndDate(t *testing.T) {
	origCommit, origDate := Commit, Date
	t.Cleanup(func() {
		Commit = origCommit
		Date = origDate
	})
	Commit = "abc123"
	Date = "2026-10-11T19:17:00Z"
	want := Version + " abc123 2026-10-11T19:17:00Z"
	if got := Report(); got != want {
		t.Fatalf("Report() = %q, want %q", got, want)
	}
}
