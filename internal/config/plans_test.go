package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewPlanLivesUnderHomeCheckout(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	body := "Status: done\n\n# Moved\n"
	dst, err := HomePlanPath(root, "new-plan")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil || string(got) != body {
		t.Fatalf("home plan %q err=%v", got, err)
	}
	if _, err := os.Stat(LegacyPlanPath(root, "new-plan")); !os.IsNotExist(err) {
		t.Fatal("plan body was also written under the checkout")
	}

	legacy := filepath.Join(root, ".archivist", "plans")
	if err := os.MkdirAll(legacy, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "finished.md"), []byte("Status: done\n\n# Finished\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "open.md"), []byte("Status: in progress\n\n# Open\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := MoveSettledPlans(root); err != nil {
		t.Fatal(err)
	}
	moved, err := os.ReadFile(mustHomePlan(t, root, "finished"))
	if err != nil || !strings.Contains(string(moved), "# Finished") {
		t.Fatalf("moved %q err=%v", moved, err)
	}
	if _, err := os.Stat(filepath.Join(legacy, "finished.md")); !os.IsNotExist(err) {
		t.Fatal("finished plan stayed in the checkout")
	}
	if _, err := os.Stat(filepath.Join(legacy, "open.md")); err != nil {
		t.Fatal(err)
	}
}

func mustHomePlan(t *testing.T, root, slug string) string {
	t.Helper()
	p, err := HomePlanPath(root, slug)
	if err != nil {
		t.Fatal(err)
	}
	return p
}
