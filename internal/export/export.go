package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
	"github.com/bwireman/archivist/internal/version"
)

type Options struct {
	RepoRoot string
	OutDir   string
}

type Manifest struct {
	SchemaVersion int       `json:"schema_version"`
	ExportedAt    string    `json:"exported_at"`
	Records       []Entry   `json:"records"`
}

type Entry struct {
	ID          string   `json:"id"`
	Slug        string   `json:"slug"`
	Type        string   `json:"type"`
	Scope       string   `json:"scope"`
	Title       string   `json:"title"`
	Status      string   `json:"status"`
	Severity    string   `json:"severity,omitempty"`
	SourcePath  string   `json:"source_path"`
	Tags        []string `json:"tags,omitempty"`
	AppliesTo   []string `json:"applies_to,omitempty"`
	ContentHash string   `json:"content_hash"`
}

func Run(repo *store.Store, home *store.Store, opts Options) error {
	if opts.OutDir == "" {
		opts.OutDir = filepath.Join(opts.RepoRoot, config.DefaultArchiveDir)
	}
	recs := collectRecords(repo, home)
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return err
	}
	recDir := filepath.Join(opts.OutDir, "records")
	if err := writeRecords(recs, recDir); err != nil {
		return err
	}
	if err := writeIndex(recs, filepath.Join(opts.OutDir, "INDEX.md")); err != nil {
		return err
	}
	if err := writeRules(recs, filepath.Join(opts.OutDir, "rules.md")); err != nil {
		return err
	}
	if err := writeMap(repo, filepath.Join(opts.OutDir, "map.md")); err != nil {
		return err
	}
	return writeManifest(recs, filepath.Join(opts.OutDir, "archive.json"))
}

func collectRecords(repo, home *store.Store) []*record.Record {
	var all []*record.Record
	for _, st := range []*store.Store{repo, home} {
		if st == nil {
			continue
		}
		recs, err := st.AllRecords()
		if err == nil {
			all = append(all, recs...)
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Type != all[j].Type {
			return all[i].Type < all[j].Type
		}
		return all[i].Title < all[j].Title
	})
	return all
}

func writeRecords(recs []*record.Record, dir string) error {
	for _, r := range recs {
		path := filepath.Join(dir, string(r.Scope), r.Slug+".md")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(record.Serialize(r)), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func writeIndex(recs []*record.Record, path string) error {
	var b strings.Builder
	b.WriteString("# Archive Index\n\n")
	byType := map[record.Type][]*record.Record{}
	for _, r := range recs {
		byType[r.Type] = append(byType[r.Type], r)
	}
	for _, typ := range []record.Type{record.TypeRule, record.TypeDecision, record.TypeGuide, record.TypeMap, record.TypePitfall} {
		items := byType[typ]
		if len(items) == 0 {
			continue
		}
		fmt.Fprintf(&b, "## %s\n\n", typ)
		for _, r := range items {
			sev := ""
			if r.Severity != "" {
				sev = fmt.Sprintf(" [%s]", r.Severity)
			}
			fmt.Fprintf(&b, "- [%s](records/%s/%s.md) (%s)%s\n", r.Title, r.Scope, r.Slug, r.Status, sev)
		}
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeRules(recs []*record.Record, path string) error {
	var b strings.Builder
	b.WriteString("# Rules\n\n")
	for _, r := range recs {
		if r.Type != record.TypeRule {
			continue
		}
		fmt.Fprintf(&b, "## %s\n\n", r.Title)
		if r.Severity != "" {
			fmt.Fprintf(&b, "- Severity: %s\n", r.Severity)
		}
		if len(r.AppliesTo) > 0 {
			fmt.Fprintf(&b, "- Applies to: %s\n", strings.Join(r.AppliesTo, ", "))
		}
		b.WriteString("\n")
		b.WriteString(r.Body)
		b.WriteString("\n\n---\n\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeMap(repo *store.Store, path string) error {
	var b strings.Builder
	b.WriteString("# Code Map\n\n")
	if repo == nil {
		return os.WriteFile(path, []byte(b.String()), 0o644)
	}
	files, err := repo.AllFilePaths()
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, f := range files {
		syms, err := repo.SymbolsForFile(f)
		if err != nil || len(syms) == 0 {
			continue
		}
		fmt.Fprintf(&b, "## %s\n\n", f)
		for _, s := range syms {
			fmt.Fprintf(&b, "- `%s` (%s) line %d\n", s.Name, s.Kind, s.Line)
		}
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeManifest(recs []*record.Record, path string) error {
	m := Manifest{
		SchemaVersion: version.Schema,
		ExportedAt:    time.Now().UTC().Format(time.RFC3339),
	}
	for _, r := range recs {
		m.Records = append(m.Records, Entry{
			ID: r.ID, Slug: r.Slug, Type: string(r.Type), Scope: string(r.Scope),
			Title: r.Title, Status: string(r.Status), Severity: string(r.Severity),
			SourcePath: r.SourcePath, Tags: r.Tags, AppliesTo: r.AppliesTo, ContentHash: r.ContentHash,
		})
	}
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// WriteBundle writes a portable tar-less directory bundle.
func WriteBundle(repo *store.Store, home *store.Store, bundlePath string) error {
	return Run(repo, home, Options{OutDir: bundlePath})
}
