package check

import (
	"context"
	"regexp"
	"strings"

	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/retrieve"
	"github.com/bwireman/archivist/internal/store"
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
				addMatch(&matches, seen, r, "applies_to glob match")
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
			addMatch(&matches, seen, r.Record, "semantic match")
		}
	}

	res := &Result{Matches: matches}
	for _, m := range matches {
		if m.Record.IsEnforceable() {
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

var diffPathRe = regexp.MustCompile(`^\+\+\+ [ab]/(.*)$|^\+\+\+ (.*)$|^diff --git a/(.*) b/`)

func pathsFromDiff(diff string) []string {
	var paths []string
	for _, line := range strings.Split(diff, "\n") {
		if m := diffPathRe.FindStringSubmatch(line); len(m) > 1 {
			p := m[1]
			if p == "" && len(m) > 2 {
				p = m[2]
			}
			p = strings.TrimSpace(p)
			if p != "" && p != "dev/null" {
				paths = append(paths, p)
			}
		}
	}
	return paths
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
