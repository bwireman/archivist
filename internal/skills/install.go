package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	dir := filepath.Join(repoRoot, "rules")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var parts []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		body := strings.TrimSpace(string(data))
		switch target {
		case TargetCursor:
			dst := filepath.Join(repoRoot, ".cursor", "rules", "archivist-"+name+".mdc")
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(dst, []byte(wrapCursorRule(body, name)), 0o644); err != nil {
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
	skillsDir := filepath.Join(repoRoot, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		src := filepath.Join(skillsDir, e.Name(), "SKILL.md")
		data, err := os.ReadFile(src)
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

func wrapCursorRule(body, name string) string {
	return fmt.Sprintf("---\ndescription: Archivist rule %s\nalwaysApply: true\n---\n\n%s\n", name, body)
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
