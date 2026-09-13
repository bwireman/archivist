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
	if err := writeTypeDigests(recs, opts.OutDir); err != nil {
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
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	for _, r := range recs {
		p := filepath.Join(dir, string(r.Scope), string(r.Type), r.Slug+".md")
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
		if len(byType[typ]) == 0 && typ != record.TypeRule {
			continue
		}
		digests = append(digests, fmt.Sprintf("[%s](%s)", typeHeading(typ), DigestFile(typ)))
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
		dest := filepath.Join(dir, DigestFile(typ))
		if len(items) == 0 {
			if typ == record.TypeRule {
				if err := writeTypeDigest(typ, nil, dest); err != nil {
					return err
				}
				continue
			}
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

// DigestFile is the export-root markdown file for a record type.
// Map-type records use maps.md so they do not collide with the code map at map.md.
func DigestFile(t record.Type) string {
	if t == record.TypeMap {
		return "maps.md"
	}
	return string(t) + "s.md"
}

func typeHeading(t record.Type) string {
	switch t {
	case record.TypeRule:
		return "Rules"
	case record.TypeDecision:
		return "Decisions"
	case record.TypeFeature:
		return "Features"
	case record.TypeGuide:
		return "Guides"
	case record.TypeMap:
		return "Maps"
	case record.TypePitfall:
		return "Pitfalls"
	default:
		return string(t)
	}
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
