package archive

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
)

type Service struct {
	RepoRoot string
	Records  config.RecordsConfig
	RepoDB   *store.Store
	HomeDB   *store.Store
}

func New(repoRoot string, cfg *config.Config, repoDB, homeDB *store.Store) *Service {
	recs := config.Default().Records
	if cfg != nil {
		recs = cfg.Records
	}
	return &Service{RepoRoot: repoRoot, Records: recs, RepoDB: repoDB, HomeDB: homeDB}
}

func (s *Service) Remember(rec *record.Record) (string, error) {
	if rec.Slug == "" {
		rec.Slug = slugify(rec.Title)
	}
	if rec.SourcePath == "" {
		rec.SourcePath = s.defaultPath(rec)
	}
	if err := s.adoptExistingIdentity(rec); err != nil {
		return "", err
	}
	if rec.ID == "" {
		rec.ID = record.NewID()
	}
	if err := rec.Validate(); err != nil {
		return "", err
	}
	rec.ContentHash = record.ContentHash(rec)

	abs := s.absPath(rec.SourcePath)
	if err := writeFileAtomic(abs, []byte(record.Serialize(rec))); err != nil {
		return "", err
	}
	st := s.storeFor(rec.Scope)
	if st == nil {
		return "", fmt.Errorf("no store for scope %s", rec.Scope)
	}
	if err := st.UpsertRecord(rec); err != nil {
		return "", err
	}
	return rec.ID, nil
}

func (s *Service) Update(id string, fn func(*record.Record) error) error {
	rec, st, err := s.find(id)
	if err != nil {
		return err
	}
	if err := fn(rec); err != nil {
		return err
	}
	rec.UpdatedAt = time.Now().UTC()
	rec.ContentHash = record.ContentHash(rec)
	if err := rec.Validate(); err != nil {
		return err
	}
	abs := s.absPath(rec.SourcePath)
	if err := writeFileAtomic(abs, []byte(record.Serialize(rec))); err != nil {
		return err
	}
	return st.UpsertRecord(rec)
}

func (s *Service) adoptExistingIdentity(rec *record.Record) error {
	st := s.storeFor(rec.Scope)
	if st != nil {
		existing, ok, err := st.GetRecordByPath(rec.SourcePath)
		if err != nil {
			return err
		}
		if ok {
			rec.ID = existing.ID
			rec.CreatedAt = existing.CreatedAt
			return nil
		}
	}
	data, err := os.ReadFile(s.absPath(rec.SourcePath))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	parsed, err := record.ParseFile(rec.SourcePath, string(data))
	if err != nil {
		return nil
	}
	if parsed.ID != "" {
		rec.ID = parsed.ID
		rec.CreatedAt = parsed.CreatedAt
	}
	return nil
}

func writeFileAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

func (s *Service) Retire(id, supersededBy string) error {
	return s.Update(id, func(r *record.Record) error {
		r.Status = record.StatusSuperseded
		r.SupersededBy = supersededBy
		return nil
	})
}

func (s *Service) Get(idOrSlug string) (*record.Record, error) {
	rec, _, err := s.find(idOrSlug)
	return rec, err
}

func (s *Service) find(idOrSlug string) (*record.Record, *store.Store, error) {
	for _, st := range []*store.Store{s.RepoDB, s.HomeDB} {
		if st == nil {
			continue
		}
		rec, ok, err := st.GetRecordByID(idOrSlug)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			return rec, st, nil
		}
		rec, ok, err = st.GetRecordBySlug(idOrSlug)
		if err != nil {
			return nil, nil, err
		}
		if ok {
			return rec, st, nil
		}
	}
	return nil, nil, fmt.Errorf("record not found: %s", idOrSlug)
}

func (s *Service) storeFor(scope record.Scope) *store.Store {
	switch scope {
	case record.ScopeDev, record.ScopeGlobal:
		return s.HomeDB
	default:
		return s.RepoDB
	}
}

func (s *Service) defaultPath(rec *record.Record) string {
	slug := rec.Slug
	if slug == "" {
		slug = slugify(rec.Title)
	}
	name := slug + ".md"
	switch rec.Scope {
	case record.ScopeGlobal:
		if s.Records.GlobalInRepo() {
			return filepath.ToSlash(filepath.Join(s.Records.Global, name))
		}
		return config.VirtualHomeGlobalPath(name)
	case record.ScopeDev:
		return config.VirtualUserADRPath(name)
	default:
		return filepath.ToSlash(filepath.Join(s.Records.Repo, name))
	}
}

func (s *Service) absPath(sourcePath string) string {
	if strings.HasPrefix(sourcePath, config.UserGlobalPrefix+"/") {
		rel := strings.TrimPrefix(sourcePath, config.UserGlobalPrefix+"/")
		return filepath.Join(s.devDir(), rel)
	}
	if config.IsHomeGlobalPath(sourcePath) && !s.Records.GlobalInRepo() {
		rel := strings.TrimPrefix(sourcePath, config.HomeGlobalPrefix+"/")
		return filepath.Join(s.Records.GlobalDir(s.RepoRoot), rel)
	}
	return filepath.Join(s.RepoRoot, sourcePath)
}

func (s *Service) devDir() string {
	return (&config.Config{Records: s.Records}).DevRecordsDir()
}

func slugify(title string) string {
	title = strings.ToLower(title)
	var b strings.Builder
	lastDash := false
	for _, r := range title {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
