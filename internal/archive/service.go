// Package archive creates, updates, retires, and imports records.
package archive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
)

type Service struct {
	RepoRoot string
	Records  config.RecordsConfig
	DB       *store.Store
}

func New(repoRoot string, cfg *config.Config, db *store.Store) *Service {
	recs := config.Default().Records
	if cfg != nil {
		recs = cfg.Records
	}
	return &Service{RepoRoot: repoRoot, Records: recs, DB: db}
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
	if rec.Scope == record.ScopeRepo && rec.Checkout == "" {
		if strings.TrimSpace(archiveRoot) == "" {
			rec.Checkout = s.RepoRoot
		} else if root, err := s.matchExtra(archiveRoot); err != nil {
			return "", err
		} else {
			rec.Checkout = root
		}
	}
	if rec.Scope == record.ScopeGlobal || rec.Scope == record.ScopeDev {
		rec.Checkout = ""
	}
	if err := rec.Validate(); err != nil {
		return "", err
	}

	st, err := s.contextStore(rec.Scope, archiveRoot)
	if err != nil {
		return "", err
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

// contextStore is the one open database. Scope and archiveRoot choose the
// checkout column, which Remember sets before this runs. A listed extra is
// merged on that same connection.
func (s *Service) contextStore(scope record.Scope, archiveRoot string) (*store.Store, error) {
	if s.DB == nil {
		return nil, store.NeedsInit(config.HomeStorePath(), os.ErrNotExist)
	}
	if scope == record.ScopeGlobal || scope == record.ScopeDev {
		if strings.TrimSpace(archiveRoot) != "" {
			return nil, errors.New("global and dev records do not take a checkout")
		}
		return s.DB, nil
	}
	if strings.TrimSpace(archiveRoot) != "" {
		root, err := s.matchExtra(archiveRoot)
		if err != nil {
			return nil, err
		}
		if err := s.DB.MergeCheckout(root); err != nil {
			return nil, err
		}
	}
	return s.DB, nil
}

func (s *Service) matchExtra(archiveRoot string) (string, error) {
	canon, err := config.ResolveCheckout(s.RepoRoot, archiveRoot)
	if err != nil {
		return "", err
	}
	if s.DB == nil {
		return "", fmt.Errorf("checkout %s is not registered; run archivist init", canon)
	}
	ok, err := s.DB.HasCheckout(canon)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("checkout %s is not registered; run archivist init", canon)
	}
	return canon, nil
}

func (s *Service) Update(id string, fn func(*record.Record) error) error {
	rec, st, ok, err := s.findPrimary(id)
	if err != nil {
		return err
	}
	if ok {
		return s.writeRecord(st, rec, fn)
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

// Origin says which database satisfied a lookup.
type Origin string

const (
	OriginRepo  Origin = "repo"
	OriginOther Origin = "other"
)

func (s *Service) Get(idOrSlug string) (*record.Record, error) {
	rec, _, err := s.Lookup(idOrSlug)
	return rec, err
}

// Lookup finds a record in the open database, then a checkout named by a
// link. A link target is one hop.
func (s *Service) Lookup(idOrSlug string) (*record.Record, Origin, error) {
	rec, _, ok, err := s.findPrimary(idOrSlug)
	if err != nil {
		return nil, "", err
	}
	if ok {
		return rec, s.origin(rec), nil
	}
	got, err := s.recordFromLinks(idOrSlug)
	if err != nil {
		return nil, "", err
	}
	if got != nil {
		return got, OriginOther, nil
	}
	return nil, "", fmt.Errorf("record not found: %s", idOrSlug)
}

func (s *Service) origin(rec *record.Record) Origin {
	if rec == nil || rec.Checkout == "" {
		return OriginOther
	}
	root := s.RepoRoot
	if canon, err := config.Canonical(s.RepoRoot); err == nil {
		root = canon
	}
	if rec.Checkout == root || rec.Checkout == s.RepoRoot {
		return OriginRepo
	}
	return OriginOther
}

func (s *Service) recordFromLinks(idOrSlug string) (*record.Record, error) {
	if s.DB == nil {
		return nil, nil
	}
	edges, err := s.DB.ListLinks()
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	for _, edge := range edges {
		if edge.ToRoot == "" {
			continue
		}
		if _, ok := seen[edge.ToRoot]; ok {
			continue
		}
		seen[edge.ToRoot] = struct{}{}
		path := config.LegacyIndexPath(edge.ToRoot)
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		db, err := store.OpenReadOnly(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		got, found, err := db.GetRecordByID(idOrSlug)
		if err != nil {
			_ = db.Close()
			return nil, err
		}
		if !found {
			got, found, err = db.GetRecordBySlug(idOrSlug)
			if err != nil {
				_ = db.Close()
				return nil, err
			}
		}
		_ = db.Close()
		if found {
			return got, nil
		}
	}
	return nil, nil
}

func (s *Service) findPrimary(idOrSlug string) (*record.Record, *store.Store, bool, error) {
	if s.DB == nil {
		return nil, nil, false, store.NeedsInit(config.HomeStorePath(), os.ErrNotExist)
	}
	rec, ok, err := s.DB.GetRecordByID(idOrSlug)
	if err != nil {
		return nil, nil, false, err
	}
	if ok {
		return rec, s.DB, true, nil
	}
	rec, ok, err = s.DB.GetRecordBySlug(idOrSlug)
	if err != nil {
		return nil, nil, false, err
	}
	if ok {
		return rec, s.DB, true, nil
	}
	return nil, nil, false, nil
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

func PatchText(r *record.Record, title, body, status string, appliesTo []string) {
	if title != "" {
		r.Title = title
	}
	if body != "" {
		r.Body = body
	}
	if status != "" {
		r.Status = record.Status(status)
	}
	if len(appliesTo) > 0 {
		r.AppliesTo = append([]string(nil), appliesTo...)
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
