package gitindex

import (
	"testing"
	"time"
)

func TestParseGitLogMultipleCommits(t *testing.T) {
	raw := "\x1e" +
		"aaa111aaa111aaa111aaa111aaa111aaa111aaa1\x1fAda\x1f2026-08-28T12:00:00Z\x1fadd indexer\x1f" +
		"\x1e" +
		"bbb222bbb222bbb222bbb222bbb222bbb222bbb2\x1fBob\x1f2026-08-27T12:00:00Z\x1ffix dump\x1fexplain the dump flag\n" +
		"second body line"

	commits := parseGitLog(raw)
	if len(commits) != 2 {
		t.Fatalf("got %d commits, want 2", len(commits))
	}

	if commits[0].Hash[:6] != "aaa111" || commits[0].Author != "Ada" || commits[0].Subject != "add indexer" {
		t.Fatalf("commit 0: %#v", commits[0])
	}
	if commits[0].Body != "" {
		t.Fatalf("commit 0 body: %q", commits[0].Body)
	}
	if !commits[0].AuthoredAt.Equal(time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)) {
		t.Fatalf("commit 0 time: %s", commits[0].AuthoredAt)
	}

	if commits[1].Subject != "fix dump" {
		t.Fatalf("commit 1 subject: %q", commits[1].Subject)
	}
	if commits[1].Body != "explain the dump flag\nsecond body line" {
		t.Fatalf("commit 1 body: %q", commits[1].Body)
	}
}

func TestListCommitsThisRepo(t *testing.T) {
	commits, err := ListCommits(".", 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) == 0 {
		t.Fatal("expected commits from this repo")
	}
	if commits[0].Hash == "" || commits[0].Subject == "" {
		t.Fatalf("incomplete commit: %#v", commits[0])
	}
}
