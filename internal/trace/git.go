package trace

import (
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/gitindex"
)

const (
	noteTime        = "commits at or after this time, plus uncommitted tracked changes versus HEAD; untracked files are omitted; --until does not narrow paths"
	noteRev         = "work tree versus this revision, including staged and unstaged tracked changes; untracked files are omitted; --until does not narrow paths"
	noteNotAncestor = "; revision is not an ancestor of HEAD, so these are not the commits on the current branch since that revision"
)

// GitJoin is the path set and lower time cutoff for a --since trace.
// Note qualifies how Paths were chosen. An upper log bound is not applied here.
type GitJoin struct {
	Paths   []string
	After   time.Time
	Skipped string
	Note    string
}

// GitChanged resolves --since to changed paths and a lower log cutoff.
// An empty since returns a zero join. A directory that is not a git repo
// sets Skipped and does not fail. A rev that git rejects is an error.
// A revision range is an error: the log end is an RFC3339 --until, not a second rev.
// Paths stay a lower-bound diff, including uncommitted tracked changes.
func GitChanged(repoRoot, since string) (GitJoin, error) {
	since = strings.TrimSpace(since)
	if since == "" {
		return GitJoin{}, nil
	}
	if strings.HasPrefix(since, "-") {
		return GitJoin{}, fmt.Errorf("since %q must be a rev or an RFC3339 time", since)
	}
	if strings.Contains(since, "..") {
		return GitJoin{}, fmt.Errorf("since %q is a revision range; pass one rev and set the log end with --until (RFC3339)", since)
	}
	if !gitindex.IsGitRepo(repoRoot) {
		return GitJoin{Skipped: "not a git repository"}, nil
	}
	if t, ok := parseSinceTime(since); ok {
		paths, err := pathsSince(repoRoot, t)
		if err != nil {
			return GitJoin{}, err
		}
		return GitJoin{Paths: paths, After: t, Note: noteTime}, nil
	}
	after, err := commitTime(repoRoot, since)
	if err != nil {
		return GitJoin{}, err
	}
	paths, err := diffNames(repoRoot, since)
	if err != nil {
		return GitJoin{}, err
	}
	ancestor, err := ancestorOfHEAD(repoRoot, since)
	if err != nil {
		return GitJoin{}, err
	}
	note := noteRev
	if !ancestor {
		note += noteNotAncestor
	}
	return GitJoin{Paths: paths, After: after, Note: note}, nil
}

func ancestorOfHEAD(repoRoot, rev string) (bool, error) {
	cmd := exec.Command("git", "-C", repoRoot, "merge-base", "--is-ancestor", rev, "HEAD")
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	if exit, ok := errors.AsType[*exec.ExitError](err); ok && exit.ExitCode() == 1 {
		return false, nil
	}
	return false, fmt.Errorf("git merge-base --is-ancestor %s HEAD: %w", rev, err)
}

func parseSinceTime(s string) (time.Time, bool) {
	return parseTime(s)
}

func pathsSince(repoRoot string, t time.Time) ([]string, error) {
	logged, err := gitLines(repoRoot, "log", "--since="+t.Format(time.RFC3339), "--name-only", "--pretty=format:")
	if err != nil {
		return nil, err
	}
	dirty, err := gitLines(repoRoot, "diff", "--name-only", "HEAD")
	if err != nil {
		return nil, err
	}
	return uniqueSorted(append(logged, dirty...)), nil
}

func diffNames(repoRoot, rev string) ([]string, error) {
	lines, err := gitLines(repoRoot, "diff", "--name-only", rev)
	if err != nil {
		return nil, err
	}
	return uniqueSorted(lines), nil
}

func commitTime(repoRoot, rev string) (time.Time, error) {
	out, err := gitOutput(repoRoot, "show", "-s", "--format=%cI", rev)
	if err != nil {
		return time.Time{}, err
	}
	raw := strings.TrimSpace(out)
	t, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("commit time %q: %w", raw, err)
	}
	return t, nil
}

func gitLines(repoRoot string, args ...string) ([]string, error) {
	out, err := gitOutput(repoRoot, args...)
	if err != nil {
		return nil, err
	}
	var lines []string
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func gitOutput(repoRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		if exit, ok := errors.AsType[*exec.ExitError](err); ok {
			msg := strings.TrimSpace(string(exit.Stderr))
			if msg == "" {
				msg = err.Error()
			}
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
		}
		return "", err
	}
	return string(out), nil
}

func uniqueSorted(paths []string) []string {
	return slices.Compact(slices.Sorted(slices.Values(paths)))
}
