// Package archive creates, updates, retires, and imports records.
package archive

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
)

// Extra is a listed checkout. DB is the read-only open used by Get.
// Writes use OpenExisting on Root and do not use DB.
type Extra struct {
	Root string
	DB   *store.Store
}

type Service struct {
	RepoRoot   string
	Records    config.RecordsConfig
	RepoDB     *store.Store
	HomeDB     *store.Store
	Extras     []Extra
	ExtraRoots []string
}

func New(repoRoot string, cfg *config.Config, repoDB, homeDB *store.Store) *Service {
	recs := config.Default().Records
	if cfg != nil {
		recs = cfg.Records
	}
	return &Service{RepoRoot: repoRoot, Records: recs, RepoDB: repoDB, HomeDB: homeDB}
}

func (s *Service) Remember(rec *record.Record, archiveRoot string) (string, error) {
	if rec.Slug == "" {
		rec.Slug = slugify(rec.Title)
	}
	if rec.SourcePath == "" {
		rec.SourcePath = s.defaultPath(rec)
	}
	if rec.ID == "" {
		rec.ID = record.NewID()
	}
	if err := rec.Validate(); err != nil {
		return "", err
	}

	// Context picks one store. Never upsert the same row into a second store.
	st, closeFn, err := s.contextStore(rec.Scope, archiveRoot)
	if err != nil {
		return "", err
	}
	if closeFn != nil {
		defer closeFn()
	}
	if st == nil {
		return "", fmt.Errorf("no store for scope %s", rec.Scope)
	}
	// UpsertRecord adopts the id and created_at of any row already at this
	// source_path, so remembering the same topic twice updates in place.
	if err := st.UpsertRecord(rec); err != nil {
		return "", err
	}
	return rec.ID, nil
}

func (s *Service) contextStore(scope record.Scope, archiveRoot string) (*store.Store, func(), error) {
	if scope == record.ScopeGlobal || scope == record.ScopeDev {
		if strings.TrimSpace(archiveRoot) != "" {
			return nil, nil, errors.New("global and dev records are written to the home store")
		}
		return s.HomeDB, nil, nil
	}
	if strings.TrimSpace(archiveRoot) == "" {
		return s.RepoDB, nil, nil
	}
	root, err := s.matchExtra(archiveRoot)
	if err != nil {
		return nil, nil, err
	}
	db, err := store.OpenExisting(config.StorePath(root))
	if err != nil {
		return nil, nil, err
	}
	return db, func() { _ = db.Close() }, nil
}

func (s *Service) matchExtra(archiveRoot string) (string, error) {
	canon, err := config.ResolveCheckout(s.RepoRoot, archiveRoot)
	if err != nil {
		return "", err
	}
	for _, root := range s.ExtraRoots {
		if root == canon {
			return root, nil
		}
	}
	return "", fmt.Errorf("archive %s is not a configured extra", canon)
}

func (s *Service) Update(id string, fn func(*record.Record) error) error {
	rec, st, ok, err := s.findPrimary(id)
	if err != nil {
		return err
	}
	if ok {
		return s.writeRecord(st, rec, fn)
	}
	for _, root := range s.ExtraRoots {
		db, err := store.OpenExisting(config.StorePath(root))
		if err != nil {
			return err
		}
		got, found, err := db.GetRecordByID(id)
		if err != nil {
			_ = db.Close()
			return err
		}
		if !found {
			_ = db.Close()
			continue
		}
		err = s.writeRecord(db, got, fn)
		_ = db.Close()
		return err
	}
	return fmt.Errorf("record not found: %s", id)
}

func (s *Service) writeRecord(st *store.Store, rec *record.Record, fn func(*record.Record) error) error {
	if err := fn(rec); err != nil {
		return err
	}
	rec.ContentHash = record.ContentHash(rec)
	if err := rec.Validate(); err != nil {
		return err
	}
	return st.UpsertRecord(rec)
}

func (s *Service) Retire(id, supersededBy string) error {
	return s.Update(id, func(r *record.Record) error {
		r.Status = record.StatusSuperseded
		r.SupersededBy = supersededBy
		return nil
	})
}

func (s *Service) Get(idOrSlug string) (*record.Record, error) {
	rec, _, ok, err := s.findPrimary(idOrSlug)
	if err != nil {
		return nil, err
	}
	if ok {
		return rec, nil
	}
	for _, ex := range s.Extras {
		if ex.DB == nil {
			continue
		}
		got, found, err := ex.DB.GetRecordByID(idOrSlug)
		if err != nil {
			return nil, err
		}
		if found {
			return got, nil
		}
	}
	return nil, fmt.Errorf("record not found: %s", idOrSlug)
}

func (s *Service) findPrimary(idOrSlug string) (*record.Record, *store.Store, bool, error) {
	for _, st := range []*store.Store{s.RepoDB, s.HomeDB} {
		if st == nil {
			continue
		}
		rec, ok, err := st.GetRecordByID(idOrSlug)
		if err != nil {
			return nil, nil, false, err
		}
		if ok {
			return rec, st, true, nil
		}
		rec, ok, err = st.GetRecordBySlug(idOrSlug)
		if err != nil {
			return nil, nil, false, err
		}
		if ok {
			return rec, st, true, nil
		}
	}
	return nil, nil, false, nil
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
	name := rec.Slug + ".md"
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

func (s *Service) devDir() string {
	return (&config.Config{Records: s.Records}).DevRecordsDir()
}

func PatchText(r *record.Record, title, body, status string) {
	if title != "" {
		r.Title = title
	}
	if body != "" {
		r.Body = body
	}
	if status != "" {
		r.Status = record.Status(status)
	}
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
