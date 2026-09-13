package gitindex

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type Commit struct {
	Hash       string
	Subject    string
	Body       string
	Author     string
	AuthoredAt time.Time
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
	cmd := exec.Command("git", "-C", repoRoot, "log",
		fmt.Sprintf("-%d", limit),
		"--pretty=format:%x1e%H%x1f%an%x1f%aI%x1f%s%x1f%b",
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
		fields := strings.SplitN(rec, "\x1f", 5)
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
		commits = append(commits, c)
	}
	return commits
}
