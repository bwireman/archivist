package skills

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallUserKeepsOtherMCPServers(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	mcpPath := filepath.Join(home, ".cursor", "mcp.json")
	original := []byte("{\n  \"mcpServers\": {\"vivid\": {\"url\": \"https://example.com/mcp\"}},\n  \"extra\": true\n}\n")
	if err := os.MkdirAll(filepath.Dir(mcpPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mcpPath, original, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := InstallUser(""); err != nil {
		t.Fatal(err)
	}
	if err := InstallUser(""); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(mcpPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "https://example.com/mcp") || !strings.Contains(string(got), "\"extra\"") {
		t.Fatalf("mcp.json %s", got)
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(got, &cfg); err != nil {
		t.Fatal(err)
	}
	var servers map[string]json.RawMessage
	if err := json.Unmarshal(cfg["mcpServers"], &servers); err != nil {
		t.Fatal(err)
	}
	if _, ok := servers["vivid"]; !ok {
		t.Fatalf("servers %s", cfg["mcpServers"])
	}
	if _, ok := servers["archivist"]; !ok {
		t.Fatal("missing archivist")
	}
	if len(servers) != 2 {
		t.Fatalf("archivist duplicated: %s", cfg["mcpServers"])
	}
	if _, err := os.Stat(filepath.Join(home, ".archivist", "install.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".cursor", "rules")); !os.IsNotExist(err) {
		t.Fatal("user install wrote rules")
	}
	skill := filepath.Join(home, ".agents", "skills", "plan-changes", "SKILL.md")
	if _, err := os.Stat(skill); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".cursor", "skills")); !os.IsNotExist(err) {
		t.Fatal("user install wrote project skills")
	}

	bad := filepath.Join(home, ".cursor", "mcp.json")
	before := []byte("{not json // https://example.com")
	if err := os.WriteFile(bad, before, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := InstallUser(""); err == nil {
		t.Fatal("invalid mcp.json should fail")
	}
	after, err := os.ReadFile(bad)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("invalid file changed: %s", after)
	}
}
