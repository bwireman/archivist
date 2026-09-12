package skills

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	ruletmpl "github.com/bwireman/archivist/rules"
	skilltmpl "github.com/bwireman/archivist/skills"
)

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
	case "agents-md", "agents":
		return TargetAgentsMD, nil
	case "copilot":
		return TargetCopilot, nil
	default:
		return "", fmt.Errorf("unknown target %q", s)
	}
}

// Install writes always-on rules and, for Cursor/Claude, on-demand skills.
func Install(repoRoot string, target Target) error {
	if err := installRules(repoRoot, target); err != nil {
		return err
	}
	switch target {
	case TargetAgentsMD, TargetCopilot:
		return nil
	default:
		return installSkills(repoRoot, target)
	}
}

func installRules(repoRoot string, target Target) error {
	fsys := ruleFS(repoRoot)
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return err
	}
	var parts []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			return err
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		body := strings.TrimSpace(string(data))
		switch target {
		case TargetCursor:
			dst := filepath.Join(repoRoot, ".cursor", "rules", "archivist-"+name+".mdc")
			if err := writeFile(dst, wrapCursorRule(body, name)); err != nil {
				return err
			}
		default:
			parts = append(parts, body)
		}
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

func installSkills(repoRoot string, target Target) error {
	fsys := skillFS(repoRoot)
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := fs.ReadFile(fsys, path.Join(e.Name(), "SKILL.md"))
		if err != nil {
			continue
		}
		var dst string
		switch target {
		case TargetCursor:
			dst = filepath.Join(repoRoot, ".cursor", "skills", e.Name(), "SKILL.md")
		case TargetClaude:
			dst = filepath.Join(repoRoot, ".claude", "skills", e.Name(), "SKILL.md")
		default:
			continue
		}
		if err := writeFile(dst, string(data)); err != nil {
			return err
		}
	}
	return nil
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
	return fmt.Sprintf("---\ndescription: Archivist rule %s\nalwaysApply: true\n---\n\n%s\n", name, body)
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
