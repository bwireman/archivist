package migrate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
)

var statusRe = regexp.MustCompile(`(?m)^- Status:\s*(\S+)`)
var dateRe = regexp.MustCompile(`(?m)^- Date:\s*(.+)$`)

// Records converts legacy ADR markdown files to typed records with front matter.
func Records(repoRoot string) error {
	if err := migrateDir(repoRoot, filepath.Join(repoRoot, config.DefaultDecisionsDir), record.ScopeRepo); err != nil {
		return err
	}
	if err := migrateDir(repoRoot, filepath.Join(repoRoot, config.DefaultGlobalDecisionsDir), record.ScopeGlobal); err != nil {
		return err
	}
	oldDir := filepath.Join(config.ArchivistHome(), "decisions")
	newDir := config.UserRecordsDir()
	if oldDir != newDir {
		if _, err := os.Stat(oldDir); err == nil {
			if err := os.Rename(oldDir, newDir); err != nil {
				return fmt.Errorf("move user records: %w", err)
			}
		}
	}
	if newDir != "" {
		return migrateDir("", newDir, record.ScopeDev)
	}
	return nil
}

func migrateDir(repoRoot, dir string, scope record.Scope) error {
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || e.Name() == ".gitkeep" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(data)
		if strings.HasPrefix(content, "---\nid:") {
			continue
		}
		rec, err := legacyToRecord(repoRoot, path, content, scope)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if err := os.WriteFile(path, []byte(record.Serialize(rec)), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func legacyToRecord(repoRoot, path, content string, scope record.Scope) (*record.Record, error) {
	title := ""
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "# ") {
			title = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "# "))
			break
		}
	}
	status := record.StatusAccepted
	if m := statusRe.FindStringSubmatch(content); len(m) > 1 {
		status = record.Status(strings.ToLower(m[1]))
	}
	sourcePath := filepath.Base(path)
	if repoRoot != "" {
		rel, err := filepath.Rel(repoRoot, path)
		if err == nil && !strings.HasPrefix(rel, "..") {
			sourcePath = filepath.ToSlash(rel)
		}
	}
	rec := &record.Record{
		ID:         record.NewID(),
		Slug:       record.SlugFromPath(path),
		Type:       record.TypeDecision,
		Scope:      scope,
		Title:      title,
		Status:     status,
		Body:       strings.TrimSpace(content),
		SourcePath: sourcePath,
	}
	if scope == record.ScopeDev {
		rec.SourcePath = config.VirtualUserADRPath(filepath.Base(path))
	}
	if err := rec.Validate(); err != nil {
		return nil, err
	}
	rec.ContentHash = record.ContentHash(rec)
	return rec, nil
}
