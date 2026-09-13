package export

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
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
	SchemaVersion int     `json:"schema_version"`
	ExportedAt    string  `json:"exported_at"`
	Records       []Entry `json:"records"`
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
	recs, err := collectRecords(repo, home)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(opts.OutDir, 0o755); err != nil {
		return err
	}
	if err := writeRecords(recs, opts.OutDir); err != nil {
		return err
	}
	if err := writeIndex(recs, filepath.Join(opts.OutDir, "INDEX.md")); err != nil {
		return err
	}
	if err := writeTypeDigests(recs, opts.OutDir); err != nil {
		return err
	}
	if err := writeMap(repo, filepath.Join(opts.OutDir, "map.md")); err != nil {
		return err
	}
	return writeManifest(recs, filepath.Join(opts.OutDir, "archive.json"))
}

func collectRecords(repo, home *store.Store) ([]*record.Record, error) {
	var all []*record.Record
	for _, st := range []*store.Store{repo, home} {
		if st == nil {
			continue
		}
		recs, err := st.AllRecords()
		if err != nil {
			return nil, err
		}
		all = append(all, recs...)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Type != all[j].Type {
			return all[i].Type < all[j].Type
		}
		return all[i].Title < all[j].Title
	})
	return all, nil
}

func writeRecords(recs []*record.Record, outDir string) error {
	if err := os.RemoveAll(filepath.Join(outDir, "records")); err != nil {
		return err
	}
	for _, r := range recs {
		p := filepath.Join(outDir, filepath.FromSlash(recordRel(r)))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, []byte(record.Serialize(r)), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func writeIndex(recs []*record.Record, dest string) error {
	var b strings.Builder
	b.WriteString("# Archive Index\n\n")
	byType := groupByType(recs)
	var digests []string
	for _, typ := range record.IndexOrder {
		if !wantDigest(typ, byType[typ]) {
			continue
		}
		digests = append(digests, fmt.Sprintf("[%s](%s)", typeHeading(typ), digestFile(typ)))
	}
	if len(digests) > 0 {
		b.WriteString("Type digests: ")
		b.WriteString(strings.Join(digests, " · "))
		b.WriteString("\n\n")
	}
	for _, typ := range record.IndexOrder {
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
			fmt.Fprintf(&b, "- [%s](%s) (%s)%s\n", r.Title, recordRel(r), r.Status, sev)
		}
		b.WriteByte('\n')
	}
	return os.WriteFile(dest, []byte(b.String()), 0o644)
}

func writeTypeDigests(recs []*record.Record, dir string) error {
	byType := groupByType(recs)
	for _, typ := range record.IndexOrder {
		items := byType[typ]
		dest := filepath.Join(dir, digestFile(typ))
		if !wantDigest(typ, items) {
			if err := os.Remove(dest); err != nil && !os.IsNotExist(err) {
				return err
			}
			continue
		}
		if err := writeTypeDigest(typ, items, dest); err != nil {
			return err
		}
	}
	return nil
}

func writeTypeDigest(typ record.Type, items []*record.Record, dest string) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# %s\n\n", typeHeading(typ))
	for _, r := range items {
		fmt.Fprintf(&b, "## %s\n\n", r.Title)
		fmt.Fprintf(&b, "- Status: %s\n", r.Status)
		fmt.Fprintf(&b, "- Scope: %s\n", r.Scope)
		if r.Severity != "" {
			fmt.Fprintf(&b, "- Severity: %s\n", r.Severity)
		}
		if len(r.AppliesTo) > 0 {
			fmt.Fprintf(&b, "- Applies to: %s\n", strings.Join(r.AppliesTo, ", "))
		}
		if len(r.Tags) > 0 {
			fmt.Fprintf(&b, "- Tags: %s\n", strings.Join(r.Tags, ", "))
		}
		b.WriteString("\n")
		b.WriteString(r.Body)
		b.WriteString("\n\n---\n\n")
	}
	return os.WriteFile(dest, []byte(b.String()), 0o644)
}

func groupByType(recs []*record.Record) map[record.Type][]*record.Record {
	byType := map[record.Type][]*record.Record{}
	for _, r := range recs {
		byType[r.Type] = append(byType[r.Type], r)
	}
	return byType
}

func recordRel(r *record.Record) string {
	return path.Join("records", string(r.Scope), string(r.Type), r.Slug+".md")
}

func wantDigest(typ record.Type, items []*record.Record) bool {
	return len(items) > 0 || typ == record.TypeRule
}

func typeHeading(t record.Type) string {
	s := string(t) + "s"
	return strings.ToUpper(s[:1]) + s[1:]
}

// digestFile is the export-root markdown for a type. The plural keeps map-type
// records at maps.md so they do not collide with the code map at map.md.
func digestFile(t record.Type) string {
	return strings.ToLower(typeHeading(t)) + ".md"
}

func writeMap(repo *store.Store, path string) error {
	var b strings.Builder
	b.WriteString("# Code Map\n\n")
	if repo == nil {
		return os.WriteFile(path, []byte(b.String()), 0o644)
	}
	syms, err := repo.AllSymbols()
	if err != nil {
		return err
	}
	current := ""
	for _, s := range syms {
		if s.FilePath != current {
			if current != "" {
				b.WriteByte('\n')
			}
			current = s.FilePath
			fmt.Fprintf(&b, "## %s\n\n", current)
		}
		fmt.Fprintf(&b, "- `%s` (%s) line %d\n", s.Name, s.Kind, s.Line)
	}
	if current != "" {
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
