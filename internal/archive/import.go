package archive

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
)

type ImportResult struct {
	Imported int
	Updated  int
	Skipped  int
}

// Import upserts typed markdown from record directories and export copies into SQLite.
// It never deletes DB-only records.
func (s *Service) Import() (ImportResult, error) {
	var res ImportResult
	if s.Records.Repo != "" {
		dir := filepath.Join(s.RepoRoot, s.Records.Repo)
		prefix := s.Records.Repo
		if err := walkMarkdown(dir, func(rel, abs string) error {
			sourcePath := filepath.ToSlash(filepath.Join(prefix, rel))
			return s.importFile(sourcePath, abs, &res)
		}); err != nil {
			return res, err
		}
	}
	if s.Records.GlobalInRepo() {
		dir := filepath.Join(s.RepoRoot, s.Records.Global)
		prefix := s.Records.Global
		if err := walkMarkdown(dir, func(rel, abs string) error {
			sourcePath := filepath.ToSlash(filepath.Join(prefix, rel))
			return s.importFile(sourcePath, abs, &res)
		}); err != nil {
			return res, err
		}
	}
	if !s.Records.GlobalInRepo() {
		dir := s.Records.GlobalDir(s.RepoRoot)
		if err := s.walkHomeGlobal(dir, func(virt, abs string) error {
			return s.importFile(virt, abs, &res)
		}); err != nil {
			return res, err
		}
	}
	if dir := s.devDir(); dir != "" {
		if err := walkMarkdown(dir, func(rel, abs string) error {
			virt := config.VirtualUserADRPath(rel)
			return s.importFile(virt, abs, &res)
		}); err != nil {
			return res, err
		}
	}
	exportRecords := filepath.Join(s.RepoRoot, s.Records.Export, "records")
	if err := walkMarkdown(exportRecords, func(rel, abs string) error {
		rel = filepath.ToSlash(filepath.Join("records", rel))
		return s.importFile(rel, abs, &res)
	}); err != nil {
		return res, err
	}
	return res, nil
}

func walkMarkdown(dir string, fn func(rel, abs string) error) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") || d.Name() == ".gitkeep" {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		return fn(filepath.ToSlash(rel), path)
	})
}

func (s *Service) walkHomeGlobal(dir string, fn func(virt, abs string) error) error {
	if dir == "" {
		return nil
	}
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}
	dev := s.devDir()
	return filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if dev != "" && samePath(path, dev) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		virt := config.VirtualHomeGlobalPath(filepath.ToSlash(rel))
		return fn(virt, path)
	})
}

func samePath(a, b string) bool {
	a, err1 := filepath.Abs(a)
	b, err2 := filepath.Abs(b)
	if err1 != nil || err2 != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	return a == b
}

func (s *Service) importFile(sourcePath, abs string, res *ImportResult) error {
	data, err := os.ReadFile(abs)
	if err != nil {
		return err
	}
	content := string(data)
	if !record.HasFrontMatterID(content) {
		return nil
	}
	parsed, err := record.ParseFile(sourcePath, content)
	if err != nil {
		return fmt.Errorf("%s: %w", sourcePath, err)
	}
	existing, st, ok := s.findByID(parsed.ID)
	if !ok {
		existing, st, ok = s.findByPath(sourcePath)
	}
	if ok {
		if existing.ContentHash == parsed.ContentHash {
			res.Skipped++
			return nil
		}
		parsed.ID = existing.ID
		parsed.CreatedAt = existing.CreatedAt
		parsed.SourcePath = existing.SourcePath
		if err := st.UpsertRecord(parsed); err != nil {
			return err
		}
		res.Updated++
		return nil
	}
	st = s.storeFor(parsed.Scope)
	if st == nil {
		return fmt.Errorf("no store for scope %s", parsed.Scope)
	}
	parsed.SourcePath = sourcePath
	if err := st.UpsertRecord(parsed); err != nil {
		return err
	}
	res.Imported++
	return nil
}

func (s *Service) findByID(id string) (*record.Record, *store.Store, bool) {
	for _, st := range []*store.Store{s.RepoDB, s.HomeDB} {
		if st == nil {
			continue
		}
		rec, ok, err := st.GetRecordByID(id)
		if err != nil {
			continue
		}
		if ok {
			return rec, st, true
		}
	}
	return nil, nil, false
}

func (s *Service) findByPath(path string) (*record.Record, *store.Store, bool) {
	for _, st := range []*store.Store{s.RepoDB, s.HomeDB} {
		if st == nil {
			continue
		}
		rec, ok, err := st.GetRecordByPath(path)
		if err != nil {
			continue
		}
		if ok {
			return rec, st, true
		}
	}
	return nil, nil, false
}
