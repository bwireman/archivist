package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/record"
)

func TestLegacyToRecord(t *testing.T) {
	content := "# Use WAL\n\n- Status: Proposed\n\nSQLite should use WAL.\n"
	rec, err := legacyToRecord("/repo", "/repo/docs/decisions/sqlite-wal.md", content, record.ScopeRepo)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Title != "Use WAL" {
		t.Fatalf("title %q", rec.Title)
	}
	if rec.Status != record.StatusProposed {
		t.Fatalf("status %q", rec.Status)
	}
	if rec.Type != record.TypeDecision || rec.Scope != record.ScopeRepo {
		t.Fatalf("type/scope %s/%s", rec.Type, rec.Scope)
	}
	if rec.SourcePath != "docs/decisions/sqlite-wal.md" {
		t.Fatalf("source %q", rec.SourcePath)
	}
	if rec.Slug != "sqlite-wal" {
		t.Fatalf("slug %q", rec.Slug)
	}
}

func TestMigrateDirConvertsLegacyAndSkipsFrontMatter(t *testing.T) {
	dir := t.TempDir()
	legacy := filepath.Join(dir, "001-old.md")
	if err := os.WriteFile(legacy, []byte("# Old ADR\n\n- Status: Accepted\n\nDo the thing.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	typed := filepath.Join(dir, "already.md")
	existing := "---\nid: rec_keep\ntype: decision\nscope: repo\nstatus: accepted\ntitle: Keep\n---\n\nAlready typed.\n"
	if err := os.WriteFile(typed, []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := migrateDir(filepath.Dir(dir), dir, record.ScopeRepo); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(legacy)
	if err != nil {
		t.Fatal(err)
	}
	text := string(got)
	if !strings.HasPrefix(text, "---\nid: rec_") {
		t.Fatalf("legacy not converted: %s", text)
	}
	if !strings.Contains(text, "title: Old ADR") {
		t.Fatalf("missing title: %s", text)
	}

	keep, err := os.ReadFile(typed)
	if err != nil {
		t.Fatal(err)
	}
	if string(keep) != existing {
		t.Fatalf("typed file rewritten: %s", keep)
	}
}
