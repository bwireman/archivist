package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// HomePlanPath is ~/.archivist/checkouts/<hex>/plans/<slug>.md.
func HomePlanPath(checkout, slug string) (string, error) {
	canon, err := Canonical(checkout)
	if err != nil {
		return "", err
	}
	dir := CheckoutDir(canon)
	if dir == "" {
		return "", errors.New("cannot resolve home plan directory (set $HOME)")
	}
	return filepath.Join(dir, "plans", slug+".md"), nil
}

// LegacyPlanPath is <checkout>/.archivist/plans/<slug>.md.
func LegacyPlanPath(checkout, slug string) string {
	return filepath.Join(checkout, ".archivist", "plans", slug+".md")
}

// MoveSettledPlans moves proposed, rejected, and done plans to the home
// checkout directory. Accepted and in-progress plans stay put.
func MoveSettledPlans(checkout string) error {
	dir := filepath.Join(checkout, ".archivist", "plans")
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		src := filepath.Join(dir, entry.Name())
		body, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		switch planStatus(string(body)) {
		case "proposed", "rejected", "done":
		default:
			continue
		}
		slug := strings.TrimSuffix(entry.Name(), ".md")
		dst, err := HomePlanPath(checkout, slug)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dst, body, 0o644); err != nil {
			return err
		}
		if err := os.Remove(src); err != nil {
			return err
		}
	}
	return nil
}

func planStatus(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Status:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Status:"))
		}
	}
	return ""
}
