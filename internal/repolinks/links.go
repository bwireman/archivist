// Package repolinks stores typed checkout edges in ~/.archivist/repo-links.json.
package repolinks

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
)

const (
	KindRelated   = "related"
	KindDependsOn = "depends-on"
)

// RecordRef names a record in a checkout database. An empty Root is the home store.
type RecordRef struct {
	Root string `json:"root,omitempty"`
	ID   string `json:"id"`
}

// Edge is one stored connection. Identity is the kind plus the endpoints.
// Records is replaced wholesale by a later link of the same identity.
type Edge struct {
	Kind    string      `json:"kind"`
	A       string      `json:"a,omitempty"`
	B       string      `json:"b,omitempty"`
	From    string      `json:"from,omitempty"`
	To      string      `json:"to,omitempty"`
	Records []RecordRef `json:"records,omitempty"`
}

type file struct {
	Links []Edge `json:"links"`
}

// SameEdge reports whether a and b are the same kind and endpoints.
// Records are not part of identity.
func SameEdge(a, b Edge) bool {
	return a.Kind == b.Kind && a.A == b.A && a.B == b.B && a.From == b.From && a.To == b.To
}

// Normalize builds an edge from two canonical checkout paths.
// related stores one object with A <= B. depends-on stores from -> to.
func Normalize(kind, from, to string, records []RecordRef) (Edge, error) {
	kind = strings.TrimSpace(kind)
	if from == "" || to == "" {
		return Edge{}, errors.New("repo links: checkout path is empty")
	}
	if from == to {
		return Edge{}, errors.New("repo links: checkout cannot link to itself")
	}
	if len(records) == 0 {
		records = nil
	}
	switch kind {
	case KindRelated:
		if to < from {
			from, to = to, from
		}
		return Edge{Kind: kind, A: from, B: to, Records: records}, nil
	case KindDependsOn:
		return Edge{Kind: kind, From: from, To: to, Records: records}, nil
	default:
		return Edge{}, fmt.Errorf("repo links: unknown kind %q", kind)
	}
}

// ParseRecord splits home#id or checkout#id on the last '#'.
// home stores an empty Root. A checkout path is resolved against primary.
func ParseRecord(primary, raw string) (RecordRef, error) {
	raw = strings.TrimSpace(raw)
	i := strings.LastIndex(raw, "#")
	if i <= 0 || i == len(raw)-1 {
		return RecordRef{}, fmt.Errorf("repo links: record %q must be home#id or checkout#id", raw)
	}
	left, id := strings.TrimSpace(raw[:i]), strings.TrimSpace(raw[i+1:])
	if id == "" {
		return RecordRef{}, fmt.Errorf("repo links: record %q must be home#id or checkout#id", raw)
	}
	if left == "home" {
		return RecordRef{ID: id}, nil
	}
	root, err := config.ResolveCheckout(primary, left)
	if err != nil {
		return RecordRef{}, err
	}
	return RecordRef{Root: root, ID: id}, nil
}

// Load reads edges. A missing file is an empty list.
// Invalid JSON returns the parse error.
func Load(path string) ([]Edge, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("repo links: %s: %w", path, err)
	}
	return f.Links, nil
}

// List returns every edge, sorted by kind then paths.
func List(path string) ([]Edge, error) {
	edges, err := Load(path)
	if err != nil {
		return nil, err
	}
	sortEdges(edges)
	return edges, nil
}

// Link records an edge from primary to other. primary and other are checkout paths.
// A second call with the same kind and endpoints replaces records.
func Link(path, primary, kind, other string, records []RecordRef) (Edge, error) {
	edges, err := Load(path)
	if err != nil {
		return Edge{}, err
	}
	from, err := resolveEnd(primary, primary)
	if err != nil {
		return Edge{}, err
	}
	to, err := resolveEnd(primary, other)
	if err != nil {
		return Edge{}, err
	}
	records, err = prepareRecords(from, records)
	if err != nil {
		return Edge{}, err
	}
	edge, err := Normalize(kind, from, to, records)
	if err != nil {
		return Edge{}, err
	}
	replaced := false
	for i := range edges {
		if SameEdge(edges[i], edge) {
			edges[i] = edge
			replaced = true
			break
		}
	}
	if !replaced {
		edges = append(edges, edge)
	}
	if err := save(path, edges); err != nil {
		return Edge{}, err
	}
	return edge, nil
}

// Unlink removes the edge from primary to other. Either argument order matches
// a related edge. The reverse depends-on edge is left in place.
// A missing file or a missing edge succeeds and does not write.
func Unlink(path, primary, kind, other string) error {
	edges, err := Load(path)
	if err != nil {
		return err
	}
	from, err := config.ResolveCheckout(primary, primary)
	if err != nil {
		return err
	}
	to, err := config.ResolveCheckout(from, other)
	if err != nil {
		return err
	}
	want, err := Normalize(kind, from, to, nil)
	if err != nil {
		return err
	}
	next := edges[:0]
	removed := false
	for _, edge := range edges {
		if SameEdge(edge, want) {
			removed = true
			continue
		}
		next = append(next, edge)
	}
	if !removed {
		return nil
	}
	return save(path, next)
}

func resolveEnd(primary, raw string) (string, error) {
	root, err := config.ResolveCheckout(primary, raw)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("repo links: %s: %w", root, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("repo links: %s is not a directory", root)
	}
	db := config.StorePath(root)
	st, err := os.Stat(db)
	if err != nil || st.IsDir() {
		return "", fmt.Errorf("repo links: %s has no .archivist/index.db", root)
	}
	return root, nil
}

func prepareRecords(primary string, records []RecordRef) ([]RecordRef, error) {
	if len(records) == 0 {
		return nil, nil
	}
	seen := map[string]struct{}{}
	out := make([]RecordRef, len(records))
	for i, ref := range records {
		ref.ID = strings.TrimSpace(ref.ID)
		if ref.ID == "" {
			return nil, errors.New("repo links: record id is empty")
		}
		if ref.Root != "" {
			root, err := config.ResolveCheckout(primary, ref.Root)
			if err != nil {
				return nil, err
			}
			ref.Root = root
		}
		key := ref.Root + "\x00" + ref.ID
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("repo links: duplicate record %s", ref.ID)
		}
		seen[key] = struct{}{}
		out[i] = ref
	}
	if err := recordsExist(out); err != nil {
		return nil, err
	}
	slices.SortFunc(out, func(a, b RecordRef) int {
		if c := cmp.Compare(a.Root, b.Root); c != 0 {
			return c
		}
		return cmp.Compare(a.ID, b.ID)
	})
	return out, nil
}

func recordsExist(refs []RecordRef) error {
	ids := map[string][]string{}
	var order []string
	for _, ref := range refs {
		dbPath := config.HomeStorePath()
		if ref.Root != "" {
			dbPath = config.StorePath(ref.Root)
		}
		if dbPath == "" {
			return errors.New("repo links: cannot resolve home archive path (set $HOME)")
		}
		if _, ok := ids[dbPath]; !ok {
			order = append(order, dbPath)
		}
		ids[dbPath] = append(ids[dbPath], ref.ID)
	}
	for _, dbPath := range order {
		if err := recordsInStore(dbPath, ids[dbPath]); err != nil {
			return err
		}
	}
	return nil
}

func recordsInStore(dbPath string, ids []string) error {
	db, err := store.OpenReadOnly(dbPath)
	if err != nil {
		return fmt.Errorf("repo links: %w", err)
	}
	defer db.Close()
	got, err := db.GetRecordsByIDs(ids)
	if err != nil {
		return err
	}
	for _, id := range ids {
		rec, ok := got[id]
		if !ok {
			return fmt.Errorf("repo links: record %s not found", id)
		}
		if !record.ValidType(rec.Type) {
			return fmt.Errorf("repo links: record %s has type %s", id, rec.Type)
		}
	}
	return nil
}

func save(path string, edges []Edge) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	sortEdges(edges)
	body, err := json.MarshalIndent(file{Links: edges}, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	tmp := filepath.Join(dir, "repo-links.json.tmp")
	if err := os.WriteFile(tmp, body, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func sortEdges(edges []Edge) {
	slices.SortFunc(edges, func(a, b Edge) int {
		if c := cmp.Compare(a.Kind, b.Kind); c != 0 {
			return c
		}
		if c := cmp.Compare(a.A, b.A); c != 0 {
			return c
		}
		if c := cmp.Compare(a.B, b.B); c != 0 {
			return c
		}
		if c := cmp.Compare(a.From, b.From); c != 0 {
			return c
		}
		return cmp.Compare(a.To, b.To)
	})
}
