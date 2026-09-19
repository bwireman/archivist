package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/version"
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
	if sha256Hex(data) != sha256Hex([]byte(renderRule(TargetCursor, "consult.md", fileBytes(t, filepath.Join(root, "rules", "consult.md"))))) {
		t.Fatal("on-disk consult.mdc hash should match cursor render")
	}
	skill := filepath.Join(root, ".cursor", "skills", "record-decision", "SKILL.md")
	if _, err := os.Stat(skill); err != nil {
		t.Fatal(err)
	}
	assertManifest(t, root, TargetCursor)
	consult := fileBytes(t, filepath.Join(root, "rules", "consult.md"))
	wantCursor := sha256Hex([]byte(renderRule(TargetCursor, "consult.md", consult)))
	wantPlain := sha256Hex([]byte(renderRule(TargetClaude, "consult.md", consult)))
	m := readManifest(t, root)
	if m.Rules["consult.md"][string(TargetCursor)] != wantCursor {
		t.Fatalf("cursor consult hash %q want %q", m.Rules["consult.md"][string(TargetCursor)], wantCursor)
	}
	if m.Rules["consult.md"][string(TargetCursor)] == wantPlain {
		t.Fatal("cursor wrap should change the consult hash")
	}
	for _, host := range []Target{TargetClaude, TargetAgentsMD, TargetCopilot} {
		if m.Rules["consult.md"][string(host)] != wantPlain {
			t.Fatalf("%s consult hash %q want %q", host, m.Rules["consult.md"][string(host)], wantPlain)
		}
	}
	skillSum := fileSHA256(t, filepath.Join(root, "skills", "record-decision", "SKILL.md"))
	if m.Skills["record-decision/SKILL.md"][string(TargetCursor)] != skillSum {
		t.Fatalf("cursor skill hash %q", m.Skills["record-decision/SKILL.md"][string(TargetCursor)])
	}
	if _, ok := m.Skills["record-decision/SKILL.md"][string(TargetAgentsMD)]; ok {
		t.Fatal("agents-md should not list skill hashes")
	}
}

func TestInstallCursorEmbeddedTemplates(t *testing.T) {
	root := t.TempDir()
	if err := Install(root, TargetCursor); err != nil {
		t.Fatal(err)
	}
	rule := filepath.Join(root, ".cursor", "rules", "archivist-consult.mdc")
	data, err := os.ReadFile(rule)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "Consult the archive") {
		t.Fatalf("embedded consult rule missing: %s", data)
	}
	if !strings.Contains(string(data), "do not trust chat memory") {
		t.Fatalf("consult rule should search rather than trust chat: %s", data)
	}
	recordPath := filepath.Join(root, ".cursor", "rules", "archivist-record.mdc")
	recordData, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(recordData)
	if !strings.Contains(got, "Distill lasting decisions") {
		t.Fatalf("record rule description missing distill guidance: %s", got)
	}
	if !strings.Contains(got, "Scan this conversation") {
		t.Fatalf("record rule missing conversation distill: %s", got)
	}
	if !strings.Contains(got, "Skip chat transcripts") {
		t.Fatalf("record rule missing glut bar: %s", got)
	}
	skill := filepath.Join(root, ".cursor", "skills", "record-decision", "SKILL.md")
	skillData, err := os.ReadFile(skill)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(skillData), "Search for the same topic") {
		t.Fatalf("record-decision skill should search first: %s", skillData)
	}
	feature := filepath.Join(root, ".cursor", "skills", "record-feature", "SKILL.md")
	if _, err := os.Stat(feature); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile(filepath.Join(root, ManifestFile))
	if err != nil {
		t.Fatal(err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	if m.Version != version.Version {
		t.Fatalf("manifest version %q", m.Version)
	}
	for _, name := range []string{"consult.md", "record.md", "refresh.md"} {
		hosts := m.Rules[name]
		if hosts[string(TargetCursor)] == "" || hosts[string(TargetClaude)] == "" || hosts[string(TargetAgentsMD)] == "" || hosts[string(TargetCopilot)] == "" {
			t.Fatalf("missing per-host rule hashes for %s: %v", name, hosts)
		}
		if hosts[string(TargetCursor)] == hosts[string(TargetClaude)] {
			t.Fatalf("%s cursor hash should differ from claude", name)
		}
		if hosts[string(TargetClaude)] != hosts[string(TargetAgentsMD)] || hosts[string(TargetClaude)] != hosts[string(TargetCopilot)] {
			t.Fatalf("%s concatenated hosts should share a hash: %v", name, hosts)
		}
	}
	if m.Skills["record-decision/SKILL.md"][string(TargetCursor)] == "" || m.Skills["record-decision/SKILL.md"][string(TargetClaude)] == "" {
		t.Fatalf("missing skill hashes: %v", m.Skills)
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
	assertManifest(t, root, TargetAgentsMD)
}

func TestCursorRuleDescriptions(t *testing.T) {
	if !strings.Contains(cursorRuleDescription("record"), "Distill lasting") {
		t.Fatal(cursorRuleDescription("record"))
	}
	if got := cursorRuleDescription("unknown"); got != "Archivist rule unknown" {
		t.Fatalf("fallback: %s", got)
	}
}

func TestParseTargetCodexIsAgentsMD(t *testing.T) {
	got, err := ParseTarget("codex")
	if err != nil || got != TargetAgentsMD {
		t.Fatalf("codex: %v %v", got, err)
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

func assertManifest(t *testing.T, root string, target Target) {
	t.Helper()
	m := readManifest(t, root)
	if m.Version != version.Version {
		t.Fatalf("manifest version %q want %q", m.Version, version.Version)
	}
	if m.Target != string(target) {
		t.Fatalf("manifest target %q want %q", m.Target, target)
	}
	if len(m.Rules) == 0 {
		t.Fatal("expected rule hashes")
	}
	for name, hosts := range m.Rules {
		for _, host := range hashTargets() {
			if hosts[string(host)] == "" {
				t.Fatalf("rule %s missing %s hash", name, host)
			}
		}
	}
}

func readManifest(t *testing.T, root string) Manifest {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ManifestFile))
	if err != nil {
		t.Fatal(err)
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func fileBytes(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func fileSHA256(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
