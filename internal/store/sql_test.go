package store

import (
	"strings"
	"testing"
)

func TestPlaceholders(t *testing.T) {
	if got := placeholders(0); got != "" {
		t.Fatalf("placeholders(0)=%q", got)
	}
	if got := placeholders(1); got != "?" {
		t.Fatalf("placeholders(1)=%q", got)
	}
	if got := placeholders(3); got != "?,?,?" {
		t.Fatalf("placeholders(3)=%q", got)
	}
}

func TestSqliteDSN(t *testing.T) {
	got, err := sqliteDSN("/tmp/archive.db")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "_pragma") {
		t.Fatalf("dsn uses raw _pragma: %s", got)
	}
	for _, key := range []string{"_busy_timeout=5000", "_foreign_keys=on", "_journal_mode=WAL"} {
		if !strings.Contains(got, key) {
			t.Fatalf("dsn missing %s: %s", key, got)
		}
	}
	if !strings.HasPrefix(got, "/tmp/archive.db?") {
		t.Fatalf("dsn path prefix: %s", got)
	}

	if _, err := sqliteDSN("/tmp/foo?bar.db"); err == nil {
		t.Fatal("expected error for ? in path")
	}
	if _, err := sqliteDSN("/tmp/foo#bar.db"); err == nil {
		t.Fatal("expected error for # in path")
	}
}
