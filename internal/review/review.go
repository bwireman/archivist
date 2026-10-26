// Package review lists evidence for an occasional archive pass.
// It does not update, retire, or rank records.
package review

import (
	"cmp"
	"fmt"
	"math"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/cmdlog"
	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
	"github.com/bwireman/archivist/internal/trace"
)

// Interval is how long after a full review the next one is due.
// It matches the default trace window: the last 7 days of work.
const Interval = 7 * 24 * time.Hour

// Measured 2026-10-26 on this product's qwen3-embedding:8b doc-doc pairs.
// At 0.80 the pairs were a feature and the decision or rule on the same topic (9 pairs).
// The next lower band, 0.75, mixed related but distinct topics.
// Hybrid search does not use this floor. Not a config key.
const similarMinCosine = 0.80

const (
	maxSimilar      = 20
	maxFeatureFiles = 4
)

// DueReport is the clock. A due check does not count as a review.
type DueReport struct {
	Due        bool      `json:"due"`
	LoggingOff bool      `json:"logging_off,omitempty"`
	LastReview time.Time `json:"last_review,omitzero"`
}

// Card is one record in a review list.
type Card struct {
	ID        string   `json:"id"`
	Title     string   `json:"title,omitempty"`
	Type      string   `json:"type,omitempty"`
	Scope     string   `json:"scope,omitempty"`
	AppliesTo []string `json:"applies_to,omitempty"`
}

// Pair is two current records close enough to read for a disagreement.
type Pair struct {
	Cosine float64 `json:"cosine"`
	A      Card    `json:"a"`
	B      Card    `json:"b"`
}

// Feature is a current feature whose applies_to matches files in this checkout.
type Feature struct {
	ID           string   `json:"id"`
	Title        string   `json:"title,omitempty"`
	Scope        string   `json:"scope,omitempty"`
	AppliesTo    []string `json:"applies_to,omitempty"`
	Changed      bool     `json:"changed,omitempty"`
	Files        []string `json:"files,omitempty"`
	FilesOmitted int      `json:"files_omitted,omitempty"`
}

// Report is the evidence list. It is not a ranking and it writes nothing.
type Report struct {
	LoggingOff     bool      `json:"logging_off,omitempty"`
	Since          time.Time `json:"since,omitzero"`
	GitSkipped     string    `json:"git_skipped,omitempty"`
	GitNote        string    `json:"git_note,omitempty"`
	WideDiff       bool      `json:"wide_diff,omitempty"`
	ChangedPaths   int       `json:"changed_paths"`
	MissedRules    []Card    `json:"missed_rules,omitempty"`
	StaleAppliesTo []Card    `json:"stale_applies_to,omitempty"`
	Similar        []Pair    `json:"similar,omitempty"`
	SimilarOmitted int       `json:"similar_omitted,omitempty"`
	Features       []Feature `json:"features,omitempty"`
	Unembedded     int       `json:"unembedded,omitempty"`
}

// Input is the evidence Build reads. Paths are ignored when GitSkipped is set.
type Input struct {
	Since      time.Time
	Entries    []cmdlog.Entry
	LoggingOff bool
	GitSkipped string
	GitNote    string
	Paths      []string
	Files      []string
	Dirs       []string // top-level directories in the checkout
	Records    []*record.Record
	Vectors    map[string][]float32
}

// Clock reports whether a full review is due. log_commands off has no clock.
func Clock(root string, cfg *config.Config, now time.Time) (DueReport, error) {
	entries, off, err := readLog(root, cfg)
	if err != nil {
		return DueReport{}, err
	}
	if off {
		return DueReport{LoggingOff: true}, nil
	}
	last := LastReview(entries)
	return DueReport{Due: due(last, now), LastReview: last}, nil
}

// Collect reads the checkout and home stores and builds a report.
// repo may be nil only when the caller has already failed open.
func Collect(root string, cfg *config.Config, repo, home *store.Store, now time.Time) (*Report, error) {
	if repo == nil {
		return nil, fmt.Errorf("review: checkout database is not open")
	}
	if cfg == nil {
		cfg = config.Default()
	}
	since := now.Add(-Interval)
	entries, off, err := readLog(root, cfg)
	if err != nil {
		return nil, err
	}
	join, err := trace.GitChanged(root, since.UTC().Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	files, err := ListFiles(root, cfg)
	if err != nil {
		return nil, err
	}
	dirs, err := topDirs(root)
	if err != nil {
		return nil, err
	}
	recs, err := loadRecords(repo, home)
	if err != nil {
		return nil, err
	}
	vecs, err := loadVectors(cfg.Ollama.EmbedModel, repo, home)
	if err != nil {
		return nil, err
	}
	return Build(Input{
		Since:      since,
		Entries:    entries,
		LoggingOff: off,
		GitSkipped: join.Skipped,
		GitNote:    join.Note,
		Paths:      join.Paths,
		Files:      files,
		Dirs:       dirs,
		Records:    recs,
		Vectors:    vecs,
	})
}

// Build assembles a report from already loaded evidence.
func Build(in Input) (*Report, error) {
	rep := &Report{
		LoggingOff: in.LoggingOff,
		Since:      in.Since,
		GitSkipped: in.GitSkipped,
		GitNote:    in.GitNote,
	}
	paths := in.Paths
	if in.GitSkipped != "" {
		paths = nil
	} else {
		rep.ChangedPaths = len(paths)
	}
	current := currentRecords(in.Records)
	byID := map[string]*record.Record{}
	for _, rec := range current {
		byID[rec.ID] = rec
	}
	if !in.LoggingOff && in.GitSkipped == "" {
		missed, wide, err := missedRules(in.Entries, in.Since, paths, current)
		if err != nil {
			return nil, err
		}
		rep.WideDiff = wide
		for _, hit := range missed {
			rec := byID[hit.ID]
			card := Card{ID: hit.ID, Title: hit.Title, Type: string(record.TypeRule), AppliesTo: hit.AppliesTo}
			if rec != nil {
				card = cardFrom(rec)
			}
			rep.MissedRules = append(rep.MissedRules, card)
		}
	}
	dirs := map[string]struct{}{}
	for _, d := range in.Dirs {
		dirs[d] = struct{}{}
	}
	for _, rec := range current {
		if _, ok := in.Vectors[rec.ID]; !ok {
			rep.Unembedded++
		}
		if staleApplies(rec, in.Files, dirs) {
			rep.StaleAppliesTo = append(rep.StaleAppliesTo, cardFrom(rec))
		}
		if rec.Type != record.TypeFeature || len(rec.AppliesTo) == 0 || !rec.MatchesPaths(in.Files) {
			continue
		}
		files, omitted := matchingFiles(rec, in.Files, maxFeatureFiles)
		rep.Features = append(rep.Features, Feature{
			ID:           rec.ID,
			Title:        rec.Title,
			Scope:        string(rec.Scope),
			AppliesTo:    append([]string(nil), rec.AppliesTo...),
			Changed:      rec.MatchesPaths(paths),
			Files:        files,
			FilesOmitted: omitted,
		})
	}
	rep.Similar, rep.SimilarOmitted = similarPairs(current, in.Vectors)
	slices.SortFunc(rep.StaleAppliesTo, func(a, b Card) int { return cmp.Compare(a.ID, b.ID) })
	slices.SortFunc(rep.Features, func(a, b Feature) int { return cmp.Compare(a.ID, b.ID) })
	slices.SortFunc(rep.MissedRules, func(a, b Card) int { return cmp.Compare(a.ID, b.ID) })
	return rep, nil
}

// LastReview is the time of the latest successful full review in the log.
// A due check and a failed review do not count.
func LastReview(entries []cmdlog.Entry) time.Time {
	type call struct {
		dueOnly bool
		outAt   time.Time
		failed  bool
		hasOut  bool
	}
	byID := map[string]*call{}
	var pending []*call
	var calls []*call
	for _, e := range entries {
		if commandBase(e.Command) != "review" {
			continue
		}
		switch e.Dir {
		case "in":
			c := &call{dueOnly: dueOnlyArgs(e.Args)}
			calls = append(calls, c)
			if e.ID != "" {
				byID[e.ID] = c
			} else {
				pending = append(pending, c)
			}
		case "out":
			var c *call
			if e.ID != "" {
				c = byID[e.ID]
				delete(byID, e.ID)
			} else if len(pending) > 0 {
				c = pending[0]
				pending = pending[1:]
			}
			if c == nil {
				c = &call{}
				calls = append(calls, c)
			}
			c.hasOut = true
			c.outAt = parseStamp(e.TS)
			c.failed = e.Error != ""
		}
	}
	var last time.Time
	for _, c := range calls {
		if !c.hasOut || c.failed || c.dueOnly || c.outAt.IsZero() {
			continue
		}
		if last.IsZero() || c.outAt.After(last) {
			last = c.outAt
		}
	}
	return last
}

func due(last, now time.Time) bool {
	if last.IsZero() {
		return true
	}
	return !now.Before(last.Add(Interval))
}

func readLog(root string, cfg *config.Config) ([]cmdlog.Entry, bool, error) {
	if cfg == nil || !cfg.LogCommands {
		return nil, true, nil
	}
	path := config.CommandsLogPath(root)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	entries, err := cmdlog.Read(path)
	return entries, false, err
}

func loadRecords(stores ...*store.Store) ([]*record.Record, error) {
	var out []*record.Record
	seen := map[string]struct{}{}
	for _, st := range stores {
		if st == nil {
			continue
		}
		recs, err := st.AllRecords()
		if err != nil {
			return nil, err
		}
		for _, rec := range recs {
			if rec == nil || rec.ID == "" {
				continue
			}
			if _, ok := seen[rec.ID]; ok {
				continue
			}
			seen[rec.ID] = struct{}{}
			out = append(out, rec)
		}
	}
	return out, nil
}

func loadVectors(model string, stores ...*store.Store) (map[string][]float32, error) {
	out := map[string][]float32{}
	for _, st := range stores {
		if st == nil {
			continue
		}
		rows, err := st.StoredVectors()
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if model != "" && row.Model != model {
				continue
			}
			if _, ok := out[row.ID]; ok {
				continue
			}
			out[row.ID] = row.Values
		}
	}
	return out, nil
}

// staleApplies reports a current record whose globs match no file.
// Repo records are about this checkout. Global and dev records are stale
// only when a leading directory of applies_to exists here, so a product
// path is not stale in an unrelated tree.
func staleApplies(rec *record.Record, files []string, dirs map[string]struct{}) bool {
	if rec == nil || len(rec.AppliesTo) == 0 || rec.MatchesPaths(files) {
		return false
	}
	if rec.Scope == record.ScopeRepo {
		return true
	}
	for _, p := range rec.AppliesTo {
		if anchoredHere(p, dirs) {
			return true
		}
	}
	return false
}

func anchoredHere(pattern string, dirs map[string]struct{}) bool {
	pattern = strings.TrimSpace(pattern)
	pattern = strings.TrimPrefix(pattern, "/")
	if pattern == "" || !strings.Contains(pattern, "/") {
		return false
	}
	first, _, _ := strings.Cut(pattern, "/")
	first = strings.Trim(first, "*")
	if first == "" || strings.ContainsAny(first, "*?[") {
		return false
	}
	_, ok := dirs[first]
	return ok
}

func currentRecords(recs []*record.Record) []*record.Record {
	seen := map[string]struct{}{}
	var out []*record.Record
	for _, rec := range recs {
		if rec == nil || rec.ID == "" || rec.Retired() {
			continue
		}
		if _, ok := seen[rec.ID]; ok {
			continue
		}
		seen[rec.ID] = struct{}{}
		out = append(out, rec)
	}
	return out
}

func missedRules(entries []cmdlog.Entry, since time.Time, paths []string, recs []*record.Record) ([]trace.RecordHit, bool, error) {
	rep, err := trace.Build(entries, trace.Options{
		Since: since.UTC().Format(time.RFC3339),
		After: since,
		Paths: paths,
	}, recordCatalog{recs: recs})
	if err != nil {
		return nil, false, err
	}
	return rep.MissedRules, rep.WideDiff, nil
}

type recordCatalog struct {
	recs []*record.Record
}

func (c recordCatalog) Record(id string) (*record.Record, error) {
	for _, rec := range c.recs {
		if rec != nil && rec.ID == id {
			return rec, nil
		}
	}
	return nil, nil
}

func (c recordCatalog) Rules() ([]*record.Record, error) {
	var out []*record.Record
	for _, rec := range c.recs {
		if rec != nil && rec.Type == record.TypeRule {
			out = append(out, rec)
		}
	}
	return out, nil
}

func similarPairs(recs []*record.Record, vectors map[string][]float32) ([]Pair, int) {
	var have []*record.Record
	for _, rec := range recs {
		if _, ok := vectors[rec.ID]; ok {
			have = append(have, rec)
		}
	}
	slices.SortFunc(have, func(a, b *record.Record) int { return cmp.Compare(a.ID, b.ID) })
	var pairs []Pair
	for i := range have {
		for j := i + 1; j < len(have); j++ {
			a, b := have[i], have[j]
			score := cosine(vectors[a.ID], vectors[b.ID])
			if score < similarMinCosine {
				continue
			}
			pairs = append(pairs, Pair{Cosine: score, A: cardFrom(a), B: cardFrom(b)})
		}
	}
	slices.SortFunc(pairs, func(a, b Pair) int {
		if c := cmp.Compare(b.Cosine, a.Cosine); c != 0 {
			return c
		}
		if c := cmp.Compare(a.A.ID, b.A.ID); c != 0 {
			return c
		}
		return cmp.Compare(a.B.ID, b.B.ID)
	})
	if len(pairs) <= maxSimilar {
		return pairs, 0
	}
	return pairs[:maxSimilar], len(pairs) - maxSimilar
}

func matchingFiles(rec *record.Record, files []string, limit int) ([]string, int) {
	var sample []string
	n := 0
	for _, f := range files {
		if !rec.MatchesPaths([]string{f}) {
			continue
		}
		n++
		if len(sample) < limit {
			sample = append(sample, f)
		}
	}
	omitted := 0
	if n > limit {
		omitted = n - limit
	}
	return sample, omitted
}

func cardFrom(rec *record.Record) Card {
	return Card{
		ID:        rec.ID,
		Title:     rec.Title,
		Type:      string(rec.Type),
		Scope:     string(rec.Scope),
		AppliesTo: append([]string(nil), rec.AppliesTo...),
	}
}

func cosine(a, b []float32) float64 {
	if len(a) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		av := float64(a[i])
		bv := float64(b[i])
		dot += av * bv
		na += av * av
		nb += bv * bv
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func dueOnlyArgs(args any) bool {
	m, ok := args.(map[string]any)
	if !ok {
		return false
	}
	if truthy(m["due"]) {
		return true
	}
	flags, ok := m["flags"].(map[string]any)
	if !ok {
		return false
	}
	return truthy(flags["due"])
}

func truthy(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return strings.EqualFold(strings.TrimSpace(t), "true")
	default:
		return false
	}
}

func commandBase(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, " "); i >= 0 {
		s = s[i+1:]
	}
	return s
}

func parseStamp(s string) time.Time {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// FormatDue renders the clock for stdout.
func FormatDue(r DueReport) string {
	var b strings.Builder
	if r.LoggingOff {
		b.WriteString("log_commands is false; review has no clock\n")
	}
	fmt.Fprintf(&b, "due: %t\n", r.Due)
	if !r.LastReview.IsZero() {
		fmt.Fprintf(&b, "last review: %s\n", r.LastReview.UTC().Format(time.RFC3339))
	} else if !r.LoggingOff {
		b.WriteString("last review: none\n")
	}
	return b.String()
}

// Format renders the evidence list for stdout.
func Format(r *Report) string {
	if r == nil {
		return ""
	}
	var b strings.Builder
	if !r.Since.IsZero() {
		fmt.Fprintf(&b, "Since: %s\n", r.Since.UTC().Format(time.RFC3339))
	}
	if r.LoggingOff {
		b.WriteString("log_commands is false; missed rules were not computed\n")
	}
	if r.GitSkipped != "" {
		fmt.Fprintf(&b, "Git join skipped: %s\n", r.GitSkipped)
	} else {
		fmt.Fprintf(&b, "Changed paths: %d\n", r.ChangedPaths)
		if r.GitNote != "" {
			fmt.Fprintf(&b, "Paths: %s\n", r.GitNote)
		}
	}
	if r.WideDiff {
		b.WriteString("Wide diff: the path set is too broad for a miss to mean the rule was skipped\n")
	}
	writeCards(&b, "Missed rules", r.MissedRules)
	writeCards(&b, "Stale applies_to", r.StaleAppliesTo)
	if len(r.Similar) > 0 {
		fmt.Fprintf(&b, "Similar: %d\n", len(r.Similar))
		if r.SimilarOmitted > 0 {
			fmt.Fprintf(&b, "  %d more above the floor\n", r.SimilarOmitted)
		}
		for _, p := range r.Similar {
			fmt.Fprintf(&b, "  %.3f %s %s || %s %s\n", p.Cosine, p.A.ID, p.A.Title, p.B.ID, p.B.Title)
		}
	}
	if len(r.Features) > 0 {
		fmt.Fprintf(&b, "Features: %d\n", len(r.Features))
		for _, f := range r.Features {
			line := "  " + f.ID + " " + f.Title
			if f.Changed {
				line += " changed"
			}
			fmt.Fprintf(&b, "%s\n", line)
			for _, file := range f.Files {
				fmt.Fprintf(&b, "    %s\n", file)
			}
			if f.FilesOmitted > 0 {
				fmt.Fprintf(&b, "    (+%d more)\n", f.FilesOmitted)
			}
		}
	}
	if r.Unembedded > 0 {
		fmt.Fprintf(&b, "Unembedded: %d\n", r.Unembedded)
	}
	if len(r.MissedRules) == 0 && len(r.StaleAppliesTo) == 0 && len(r.Similar) == 0 && len(r.Features) == 0 && r.Unembedded == 0 && !r.LoggingOff && r.GitSkipped == "" && !r.WideDiff {
		b.WriteString("Nothing to report.\n")
	}
	return b.String()
}

func writeCards(b *strings.Builder, title string, cards []Card) {
	if len(cards) == 0 {
		return
	}
	fmt.Fprintf(b, "%s: %d\n", title, len(cards))
	for _, c := range cards {
		fmt.Fprintf(b, "  %s %s\n", c.ID, c.Title)
		if len(c.AppliesTo) > 0 {
			fmt.Fprintf(b, "    %s\n", strings.Join(c.AppliesTo, ", "))
		}
	}
}
