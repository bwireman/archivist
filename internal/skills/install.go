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

func Install(repoRoot string, target Target) error {
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
		dst := destPath(repoRoot, target, e.Name())
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		content := string(data)
		if target == TargetCursor {
			content = wrapCursorMDC(content, e.Name())
		}
		if err := os.WriteFile(dst, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func destPath(repoRoot string, target Target, name string) string {
	switch target {
	case TargetCursor:
		return filepath.Join(repoRoot, ".cursor", "rules", "archivist-"+name+".mdc")
	case TargetClaude:
		return filepath.Join(repoRoot, ".claude", "skills", name, "SKILL.md")
	case TargetAgentsMD:
		return filepath.Join(repoRoot, "AGENTS.md")
	case TargetCopilot:
		return filepath.Join(repoRoot, ".github", "copilot-instructions.md")
	default:
		return filepath.Join(repoRoot, name+".md")
	}
}

func wrapCursorMDC(body, name string) string {
	return fmt.Sprintf("---\ndescription: Archivist skill %s\nalwaysApply: false\n---\n\n%s\n", name, body)
}
