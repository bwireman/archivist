// Package skills installs agent rules and skills.
package skills

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bwireman/archivist/internal/version"
	ruletmpl "github.com/bwireman/archivist/rules"
	skilltmpl "github.com/bwireman/archivist/skills"
)

// ManifestFile is written next to the checkout by Install.
const ManifestFile = ".archivist-install.json"

type Manifest struct {
	Version string            `json:"version"`
	Commit  string            `json:"commit,omitempty"`
	Target  string            `json:"target"`
	CLI     string            `json:"cli,omitempty"`
	Rules   map[string]string `json:"rules"`
	Skills  map[string]string `json:"skills,omitempty"`
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
	return InstallCLI(repoRoot, target, "")
}

// InstallCLI is Install with the bare `archivist` command in rules and skills
// replaced by cli (for example `./bin/archivist`). The manifest records cli so
// RulesStale compares against the same rendering.
func InstallCLI(repoRoot string, target Target, cli string) error {
	rules, err := loadRules(repoRoot)
	if err != nil {
		return err
	}
	skillFiles, err := loadSkills(repoRoot)
	if err != nil {
		return err
	}
	rules = pinCLI(rules, cli)
	skillFiles = pinCLI(skillFiles, cli)
	if err := installRules(repoRoot, target, rules); err != nil {
		return err
	}
	if installsSkills(target) {
		if err := installSkills(repoRoot, target, skillFiles); err != nil {
			return err
		}
	} else {
		skillFiles = nil
	}
	return writeManifest(repoRoot, target, cli, rules, skillFiles)
}

func pinCLI(files map[string][]byte, cli string) map[string][]byte {
	if cli == "" {
		return files
	}
	out := make(map[string][]byte, len(files))
	for name, data := range files {
		out[name] = []byte(PinCLI(string(data), cli))
	}
	return out
}

// PinCLI replaces each standalone `archivist` command word with cli. A word
// preceded by a letter, digit, `_`, `.`, or `/` (a path or identifier) is kept.
func PinCLI(text, cli string) string {
	const word = "archivist"
	var b strings.Builder
	for i := 0; i < len(text); {
		j := strings.Index(text[i:], word)
		if j < 0 {
			b.WriteString(text[i:])
			break
		}
		j += i
		end := j + len(word)
		before := j > 0 && isPathByte(text[j-1])
		after := end < len(text) && isWordByte(text[end])
		b.WriteString(text[i:j])
		if before || after {
			b.WriteString(word)
		} else {
			b.WriteString(cli)
		}
		i = end
	}
	return b.String()
}

func isWordByte(c byte) bool {
	return c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func isPathByte(c byte) bool {
	return isWordByte(c) || c == '.' || c == '/'
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
	for rel, body := range renderedRuleFiles(target, rules) {
		if err := writeFile(filepath.Join(repoRoot, rel), body); err != nil {
			return err
		}
	}
	return nil
}

// renderedRuleFiles is what Install writes for rules. Cursor gets one file per
// rule; other hosts get a single concatenated file.
func renderedRuleFiles(target Target, rules map[string][]byte) map[string]string {
	out := map[string]string{}
	if target == TargetCursor {
		for _, name := range sortedKeys(rules) {
			stem := strings.TrimSuffix(name, ".md")
			rel := filepath.Join(".cursor", "rules", "archivist-"+stem+".mdc")
			out[rel] = renderRule(target, name, rules[name])
		}
		return out
	}
	var parts []string
	for _, name := range sortedKeys(rules) {
		parts = append(parts, renderRule(target, name, rules[name]))
	}
	if len(parts) == 0 {
		return out
	}
	body := strings.Join(parts, "\n\n") + "\n"
	switch target {
	case TargetClaude:
		out["CLAUDE.md"] = body
	case TargetAgentsMD:
		out["AGENTS.md"] = body
	case TargetCopilot:
		out[filepath.Join(".github", "copilot-instructions.md")] = body
	}
	return out
}

// RulesStale reports whether a previous skills install's rule files differ from
// the templates Install would write now. A missing manifest is not stale.
// target is the host recorded in the manifest, or empty when it cannot be used.
func RulesStale(repoRoot string) (stale bool, target string, err error) {
	if home, herr := homeManifest(); herr == nil {
		stale, err = userSkillsStale(home)
		if err != nil || stale {
			return stale, "user", err
		}
	} else if !errors.Is(herr, os.ErrNotExist) {
		return false, "", herr
	}
	m, err := loadManifest(repoRoot)
	if errors.Is(err, os.ErrNotExist) {
		return false, "", nil
	}
	if err != nil {
		var syn *json.SyntaxError
		var typ *json.UnmarshalTypeError
		if errors.As(err, &syn) || errors.As(err, &typ) {
			return true, manifestTarget(repoRoot), nil
		}
		return false, "", err
	}
	parsed, perr := ParseTarget(m.Target)
	if perr != nil {
		return true, "", nil
	}
	rules, err := loadRules(repoRoot)
	if err != nil {
		return false, m.Target, err
	}
	rules = pinCLI(rules, m.CLI)
	for rel, want := range renderedRuleFiles(parsed, rules) {
		got, rerr := os.ReadFile(filepath.Join(repoRoot, rel))
		if rerr != nil {
			if errors.Is(rerr, os.ErrNotExist) {
				return true, m.Target, nil
			}
			return false, m.Target, rerr
		}
		if string(got) != want {
			return true, m.Target, nil
		}
	}
	return false, m.Target, nil
}

// StaleRulesNotice is the stderr prompt when installed rules do not match.
func StaleRulesNotice(target string) string {
	if target == "user" {
		return "archivist: installed agent rules are out of date. Reinstall with:\n  archivist skills install --user\n"
	}
	if target == "" {
		target = "cursor"
	}
	return fmt.Sprintf("archivist: installed agent rules are out of date. Reinstall with:\n  archivist skills install --target %s\n", target)
}

func loadManifest(repoRoot string) (Manifest, error) {
	data, err := os.ReadFile(filepath.Join(repoRoot, ManifestFile))
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

// manifestTarget reads the host from a manifest whose rule hashes use an
// older shape. A syntax error returns "".
func manifestTarget(repoRoot string) string {
	data, err := os.ReadFile(filepath.Join(repoRoot, ManifestFile))
	if err != nil {
		return ""
	}
	var head struct {
		Target string `json:"target"`
	}
	if json.Unmarshal(data, &head) != nil {
		return ""
	}
	if _, err := ParseTarget(head.Target); err != nil {
		return ""
	}
	return head.Target
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

func writeManifest(repoRoot string, target Target, cli string, rules, skillFiles map[string][]byte) error {
	m := Manifest{
		Version: version.Version,
		Commit:  version.Revision(),
		Target:  string(target),
		CLI:     cli,
		Rules:   hashRules(target, rules),
		Skills:  hashSkills(skillFiles),
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFile(filepath.Join(repoRoot, ManifestFile), string(data))
}

func installsSkills(target Target) bool {
	switch target {
	case TargetCursor, TargetClaude:
		return true
	default:
		return false
	}
}

func hashRules(target Target, rules map[string][]byte) map[string]string {
	out := map[string]string{}
	for _, name := range sortedKeys(rules) {
		out[name] = sha256Hex([]byte(renderRule(target, name, rules[name])))
	}
	return out
}

func hashSkills(files map[string][]byte) map[string]string {
	if len(files) == 0 {
		return nil
	}
	out := map[string]string{}
	for _, rel := range sortedKeys(files) {
		out[rel] = sha256Hex(files[rel])
	}
	return out
}

func renderRule(target Target, name string, data []byte) string {
	stem := strings.TrimSuffix(name, ".md")
	body := strings.TrimSpace(string(data))
	if target == TargetCursor {
		return wrapCursorRule(body, stem)
	}
	return body
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sortedKeys(files map[string][]byte) []string {
	return slices.Sorted(maps.Keys(files))
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
		return "Write a short record when a later session would look it up; skip chat glut"
	case "refresh":
		return "Rebuild the Archivist index, embeddings, and export after record or code changes"
	case "current":
		return "Keep the Archivist code map current as source code changes"
	case "ask":
		return "Ask the user when a choice or preference is unsettled after archive search"
	case "cite":
		return "Cite an Archivist record that changed the work before the turn ends"
	case "link":
		return "Link a record in another checkout when it would help a later session in this one"
	case "review":
		return "Review the archive and ask before a condensation, update, or retire"
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

// InstallUser writes skills once under ~/.agents/skills and one archivist
// server into ~/.cursor/mcp.json. It does not write ~/.cursor/rules.
func InstallUser(cli string) error {
	skillFiles, err := loadSkills("")
	if err != nil {
		return err
	}
	skillFiles = pinCLI(skillFiles, cli)
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	for _, rel := range sortedKeys(skillFiles) {
		name := path.Dir(rel)
		dst := filepath.Join(home, ".agents", "skills", name, "SKILL.md")
		if err := writeFile(dst, string(skillFiles[rel])); err != nil {
			return err
		}
	}
	if err := writeFile(filepath.Join(home, ".archivist", "install.json"), string(mustManifest("user", cli, nil, skillFiles))); err != nil {
		return err
	}
	return mergeUserMCP(filepath.Join(home, ".cursor", "mcp.json"))
}

func mustManifest(target, cli string, rules, skillFiles map[string][]byte) []byte {
	m := Manifest{
		Version: version.Version,
		Commit:  version.Revision(),
		Target:  target,
		CLI:     cli,
		Rules:   hashRules(TargetCursor, rules),
		Skills:  hashSkills(skillFiles),
	}
	if rules == nil {
		m.Rules = map[string]string{}
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		panic(err)
	}
	return append(data, '\n')
}

func homeManifest() (Manifest, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Manifest{}, err
	}
	data, err := os.ReadFile(filepath.Join(home, ".archivist", "install.json"))
	if err != nil {
		return Manifest{}, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return Manifest{}, err
	}
	return m, nil
}

func userSkillsStale(m Manifest) (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}
	files, err := loadSkills("")
	if err != nil {
		return false, err
	}
	files = pinCLI(files, m.CLI)
	for rel, want := range files {
		name := path.Dir(rel)
		got, rerr := os.ReadFile(filepath.Join(home, ".agents", "skills", name, "SKILL.md"))
		if rerr != nil {
			if errors.Is(rerr, os.ErrNotExist) {
				return true, nil
			}
			return false, rerr
		}
		if string(got) != string(want) {
			return true, nil
		}
	}
	return false, nil
}

func mergeUserMCP(path string) error {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		data = []byte(`{"mcpServers":{}}`)
	} else if err != nil {
		return err
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	servers := map[string]json.RawMessage{}
	if raw, ok := cfg["mcpServers"]; ok && len(raw) > 0 && string(raw) != "null" {
		if err := json.Unmarshal(raw, &servers); err != nil {
			return err
		}
	}
	if _, ok := servers["archivist"]; !ok {
		servers["archivist"] = json.RawMessage(`{"command":"archivist","args":["mcp"]}`)
	}
	encoded, err := json.Marshal(servers)
	if err != nil {
		return err
	}
	cfg["mcpServers"] = encoded
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return writeFile(path, string(out))
}
