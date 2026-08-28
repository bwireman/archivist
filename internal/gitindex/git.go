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
}

// ListCommits returns recent commits from the repository.
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

// BlameFile returns per-line blame info for a file.
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
