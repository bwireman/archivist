package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bwireman/archivist/internal/version"
	ruletmpl "github.com/bwireman/archivist/rules"
	skilltmpl "github.com/bwireman/archivist/skills"
)

// ManifestFile is written next to the checkout by Install.
const ManifestFile = ".archivist-install.json"

type Manifest struct {
	Version string                       `json:"version"`
	Commit  string                       `json:"commit,omitempty"`
	Target  string                       `json:"target"`
	Rules   map[string]map[string]string `json:"rules"`
	Skills  map[string]map[string]string `json:"skills"`
}

type Target string

const (
	TargetCursor   Target = "cursor"
	TargetClaude   Target = "claude"
	TargetAgentsMD Target = "agents-md"
	TargetCopilot  Target = "copilot"
)

func ParseTarget(s string) (Target, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "cursor":
		return TargetCursor, nil
	case "claude":
		return TargetClaude, nil
	case "agents-md", "agents", "codex":
		return TargetAgentsMD, nil
	case "copilot":
		return TargetCopilot, nil
	default:
		return "", fmt.Errorf("unknown target %q", s)
	}
}

// Install writes always-on rules and, for Cursor/Claude, on-demand skills.
func Install(repoRoot string, target Target) error {
	rules, err := loadRules(repoRoot)
	if err != nil {
		return err
	}
	skillFiles, err := loadSkills(repoRoot)
	if err != nil {
		return err
	}
	if err := installRules(repoRoot, target, rules); err != nil {
		return err
	}
	if err := InstallGithooks(repoRoot); err != nil {
		return err
	}
	switch target {
	case TargetAgentsMD, TargetCopilot:
	default:
		if err := installSkills(repoRoot, target, skillFiles); err != nil {
			return err
		}
	}
	return writeManifest(repoRoot, target, rules, skillFiles)
}

func loadRules(repoRoot string) (map[string][]byte, error) {
	fsys := ruleFS(repoRoot)
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			return nil, err
		}
		out[e.Name()] = data
	}
	return out, nil
}

func loadSkills(repoRoot string) (map[string][]byte, error) {
	fsys := skillFS(repoRoot)
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		rel := path.Join(e.Name(), "SKILL.md")
		data, err := fs.ReadFile(fsys, rel)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out[rel] = data
	}
	return out, nil
}

func installRules(repoRoot string, target Target, rules map[string][]byte) error {
	var parts []string
	for _, name := range sortedKeys(rules) {
		rendered := renderRule(target, name, rules[name])
		if target == TargetCursor {
			stem := strings.TrimSuffix(name, ".md")
			dst := filepath.Join(repoRoot, ".cursor", "rules", "archivist-"+stem+".mdc")
			if err := writeFile(dst, rendered); err != nil {
				return err
			}
			continue
		}
		parts = append(parts, rendered)
	}
	if len(parts) == 0 {
		return nil
	}
	combined := strings.Join(parts, "\n\n") + "\n"
	switch target {
	case TargetClaude:
		return writeFile(filepath.Join(repoRoot, "CLAUDE.md"), combined)
	case TargetAgentsMD:
		return writeFile(filepath.Join(repoRoot, "AGENTS.md"), combined)
	case TargetCopilot:
		return writeFile(filepath.Join(repoRoot, ".github", "copilot-instructions.md"), combined)
	}
	return nil
}

func installSkills(repoRoot string, target Target, skillFiles map[string][]byte) error {
	for _, rel := range sortedKeys(skillFiles) {
		name := path.Dir(rel)
		var dst string
		switch target {
		case TargetCursor:
			dst = filepath.Join(repoRoot, ".cursor", "skills", name, "SKILL.md")
		case TargetClaude:
			dst = filepath.Join(repoRoot, ".claude", "skills", name, "SKILL.md")
		default:
			continue
		}
		if err := writeFile(dst, string(skillFiles[rel])); err != nil {
			return err
		}
	}
	return nil
}

func writeManifest(repoRoot string, target Target, rules, skillFiles map[string][]byte) error {
	m := Manifest{
		Version: version.Version,
		Commit:  version.Revision(),
		Target:  string(target),
		Rules:   hashRulesByTarget(rules),
		Skills:  hashSkillsByTarget(skillFiles),
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFile(filepath.Join(repoRoot, ManifestFile), string(data))
}

func hashTargets() []Target {
	return []Target{TargetCursor, TargetClaude, TargetAgentsMD, TargetCopilot}
}

func renderRule(target Target, name string, data []byte) string {
	stem := strings.TrimSuffix(name, ".md")
	body := strings.TrimSpace(string(data))
	if target == TargetCursor {
		return wrapCursorRule(body, stem)
	}
	return body
}

func hashRulesByTarget(rules map[string][]byte) map[string]map[string]string {
	out := map[string]map[string]string{}
	for _, name := range sortedKeys(rules) {
		per := map[string]string{}
		for _, t := range hashTargets() {
			per[string(t)] = sha256Hex([]byte(renderRule(t, name, rules[name])))
		}
		out[name] = per
	}
	return out
}

func hashSkillsByTarget(files map[string][]byte) map[string]map[string]string {
	out := map[string]map[string]string{}
	for _, rel := range sortedKeys(files) {
		sum := sha256Hex(files[rel])
		per := map[string]string{}
		for _, t := range hashTargets() {
			if t == TargetAgentsMD || t == TargetCopilot {
				continue
			}
			per[string(t)] = sum
		}
		out[rel] = per
	}
	return out
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sortedKeys(files map[string][]byte) []string {
	keys := make([]string, 0, len(files))
	for k := range files {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func ruleFS(repoRoot string) fs.FS {
	if local := dirWithSuffix(filepath.Join(repoRoot, "rules"), ".md"); local != nil {
		return local
	}
	return ruletmpl.FS
}

func skillFS(repoRoot string) fs.FS {
	dir := filepath.Join(repoRoot, "skills")
	ents, err := os.ReadDir(dir)
	if err != nil {
		return skilltmpl.FS
	}
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, e.Name(), "SKILL.md")); err == nil {
			return os.DirFS(dir)
		}
	}
	return skilltmpl.FS
}

func dirWithSuffix(dir, suffix string) fs.FS {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	for _, e := range ents {
		if !e.IsDir() && strings.HasSuffix(e.Name(), suffix) {
			return os.DirFS(dir)
		}
	}
	return nil
}

func wrapCursorRule(body, name string) string {
	return fmt.Sprintf("---\ndescription: %s\nalwaysApply: true\n---\n\n%s\n", cursorRuleDescription(name), body)
}

func cursorRuleDescription(name string) string {
	switch name {
	case "consult":
		return "Consult the Archivist archive before guessing APIs, defaults, or past decisions"
	case "record":
		return "Distill lasting decisions, rules, and features from this conversation; skip chat glut"
	case "refresh":
		return "Rebuild the Archivist index, embeddings, and export after record or code changes"
	case "ask":
		return "Ask the user when a choice or preference is unsettled after archive search"
	default:
		return "Archivist rule " + name
	}
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
