package store

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
)

const (
	LinkRecord = "record"
	LinkPin    = "pin"
	LinkRepo   = "repo"
)

// Link is one row in record_links. Empty ends are stored as "".
type Link struct {
	Kind   string `json:"kind"`
	FromID string `json:"from_id"`
	ToRoot string `json:"to_root"`
	ToID   string `json:"to_id"`
}

// ListLinks returns every edge, sorted by kind then paths. A database whose
// schema has no record_links table returns an empty list.
func (s *Store) ListLinks() ([]Link, error) {
	ok, err := s.hasRecordLinks()
	if err != nil {
		return nil, err
	}
	if !ok {
		return []Link{}, nil
	}
	rows, err := s.db.Query(`SELECT kind, from_id, to_root, to_id FROM record_links ORDER BY kind, from_id, to_root, to_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Link{}
	for rows.Next() {
		var link Link
		if err := rows.Scan(&link.Kind, &link.FromID, &link.ToRoot, &link.ToID); err != nil {
			return nil, err
		}
		out = append(out, link)
	}
	return out, rows.Err()
}

// InsertLink validates the edge and inserts it. A duplicate primary key is
// success and does not change the stored row.
func (s *Store) InsertLink(primary string, link Link) error {
	norm, err := normalizeLink(primary, link)
	if err != nil {
		return err
	}
	if err := requireCheckout(norm.ToRoot); err != nil {
		return err
	}
	if err := s.localRecordExists(norm); err != nil {
		return err
	}
	if err := targetRecordExists(norm); err != nil {
		return err
	}
	_, err = s.db.Exec(`INSERT OR IGNORE INTO record_links (kind, from_id, to_root, to_id) VALUES (?, ?, ?, ?)`,
		norm.Kind, norm.FromID, norm.ToRoot, norm.ToID)
	return err
}

// DeleteLink removes one edge. A missing row is success. The caller opens the
// store with Open, which creates the table.
func (s *Store) DeleteLink(primary string, link Link) error {
	norm, err := normalizeLink(primary, link)
	if err != nil {
		return err
	}
	if err := requireCheckout(norm.ToRoot); err != nil {
		return err
	}
	_, err = s.db.Exec(`DELETE FROM record_links WHERE kind = ? AND from_id = ? AND to_root = ? AND to_id = ?`,
		norm.Kind, norm.FromID, norm.ToRoot, norm.ToID)
	return err
}

// NewLink resolves a CLI or MCP link request against primary.
func NewLink(primary, kind, from, to string) (Link, error) {
	rawRoot, id, err := ParseLinkTo(kind, to)
	if err != nil {
		return Link{}, err
	}
	resolved, err := config.ResolveCheckout(primary, rawRoot)
	if err != nil {
		return Link{}, err
	}
	return Link{
		Kind:   strings.TrimSpace(kind),
		FromID: strings.TrimSpace(from),
		ToRoot: resolved,
		ToID:   id,
	}, nil
}

// ParseLinkTo splits the CLI and MCP `to` value. repo keeps the checkout
// path. record and pin split checkout#id on the last '#'.
func ParseLinkTo(kind, to string) (root, id string, err error) {
	kind = strings.TrimSpace(kind)
	to = strings.TrimSpace(to)
	switch kind {
	case LinkRepo:
		if to == "" {
			return "", "", errors.New("record links: checkout path is empty")
		}
		return to, "", nil
	case LinkRecord, LinkPin:
		i := strings.LastIndex(to, "#")
		if i <= 0 || i == len(to)-1 {
			return "", "", fmt.Errorf("record links: %q must be checkout#id", to)
		}
		root, id = strings.TrimSpace(to[:i]), strings.TrimSpace(to[i+1:])
		if root == "" || id == "" {
			return "", "", fmt.Errorf("record links: %q must be checkout#id", to)
		}
		return root, id, nil
	default:
		return "", "", fmt.Errorf("record links: unknown kind %q", kind)
	}
}

func normalizeLink(primary string, link Link) (Link, error) {
	link.Kind = strings.TrimSpace(link.Kind)
	link.FromID = strings.TrimSpace(link.FromID)
	link.ToID = strings.TrimSpace(link.ToID)
	link.ToRoot = strings.TrimSpace(link.ToRoot)
	switch link.Kind {
	case LinkPin:
		if link.FromID != "" {
			return Link{}, errors.New("record links: pin does not take from_id")
		}
		if link.ToID == "" {
			return Link{}, errors.New("record links: record id is empty")
		}
	case LinkRepo:
		if link.FromID == "" {
			return Link{}, errors.New("record links: from_id is required")
		}
		if link.ToID != "" {
			return Link{}, errors.New("record links: repo does not take a record id")
		}
	case LinkRecord:
		if link.FromID == "" {
			return Link{}, errors.New("record links: from_id is required")
		}
		if link.ToID == "" {
			return Link{}, errors.New("record links: record id is empty")
		}
	default:
		return Link{}, fmt.Errorf("record links: unknown kind %q", link.Kind)
	}
	if link.ToRoot == "" {
		return Link{}, errors.New("record links: checkout path is empty")
	}
	from, err := config.Canonical(primary)
	if err != nil {
		return Link{}, err
	}
	to, err := config.Canonical(link.ToRoot)
	if err != nil {
		return Link{}, err
	}
	if from == to {
		return Link{}, errors.New("record links: checkout cannot link to itself")
	}
	link.ToRoot = to
	return link, nil
}

func requireCheckout(root string) error {
	info, err := os.Stat(root)
	if err != nil {
		return fmt.Errorf("record links: %s: %w", root, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("record links: %s is not a directory", root)
	}
	db := config.StorePath(root)
	st, err := os.Stat(db)
	if err != nil {
		return fmt.Errorf("record links: %s: %w", db, err)
	}
	if st.IsDir() {
		return fmt.Errorf("record links: %s has no .archivist/index.db", root)
	}
	return nil
}

func (s *Store) localRecordExists(link Link) error {
	if link.Kind == LinkPin {
		return nil
	}
	_, ok, err := s.GetRecordByID(link.FromID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("record links: record %s not found", link.FromID)
	}
	return nil
}

func targetRecordExists(link Link) error {
	if link.Kind == LinkRepo {
		return nil
	}
	db, err := OpenReadOnly(config.StorePath(link.ToRoot))
	if err != nil {
		return fmt.Errorf("record links: %w", err)
	}
	defer db.Close()
	rec, ok, err := db.GetRecordByID(link.ToID)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("record links: record %s not found", link.ToID)
	}
	if !record.ValidType(rec.Type) {
		return fmt.Errorf("record links: record %s has type %s", link.ToID, rec.Type)
	}
	return nil
}

func (s *Store) hasRecordLinks() (bool, error) {
	return s.tableExists("record_links")
}
