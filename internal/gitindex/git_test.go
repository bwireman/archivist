package gitindex

import (
	"strings"
	"testing"
	"time"
)

func TestParseGitLogMultipleCommitsAndFiles(t *testing.T) {
	raw := "\x1e" +
		"aaa111aaa111aaa111aaa111aaa111aaa111aaa1\x1fAda\x1f2026-08-28T12:00:00Z\x1fadd indexer\x1f\x1d\n" +
		"internal/index/indexer.go\n" +
		"Makefile\n" +
		"\n" +
		"\x1e" +
		"bbb222bbb222bbb222bbb222bbb222bbb222bbb2\x1fBob\x1f2026-08-27T12:00:00Z\x1ffix dump\x1fexplain the dump flag\n" +
		"second body line\x1d\n" +
		"internal/dump/dump.go\n"

	commits := parseGitLog(raw)
	if len(commits) != 2 {
		t.Fatalf("got %d commits, want 2", len(commits))
	}

	if commits[0].Hash[:6] != "aaa111" || commits[0].Author != "Ada" || commits[0].Subject != "add indexer" {
		t.Fatalf("commit 0: %#v", commits[0])
	}
	if got := strings.Join(commits[0].Files, ","); got != "internal/index/indexer.go,Makefile" {
		t.Fatalf("commit 0 files: %v", commits[0].Files)
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
	if len(commits[1].Files) != 1 || commits[1].Files[0] != "internal/dump/dump.go" {
		t.Fatalf("commit 1 files: %v", commits[1].Files)
	}
}

func TestParseBlameExtractsCommitAndAuthor(t *testing.T) {
	raw := strings.Join([]string{
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa 1 1 2",
		"author Ada",
		"author-mail <ada@example.com>",
		"filename foo.go",
		"\tpackage foo",
		"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa 2 2",
		"filename foo.go",
		"\tfunc Bar() {}",
		"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb 3 3 1",
		"author Bob",
		"filename foo.go",
		"\tfunc Baz() {}",
		"",
	}, "\n")

	got := parseBlame(raw)
	if len(got) != 3 {
		t.Fatalf("got %d lines, want 3: %#v", len(got), got)
	}
	if got[1].Author != "Ada" || got[1].Commit != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("line 1: %#v", got[1])
	}
	if got[2].Author != "Ada" || got[2].Commit != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("line 2 should keep author from the same commit: %#v", got[2])
	}
	if got[3].Author != "Bob" || got[3].Commit != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" {
		t.Fatalf("line 3: %#v", got[3])
	}
}

func TestParseBlameSHA(t *testing.T) {
	sha, ok := parseBlameSHA("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa 1 1 3")
	if !ok || sha != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("sha=%q ok=%v", sha, ok)
	}
	if _, ok := parseBlameSHA("author Ada"); ok {
		t.Fatal("author line should not parse as SHA")
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
	if len(commits[0].Files) == 0 {
		t.Fatalf("expected files on latest commit, got %#v", commits[0])
	}
}
