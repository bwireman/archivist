package skills

import (
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
	hook := filepath.Join(root, ".githooks", "post-commit")
	if _, err := os.Stat(hook); err != nil {
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
	plan := filepath.Join(root, ".cursor", "skills", "plan-changes", "SKILL.md")
	planData, err := os.ReadFile(plan)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(planData), "Search the archive before asking") {
		t.Fatalf("plan-changes skill should search then ask: %s", planData)
	}
	if !strings.Contains(string(planData), "always-on ask rule") {
		t.Fatalf("plan-changes skill should leave the short gap to the ask rule: %s", planData)
	}
	m := readManifest(t, root)
	if m.Version != version.Version {
		t.Fatalf("manifest version %q", m.Version)
	}
	askPath := filepath.Join(root, ".cursor", "rules", "archivist-ask.mdc")
	askData, err := os.ReadFile(askPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(askData), "Ask when unsure") {
		t.Fatalf("embedded ask rule missing: %s", askData)
	}
	if !strings.Contains(string(askData), "Ask the user when a choice or preference") {
		t.Fatalf("ask rule description missing: %s", askData)
	}
	currentPath := filepath.Join(root, ".cursor", "rules", "archivist-current.mdc")
	currentData, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(currentData), "alwaysApply: true") || !strings.Contains(string(currentData), "archivist index") {
		t.Fatalf("current rule should stay always-on and reindex: %s", currentData)
	}
	if !strings.Contains(string(currentData), "Keep the Archivist code map current") {
		t.Fatalf("current rule description missing: %s", currentData)
	}
	citePath := filepath.Join(root, ".cursor", "rules", "archivist-cite.mdc")
	citeData, err := os.ReadFile(citePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(citeData), "alwaysApply: true") || !strings.Contains(string(citeData), "call cite") {
		t.Fatalf("cite rule should be its own always-on install: %s", citeData)
	}
	if !strings.Contains(string(citeData), "Cite an Archivist record") {
		t.Fatalf("cite rule description missing: %s", citeData)
	}
	for _, name := range []string{"ask.md", "cite.md", "consult.md", "current.md", "record.md", "refresh.md"} {
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
	if m.Skills["plan-changes/SKILL.md"][string(TargetCursor)] == "" || m.Skills["plan-changes/SKILL.md"][string(TargetClaude)] == "" {
		t.Fatalf("missing plan-changes skill hashes: %v", m.Skills)
	}
	initSkill := filepath.Join(root, ".cursor", "skills", "init-archive", "SKILL.md")
	initData, err := os.ReadFile(initSkill)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(initData), "Scan") || !strings.Contains(string(initData), "archivist embed --once") {
		t.Fatalf("init-archive skill should scan then embed: %s", initData)
	}
	if m.Skills["init-archive/SKILL.md"][string(TargetCursor)] == "" || m.Skills["init-archive/SKILL.md"][string(TargetClaude)] == "" {
		t.Fatalf("missing init-archive skill hashes: %v", m.Skills)
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
	if !strings.Contains(cursorRuleDescription("ask"), "Ask the user when a choice or preference") {
		t.Fatal(cursorRuleDescription("ask"))
	}
	if !strings.Contains(cursorRuleDescription("current"), "Keep the Archivist code map current") {
		t.Fatal(cursorRuleDescription("current"))
	}
	if !strings.Contains(cursorRuleDescription("cite"), "Cite an Archivist record") {
		t.Fatal(cursorRuleDescription("cite"))
	}
	if got := cursorRuleDescription("unknown"); got != "Archivist rule unknown" {
		t.Fatalf("fallback: %s", got)
	}
}

func TestRulesStale(t *testing.T) {
	root := t.TempDir()
	stale, target, err := RulesStale(root)
	if err != nil || stale || target != "" {
		t.Fatalf("missing manifest: stale=%v target=%q err=%v", stale, target, err)
	}
	if err := Install(root, TargetCursor); err != nil {
		t.Fatal(err)
	}
	stale, target, err = RulesStale(root)
	if err != nil || stale || target != string(TargetCursor) {
		t.Fatalf("fresh install: stale=%v target=%q err=%v", stale, target, err)
	}
	rule := filepath.Join(root, ".cursor", "rules", "archivist-consult.mdc")
	if err := os.WriteFile(rule, []byte("edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stale, target, err = RulesStale(root)
	if err != nil || !stale || target != string(TargetCursor) {
		t.Fatalf("edited rule: stale=%v target=%q err=%v", stale, target, err)
	}
	if !strings.Contains(StaleRulesNotice(target), "archivist skills install --target cursor") {
		t.Fatal(StaleRulesNotice(target))
	}

	agents := t.TempDir()
	if err := Install(agents, TargetAgentsMD); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agents, "AGENTS.md"), []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stale, target, err = RulesStale(agents)
	if err != nil || !stale || target != string(TargetAgentsMD) {
		t.Fatalf("agents-md: stale=%v target=%q err=%v", stale, target, err)
	}

	broken := t.TempDir()
	if err := os.WriteFile(filepath.Join(broken, ManifestFile), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	stale, target, err = RulesStale(broken)
	if err != nil || !stale || target != "" {
		t.Fatalf("bad manifest: stale=%v target=%q err=%v", stale, target, err)
	}
	if !strings.Contains(StaleRulesNotice(target), "--target cursor") {
		t.Fatal(StaleRulesNotice(target))
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
	m, err := loadManifest(root)
	if err != nil {
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
	return sha256Hex(fileBytes(t, path))
}
