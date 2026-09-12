package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallCursor(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root)
	if err := Install(root, TargetCursor); err != nil {
		t.Fatal(err)
	}
	rule := filepath.Join(root, ".cursor", "rules", "archivist-consult.mdc")
	data, err := os.ReadFile(rule)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "alwaysApply: true") {
		t.Fatalf("rule missing alwaysApply: %s", data)
	}
	skill := filepath.Join(root, ".cursor", "skills", "record-decision", "SKILL.md")
	if _, err := os.Stat(skill); err != nil {
		t.Fatal(err)
	}
}

func TestInstallAgentsMDWritesRulesOnly(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root)
	if err := Install(root, TargetAgentsMD); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Consult the archive") {
		t.Fatalf("AGENTS.md missing rules: %s", data)
	}
	if _, err := os.Stat(filepath.Join(root, ".cursor", "skills")); !os.IsNotExist(err) {
		t.Fatal("agents-md should not install cursor skills")
	}
}

func writeTree(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(root, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "rules", "consult.md"), []byte("# Consult the archive\n\nLook it up.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	skillDir := filepath.Join(root, "skills", "record-decision")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	skill := "---\nname: record-decision\ndescription: Record a decision.\n---\n\n# Record\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skill), 0o644); err != nil {
		t.Fatal(err)
	}
}
