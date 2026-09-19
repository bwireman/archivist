package check

import (
	"context"
	"strings"

	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/retrieve"
	"github.com/bwireman/archivist/internal/store"
)

const (
	ReasonAppliesTo = "applies_to glob match"
	ReasonSemantic  = "semantic match"
)

type Options struct {
	Description string
	Paths       []string
	Diff        string
	Strict      bool
	TopK        int
}

type Match struct {
	Record   *record.Record
	Reason   string
	Severity record.Severity
}

type Result struct {
	Matches      []Match
	HasViolation bool
}

func Run(ctx context.Context, engine *retrieve.Engine, embedder embed.Embedder, opts Options) (*Result, error) {
	paths := opts.Paths
	if opts.Diff != "" {
		paths = append(paths, pathsFromDiff(opts.Diff)...)
	}
	paths = unique(paths)

	var matches []Match
	seen := map[string]struct{}{}

	for _, st := range []*store.Store{engine.Repo, engine.Home} {
		if st == nil {
			continue
		}
		rules, err := st.RecordsByType(record.TypeRule)
		if err != nil {
			return nil, err
		}
		for _, r := range rules {
			if len(paths) > 0 && r.MatchesPaths(paths) {
				addMatch(&matches, seen, r, ReasonAppliesTo)
			}
		}
	}

	if opts.Description != "" {
		results, err := engine.Search(ctx, embedder, retrieve.Options{
			Query: opts.Description,
			Type:  record.TypeRule,
			TopK:  opts.TopK,
		})
		if err != nil {
			return nil, err
		}
		for _, r := range results {
			addMatch(&matches, seen, r.Record, ReasonSemantic)
		}
	}

	res := &Result{Matches: matches}
	for _, m := range matches {
		if m.Reason == ReasonAppliesTo && m.Record.IsEnforceable() {
			res.HasViolation = true
		}
	}
	return res, nil
}

func addMatch(matches *[]Match, seen map[string]struct{}, r *record.Record, reason string) {
	if _, ok := seen[r.ID]; ok {
		return
	}
	seen[r.ID] = struct{}{}
	*matches = append(*matches, Match{
		Record:   r,
		Reason:   reason,
		Severity: r.Severity,
	})
}

// SplitPathList splits a comma- or newline-separated path list (MCP check.paths).
func SplitPathList(s string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r'
	}) {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func pathsFromDiff(diff string) []string {
	var paths []string
	for _, line := range strings.Split(diff, "\n") {
		line = strings.TrimRight(line, "\r")
		switch {
		case strings.HasPrefix(line, "diff --git "):
			paths = append(paths, pathsFromGitDiffHeader(line)...)
		case strings.HasPrefix(line, "+++ "), strings.HasPrefix(line, "--- "):
			if p := pathFromDiffFileLine(line[4:]); p != "" {
				paths = append(paths, p)
			}
		}
	}
	return unique(paths)
}

func pathFromDiffFileLine(rest string) string {
	rest = strings.TrimSpace(rest)
	if i := strings.IndexByte(rest, '\t'); i >= 0 {
		rest = rest[:i]
	}
	if rest == "/dev/null" || rest == "dev/null" {
		return ""
	}
	if strings.HasPrefix(rest, "a/") || strings.HasPrefix(rest, "b/") {
		return rest[2:]
	}
	return rest
}

func pathsFromGitDiffHeader(line string) []string {
	rest := strings.TrimPrefix(line, "diff --git ")
	a, b, ok := splitGitDiffPaths(rest)
	if !ok {
		return nil
	}
	var out []string
	if a != "" {
		out = append(out, a)
	}
	if b != "" && b != a {
		out = append(out, b)
	}
	return out
}

func splitGitDiffPaths(rest string) (string, string, bool) {
	rest = strings.TrimSpace(rest)
	if strings.HasPrefix(rest, "\"") {
		return splitQuotedGitDiffPaths(rest)
	}
	idx := strings.Index(rest, " b/")
	if idx < 0 {
		return "", "", false
	}
	a := strings.TrimPrefix(rest[:idx], "a/")
	b := strings.TrimPrefix(rest[idx+1:], "b/")
	return a, b, true
}

func splitQuotedGitDiffPaths(rest string) (string, string, bool) {
	first, rest, ok := splitQuoted(rest)
	if !ok {
		return "", "", false
	}
	rest = strings.TrimSpace(rest)
	second, _, ok := splitQuoted(rest)
	if !ok {
		return "", "", false
	}
	return strings.TrimPrefix(first, "a/"), strings.TrimPrefix(second, "b/"), true
}

func splitQuoted(s string) (string, string, bool) {
	if !strings.HasPrefix(s, "\"") {
		return "", s, false
	}
	var b strings.Builder
	escaped := false
	for i := 1; i < len(s); i++ {
		c := s[i]
		if escaped {
			b.WriteByte(c)
			escaped = false
			continue
		}
		if c == '\\' {
			escaped = true
			continue
		}
		if c == '"' {
			return b.String(), s[i+1:], true
		}
		b.WriteByte(c)
	}
	return "", s, false
}

func unique(items []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
