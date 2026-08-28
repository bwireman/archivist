# Archivist context dump

Retrieved chunks from a local code index. Use only facts present here.

- Query: git log blame commits
- Chunks: 20

## `internal/gitindex/git_test.go` (code, lines 46-76, score 0.638)

Blame: bwireman (b98ffc6e8be04621393bec1b26f10a649a2e8eed)

```go
File: internal/gitindex/git_test.go

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
```


## `internal/gitindex/git.go` (code, lines 34-53, score 0.631)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/gitindex/git.go

package gitindex

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode"

	"github.com/bwireman/archivist/internal/chunk"
)

type Commit struct {
	Hash       string
	Subject    string
	Body       string
	Author     string
	AuthoredAt time.Time
	Files      []string
}

type BlameInfo struct {
	Author string
	Commit string
}

// IsGitRepo returns true if path is inside a git repository.
func IsGitRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	return cmd.Run() == nil

func ListCommits(repoRoot string, limit int) ([]Commit, error) {
	if !IsGitRepo(repoRoot) {
		return nil, nil
	}
	if limit <= 0 {
		limit = 500
	}
	// Record starts with RS (\x1e). Fields are US (\x1f) separated. GS (\x1d)
	// ends the header so a multiline body is not mixed with --name-only files.
	cmd := exec.Command("git", "-C", repoRoot, "log",
		fmt.Sprintf("-%d", limit),
		"--pretty=format:%x1e%H%x1f%an%x1f%aI%x1f%s%x1f%b%x1d",
		"--name-only",
	)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git log: %w", err)
	}
	return parseGitLog(string(out)), nil
}
```


## `internal/gitindex/git.go` (code, lines 55-91, score 0.611)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/gitindex/git.go

package gitindex

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode"

	"github.com/bwireman/archivist/internal/chunk"
)

type Commit struct {
	Hash       string
	Subject    string
	Body       string
	Author     string
	AuthoredAt time.Time
	Files      []string
}

type BlameInfo struct {
	Author string
	Commit string
}

// IsGitRepo returns true if path is inside a git repository.
func IsGitRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	return cmd.Run() == nil

func parseGitLog(raw string) []Commit {
	records := strings.Split(raw, "\x1e")
	var commits []Commit
	for _, rec := range records {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		header, filesPart, ok := strings.Cut(rec, "\x1d")
		if !ok {
			header = rec
			filesPart = ""
		}
		fields := strings.SplitN(header, "\x1f", 5)
		if len(fields) < 4 {
			continue
		}
		authoredAt, _ := time.Parse(time.RFC3339, fields[2])
		c := Commit{
			Hash:       fields[0],
			Author:     fields[1],
			AuthoredAt: authoredAt,
			Subject:    fields[3],
		}
		if len(fields) > 4 {
			c.Body = strings.TrimSpace(fields[4])
		}
		for _, line := range strings.Split(filesPart, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				c.Files = append(c.Files, line)
			}
		}
		commits = append(commits, c)
	}
	return commits
}
```


## `internal/gitindex/git.go` (code, lines 124-146, score 0.606)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/gitindex/git.go

package gitindex

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode"

	"github.com/bwireman/archivist/internal/chunk"
)

type Commit struct {
	Hash       string
	Subject    string
	Body       string
	Author     string
	AuthoredAt time.Time
	Files      []string
}

type BlameInfo struct {
	Author string
	Commit string
}

// IsGitRepo returns true if path is inside a git repository.
func IsGitRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	return cmd.Run() == nil

func parseBlame(raw string) map[int]BlameInfo {
	result := make(map[int]BlameInfo)
	lines := strings.Split(raw, "\n")
	lineNum := 0
	var cur BlameInfo
	for _, line := range lines {
		if strings.HasPrefix(line, "\t") {
			lineNum++
			result[lineNum] = cur
			continue
		}
		if sha, ok := parseBlameSHA(line); ok {
			if cur.Commit != sha {
				cur = BlameInfo{Commit: sha}
			}
			continue
		}
		if strings.HasPrefix(line, "author ") {
			cur.Author = strings.TrimPrefix(line, "author ")
		}
	}
	return result
}
```


## `internal/gitindex/git.go` (code, lines 112-122, score 0.592)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/gitindex/git.go

package gitindex

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode"

	"github.com/bwireman/archivist/internal/chunk"
)

type Commit struct {
	Hash       string
	Subject    string
	Body       string
	Author     string
	AuthoredAt time.Time
	Files      []string
}

type BlameInfo struct {
	Author string
	Commit string
}

// IsGitRepo returns true if path is inside a git repository.
func IsGitRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	return cmd.Run() == nil

func BlameFile(repoRoot, relPath string) (map[int]BlameInfo, error) {
	if !IsGitRepo(repoRoot) {
		return nil, nil
	}
	cmd := exec.Command("git", "-C", repoRoot, "blame", "--line-porcelain", relPath)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return parseBlame(string(out)), nil
}
```


## `internal/gitindex/git.go` (code, lines 148-164, score 0.591)

Blame: bwireman (b98ffc6e8be04621393bec1b26f10a649a2e8eed)

```go
File: internal/gitindex/git.go

package gitindex

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode"

	"github.com/bwireman/archivist/internal/chunk"
)

type Commit struct {
	Hash       string
	Subject    string
	Body       string
	Author     string
	AuthoredAt time.Time
	Files      []string
}

type BlameInfo struct {
	Author string
	Commit string
}

// IsGitRepo returns true if path is inside a git repository.
func IsGitRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	return cmd.Run() == nil

func parseBlameSHA(line string) (string, bool) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return "", false
	}
	sha := fields[0]
	n := len(sha)
	if n != 40 && n != 64 {
		return "", false
	}
	for _, r := range sha {
		if !unicode.Is(unicode.ASCII_Hex_Digit, r) {
			return "", false
		}
	}
	return sha, true
}
```


## `internal/gitindex/git.go` (code, lines 22-25, score 0.590)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/gitindex/git.go

package gitindex

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode"

	"github.com/bwireman/archivist/internal/chunk"
)

type Commit struct {
	Hash       string
	Subject    string
	Body       string
	Author     string
	AuthoredAt time.Time
	Files      []string
}

type BlameInfo struct {
	Author string
	Commit string
}

// IsGitRepo returns true if path is inside a git repository.
func IsGitRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	return cmd.Run() == nil

type BlameInfo struct {
	Author string
	Commit string
}
```


## `internal/gitindex/git_test.go` (code, lines 78-86, score 0.586)

Blame: bwireman (b98ffc6e8be04621393bec1b26f10a649a2e8eed)

```go
File: internal/gitindex/git_test.go

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

func TestParseBlameSHA(t *testing.T) {
	sha, ok := parseBlameSHA("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa 1 1 3")
	if !ok || sha != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("sha=%q ok=%v", sha, ok)
	}
	if _, ok := parseBlameSHA("author Ada"); ok {
		t.Fatal("author line should not parse as SHA")
	}
}
```


## `internal/gitindex/git.go` (code, lines 93-109, score 0.582)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/gitindex/git.go

package gitindex

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode"

	"github.com/bwireman/archivist/internal/chunk"
)

type Commit struct {
	Hash       string
	Subject    string
	Body       string
	Author     string
	AuthoredAt time.Time
	Files      []string
}

type BlameInfo struct {
	Author string
	Commit string
}

// IsGitRepo returns true if path is inside a git repository.
func IsGitRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	return cmd.Run() == nil

func CommitChunks(c Commit) []chunk.Chunk {
	var b strings.Builder
	fmt.Fprintf(&b, "Commit: %s\nAuthor: %s\nDate: %s\nSubject: %s\n", c.Hash, c.Author, c.AuthoredAt.Format(time.RFC3339), c.Subject)
	if c.Body != "" {
		fmt.Fprintf(&b, "\n%s\n", c.Body)
	}
	if len(c.Files) > 0 {
		fmt.Fprintf(&b, "\nFiles:\n%s\n", strings.Join(c.Files, "\n"))
	}
	content := b.String()
	return []chunk.Chunk{
		chunk.NewChunk(c.Hash, chunk.TypeCommit, 0, 0, content, map[string]string{
			"author": c.Author,
			"hash":   c.Hash,
		}),
	}
}
```


## `internal/gitindex/git.go` (code, lines 13-20, score 0.580)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/gitindex/git.go

package gitindex

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode"

	"github.com/bwireman/archivist/internal/chunk"
)

type Commit struct {
	Hash       string
	Subject    string
	Body       string
	Author     string
	AuthoredAt time.Time
	Files      []string
}

type BlameInfo struct {
	Author string
	Commit string
}

// IsGitRepo returns true if path is inside a git repository.
func IsGitRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	return cmd.Run() == nil

type Commit struct {
	Hash       string
	Subject    string
	Body       string
	Author     string
	AuthoredAt time.Time
	Files      []string
}
```


## `internal/gitindex/git_test.go` (code, lines 88-102, score 0.569)

Blame: bwireman (b98ffc6e8be04621393bec1b26f10a649a2e8eed)

```go
File: internal/gitindex/git_test.go

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
```


## `internal/gitindex/git.go` (code, lines 28-31, score 0.559)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/gitindex/git.go

package gitindex

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode"

	"github.com/bwireman/archivist/internal/chunk"
)

type Commit struct {
	Hash       string
	Subject    string
	Body       string
	Author     string
	AuthoredAt time.Time
	Files      []string
}

type BlameInfo struct {
	Author string
	Commit string
}

// IsGitRepo returns true if path is inside a git repository.
func IsGitRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	return cmd.Run() == nil

func IsGitRepo(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--git-dir")
	return cmd.Run() == nil
}
```


## `internal/gitindex/git_test.go` (code, lines 9-44, score 0.545)

Blame: bwireman (b98ffc6e8be04621393bec1b26f10a649a2e8eed)

```go
File: internal/gitindex/git_test.go

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
```


## `b98ffc6e8be04621393bec1b26f10a649a2e8eed` (commit, no line range, score 0.486)

```
Commit: b98ffc6e8be04621393bec1b26f10a649a2e8eed
Author: bwireman
Date: 2026-08-28T18:35:13-05:00
Subject: Fix git parsing, incremental index, and path matching.

Git log dropped every commit after the first, blame never stored SHAs, and a failed embed could leave a file marked current with no chunks.

Files:
.gitignore
internal/chunk/chunk.go
internal/chunk/chunk_test.go
internal/chunk/generic.go
internal/chunk/treesitter.go
internal/cmd/root.go
internal/embed/ollama.go
internal/gitindex/git.go
internal/gitindex/git_test.go
internal/index/indexer.go
internal/index/indexer_test.go
internal/search/search.go
internal/search/search_test.go
internal/store/store.go
internal/store/store_test.go
```


## `.cursor/rules/archivist-dump.mdc` (doc, lines 1-5, score 0.441)

```
---
description: Dump indexed decisions into docs/dump for later LLM sessions
alwaysApply: true
---
```


## `.cursor/rules/archivist-consult.mdc` (doc, lines 1-5, score 0.420)

```
---
description: Consult Archivist, then docs, before doing work
alwaysApply: true
---
```


## `.cursor/rules/archivist-index.mdc` (doc, lines 1-5, score 0.415)

```
---
description: Regularly re-index the repository with Archivist
alwaysApply: true
---
```


## `docs/decisions/001-use-qwen3-embedding.md` (adr, lines 1-5, score 0.412)

```md
# Use qwen3-embedding:0.6b for this repository

- Status: accepted
- Date: 2026-08-28
```


## `.cursor/rules/archivist-consult.mdc` (doc, lines 6-23, score 0.405)

````
# Consult before working

Before implementing, researching, or changing architecture, look things up in this order. Do not guess APIs, defaults, or past decisions.

1. **Archivist** (needs Ollama). Search, then dump if you need full chunks:

```bash
./archivist search "how indexing skips dump files"
./archivist dump "how indexing skips dump files"
./archivist search --type adr "sqlite"
```

2. **Docs** next, using what the index pointed at:
   - `docs/decisions/` — source ADRs (authoritative)
   - `docs/dump/` — generated context, start with `docs/dump/decisions.md`

If Ollama is down, skip step 1 and read `docs/` directly. If both are empty or silent, read the code. Skip this lookup for typos, formatting, and one-line obvious fixes.
````


## `go.sum` (code, lines 22-43, score 0.401)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```
github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec/go.mod h1:qqbHyh8v60DhA7CoWK5oRCqLrMHRGoxYCSS9EjAz6Eo=
github.com/russross/blackfriday/v2 v2.1.0/go.mod h1:+Rmxgy9KzJVeS9/2gXHxylqXiyQDYRxCVz55jmeOWTM=
github.com/smacker/go-tree-sitter v0.0.0-20240827094217-dd81d9e9be82 h1:6C8qej6f1bStuePVkLSFxoU22XBS165D3klxlzRg8F4=
github.com/smacker/go-tree-sitter v0.0.0-20240827094217-dd81d9e9be82/go.mod h1:xe4pgH49k4SsmkQq5OT8abwhWmnzkhpgnXeekbx2efw=
github.com/spf13/cobra v1.10.2 h1:DMTTonx5m65Ic0GOoRY2c16WCbHxOOw6xxezuLaBpcU=
github.com/spf13/cobra v1.10.2/go.mod h1:7C1pvHqHw5A4vrJfjNwvOdzYu0Gml16OCs2GRiTUUS4=
github.com/spf13/pflag v1.0.9 h1:9exaQaMOCwffKiiiYk6/BndUBv+iRViNW+4lEMi0PvY=
github.com/spf13/pflag v1.0.9/go.mod h1:McXfInJRrz4CZXVZOBLb0bTZqETkiAhM9Iw0y3An2Bg=
github.com/stretchr/testify v1.9.0 h1:HtqpIVDClZ4nwg75+f6Lvsy/wHu+3BoSGCbBAcpTsTg=
github.com/stretchr/testify v1.9.0/go.mod h1:r2ic/lqez/lEtzL7wO/rwa5dbSLXVDPFyf8C91i36aY=
go.yaml.in/yaml/v3 v3.0.4/go.mod h1:DhzuOOF2ATzADvBadXxruRBLzYTpT36CKvDb3+aBEFg=
golang.org/x/mod v0.37.0 h1:vF1DjpVEshcIqoEaauuHebaLk1O1forxjxBaVn884JQ=
golang.org/x/mod v0.37.0/go.mod h1:m8S8VeM9r4dzDwjrKO0a1sZP3YjeMamRRlD+fmR2Q/0=
golang.org/x/sync v0.21.0 h1:HLII4xRRTtCRkxYp4HNFF0Js/Og6q2i++KXbg0gHCwM=
golang.org/x/sync v0.21.0/go.mod h1:9xrNwdLfx4jkKbNva9FpL6vEN7evnE43NNNJQ2LF3+0=
golang.org/x/sys v0.47.0 h1:o7XGOvZQCADBQQ4Y7VNq2dRWQR7JmOUW8Kxx4ZsNgWs=
golang.org/x/sys v0.47.0/go.mod h1:4GL1E5IUh+htKOUEOaiffhrAeqysfVGipDYzABqnCmw=
golang.org/x/tools v0.47.0 h1:7Kn5x/d1svx/PzryTsqeoZN4TZwqeH5pGWjefhLi/1Q=
golang.org/x/tools v0.47.0/go.mod h1:dFHnyTvFWY212G+h7ZY4Vsp/K3U4/7W9TyVaAul8uCA=
gopkg.in/check.v1 v0.0.0-20161208181325-20d25e280405/go.mod h1:Co6ibVJAznAaIkqp8huTwlJQCZ016jof/cbN4VW5Yz0=
gopkg.in/yaml.v3 v3.0.1 h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=
gopkg.in/yaml.v3 v3.0.1/go.mod h1:K4uyk7z7BCEPqu6E+C64Yfv1cQ7kz7rIZviUmN+EgEM=
modernc.org/cc/v4 v4.29.1 h1:MKgdCV3WykTSPqpVrnxdEDS0HEd2FHpKZDzxzU5LyeI=
```

