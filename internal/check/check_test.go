package check

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/retrieve"
	"github.com/bwireman/archivist/internal/store"
)

func TestStrictViolationOnlyOnAppliesTo(t *testing.T) {
	st := openStore(t)
	mustRule(t, st, &record.Record{
		ID: "rec_must_store", Slug: "fts-sanitize", Type: record.TypeRule, Scope: record.ScopeRepo,
		Title: "Never pass unsanitized text to FTS5 MATCH", Status: record.StatusAccepted,
		Severity: record.SeverityMust, Body: "Sanitize MATCH input before search.",
		SourcePath: "docs/global-decisions/fts-sanitize.md", AppliesTo: []string{"internal/store/**"},
	})
	mustRule(t, st, &record.Record{
		ID: "rec_should_mcp", Slug: "mcp-stdio", Type: record.TypeRule, Scope: record.ScopeRepo,
		Title: "MCP is stdio only", Status: record.StatusAccepted,
		Severity: record.SeverityShould, Body: "Do not add HTTP MCP.",
		SourcePath: "docs/global-decisions/mcp-stdio.md", AppliesTo: []string{"internal/mcp/**"},
	})

	engine := &retrieve.Engine{Repo: st}
	ctx := context.Background()

	semantic, err := Run(ctx, engine, nil, Options{
		Description: "Sanitize MATCH input before search",
		Paths:       []string{"internal/mcp/server.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if semantic.HasViolation {
		t.Fatal("semantic-only must-rule hit must not set HasViolation")
	}
	if len(semantic.Matches) == 0 {
		t.Fatal("expected semantic match for the FTS rule")
	}

	glob, err := Run(ctx, engine, nil, Options{
		Paths: []string{"internal/store/fts.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !glob.HasViolation {
		t.Fatal("applies_to match of a must rule should set HasViolation")
	}
	if len(glob.Matches) != 1 || glob.Matches[0].Reason != ReasonAppliesTo {
		t.Fatalf("glob matches: %+v", glob.Matches)
	}

	advisory, err := Run(ctx, engine, nil, Options{
		Paths: []string{"internal/mcp/server.go"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if advisory.HasViolation {
		t.Fatal("should-severity glob match must not set HasViolation")
	}
}

func TestPathsFromDiffSkipsDevNullAndReadsGitHeader(t *testing.T) {
	diff := "" +
		"diff --git a/internal/store/old.go b/internal/store/old.go\n" +
		"deleted file mode 100644\n" +
		"--- a/internal/store/old.go\n" +
		"+++ /dev/null\n" +
		"diff --git a/internal/cmd/remember.go b/internal/cmd/remember.go\n" +
		"--- a/internal/cmd/remember.go\n" +
		"+++ b/internal/cmd/remember.go\n" +
		"@@ -1 +1 @@\n" +
		"-old\n" +
		"+new\n" +
		"diff --git a/docs/old.md b/docs/new.md\n" +
		"--- a/docs/old.md\n" +
		"+++ b/docs/new.md\n"

	got := pathsFromDiff(diff)
	want := map[string]bool{
		"internal/store/old.go":    true,
		"internal/cmd/remember.go": true,
		"docs/old.md":              true,
		"docs/new.md":              true,
	}
	if len(got) != len(want) {
		t.Fatalf("paths %v", got)
	}
	for _, p := range got {
		if !want[p] {
			t.Fatalf("unexpected path %q in %v", p, got)
		}
		if p == "/dev/null" || p == "dev/null" {
			t.Fatal("dev/null should not be a path")
		}
	}
}

func TestSplitPathList(t *testing.T) {
	got := SplitPathList("internal/store/fts.go, internal/mcp/server.go\ninternal/check/check.go")
	if len(got) != 3 {
		t.Fatalf("got %v", got)
	}
}

func openStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func mustRule(t *testing.T, st *store.Store, rec *record.Record) {
	t.Helper()
	if err := st.UpsertRecord(rec); err != nil {
		t.Fatal(err)
	}
}
