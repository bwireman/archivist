// Package trace reports whether archive lookups showed up in later work.
package trace

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/cmdlog"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/retrieve"
)

const followWindow = 30 * time.Minute

// widePathCount is when a --since diff is too broad for overlap to mean much.
const widePathCount = 30

// Catalog loads records for the git join. Record returns nil when id is unknown.
type Catalog interface {
	Record(id string) (*record.Record, error)
	Rules() ([]*record.Record, error)
}

// Options selects the log window and the git join.
type Options struct {
	Since      string
	After      time.Time
	Paths      []string
	GitSkipped string
}

// QueryRef is one digest row that came from a tool call.
type QueryRef struct {
	At     time.Time `json:"at,omitzero"`
	Query  string    `json:"query,omitempty"`
	ID     string    `json:"id,omitempty"`
	Title  string    `json:"title,omitempty"`
	Note   string    `json:"note,omitempty"`
	Status string    `json:"status,omitempty"`
}

// CiteHit is one cite call.
type CiteHit struct {
	At     time.Time `json:"at,omitzero"`
	ID     string    `json:"id"`
	Effect string    `json:"effect,omitempty"`
}

// RecordHit is a record in the git join.
type RecordHit struct {
	ID        string   `json:"id"`
	Title     string   `json:"title,omitempty"`
	AppliesTo []string `json:"applies_to,omitempty"`
}

// Report is the trace digest.
type Report struct {
	LoggingOff bool `json:"logging_off,omitempty"`
	Empty      bool `json:"empty,omitempty"`

	Counts map[string]int `json:"counts,omitempty"`

	EmptySearches     []QueryRef `json:"empty_searches,omitempty"`
	KeywordOnly       []QueryRef `json:"keyword_only_searches,omitempty"`
	VectorOnly        []QueryRef `json:"vector_only_searches,omitempty"`
	SupersededTop     []QueryRef `json:"superseded_top_hits,omitempty"`
	Incomplete        []QueryRef `json:"incomplete_searches,omitempty"`
	FollowGets        []QueryRef `json:"follow_through_gets,omitempty"`
	UnmatchedGets     []QueryRef `json:"gets_without_prior_hit,omitempty"`
	RemembersNoSearch []QueryRef `json:"remembers_without_search,omitempty"`
	CheckMatches      []QueryRef `json:"check_matches,omitempty"`
	Cites             []CiteHit  `json:"cites,omitempty"`

	Cited     int `json:"cited"`
	Retrieved int `json:"retrieved"`

	Since        string      `json:"since,omitempty"`
	GitSkipped   string      `json:"git_skipped,omitempty"`
	WideDiff     bool        `json:"wide_diff,omitempty"`
	ChangedPaths []string    `json:"changed_paths,omitempty"`
	Overlap      []RecordHit `json:"overlap,omitempty"`
	NoGlob       []RecordHit `json:"retrieved_no_glob,omitempty"`
	MissedRules  []RecordHit `json:"rules_absent_from_log,omitempty"`
}

type searchEvent struct {
	at    time.Time
	ids   map[string]struct{}
	query string
}

type hit struct {
	id, title, status, source, note string
	applies                         []string
}

type idMeta struct {
	title   string
	applies []string
}

var countOrder = []string{"search", "get", "map", "check", "remember", "update", "retire", "cite"}

var skippedCommand = map[string]bool{
	"index": true, "embed": true, "import": true, "export": true, "publish": true, "status": true,
}

// Build summarizes command-log entries. Maintenance commands are ignored.
// A search result marked truncated is incomplete and is left out of follow-through.
// Vector-only search hits stay in follow-through and out of the consulted set.
func Build(entries []cmdlog.Entry, opts Options, cat Catalog) (*Report, error) {
	rep := &Report{Counts: map[string]int{}}
	var searches []searchEvent
	retrieved := map[string]struct{}{}
	consulted := map[string]struct{}{}
	var consultedOrder []string
	meta := map[string]idMeta{}
	cited := map[string]struct{}{}

	// addConsulted records a get, cite, check, or hybrid/keyword search hit.
	// Vector-only search hits stay out. countRetrieved is false for cite:
	// cite is the numerator, not the cited/retrieved denominator.
	addConsulted := func(id string, m idMeta, countRetrieved bool) {
		if id == "" {
			return
		}
		if prev, ok := meta[id]; ok {
			if m.title == "" {
				m.title = prev.title
			}
			if len(m.applies) == 0 {
				m.applies = prev.applies
			}
		}
		meta[id] = m
		if countRetrieved {
			retrieved[id] = struct{}{}
		}
		if _, ok := consulted[id]; ok {
			return
		}
		consulted[id] = struct{}{}
		consultedOrder = append(consultedOrder, id)
	}

	for _, call := range pairCalls(entries) {
		if skippedCommand[call.name] || !call.hasOut {
			continue
		}
		at := call.at()
		if !opts.After.IsZero() && (at.IsZero() || at.Before(opts.After)) {
			continue
		}
		rep.Counts[call.name]++
		switch call.name {
		case "search":
			q := queryOf(call.in.Args)
			ev := searchEvent{at: at, query: q, ids: map[string]struct{}{}}
			if call.out.Error == "" {
				hits, kind := parseSearch(call.out.Result)
				ref := QueryRef{At: at, Query: q}
				switch kind {
				case "truncated":
					rep.Incomplete = append(rep.Incomplete, ref)
				case "empty":
					rep.EmptySearches = append(rep.EmptySearches, ref)
				default:
					allFTS := len(hits) > 0
					allVector := len(hits) > 0
					for _, h := range hits {
						ev.ids[h.id] = struct{}{}
						switch strings.ToLower(h.source) {
						case retrieve.SourceFTS:
							allVector = false
							addConsulted(h.id, idMeta{title: h.title, applies: h.applies}, true)
						case retrieve.SourceVector:
							allFTS = false
						default:
							allFTS = false
							allVector = false
							addConsulted(h.id, idMeta{title: h.title, applies: h.applies}, true)
						}
					}
					if allFTS {
						rep.KeywordOnly = append(rep.KeywordOnly, ref)
					}
					if allVector {
						rep.VectorOnly = append(rep.VectorOnly, ref)
					}
					if len(hits) > 0 && strings.EqualFold(hits[0].status, string(record.StatusSuperseded)) {
						top := hits[0]
						rep.SupersededTop = append(rep.SupersededTop, QueryRef{
							At: at, Query: q, ID: top.id, Title: top.title, Status: top.status,
						})
					}
				}
			}
			searches = append(searches, ev)
		case "get":
			if call.out.Error != "" {
				break
			}
			h := recordHit(call.out.Result)
			if h.id == "" {
				break
			}
			addConsulted(h.id, idMeta{title: h.title, applies: h.applies}, true)
			ref := QueryRef{At: at, ID: h.id, Title: h.title}
			if followed(searches, at, h.id) {
				rep.FollowGets = append(rep.FollowGets, ref)
			} else {
				rep.UnmatchedGets = append(rep.UnmatchedGets, ref)
			}
		case "remember":
			if call.out.Error != "" {
				break
			}
			if !searchInWindow(searches, at) {
				rep.RemembersNoSearch = append(rep.RemembersNoSearch, QueryRef{
					At:    at,
					ID:    resultID(call.out.Result),
					Title: rememberTitle(call.in.Args),
				})
			}
		case "check":
			if call.out.Error != "" {
				break
			}
			for _, m := range checkMatches(call.out.Result) {
				addConsulted(m.id, idMeta{title: m.title, applies: m.applies}, true)
				rep.CheckMatches = append(rep.CheckMatches, QueryRef{
					At: at, ID: m.id, Title: m.title, Note: m.note, Status: m.status,
				})
			}
		case "cite":
			if call.out.Error != "" {
				break
			}
			id, effect := citeFields(call)
			if id == "" {
				break
			}
			addConsulted(id, idMeta{}, false)
			cited[id] = struct{}{}
			rep.Cites = append(rep.Cites, CiteHit{At: at, ID: id, Effect: effect})
		}
	}

	rep.Cited = len(cited)
	rep.Retrieved = len(retrieved)

	if opts.Since != "" {
		rep.Since = opts.Since
		rep.GitSkipped = opts.GitSkipped
		if opts.GitSkipped == "" {
			rep.ChangedPaths = append([]string(nil), opts.Paths...)
			rep.WideDiff = len(opts.Paths) > widePathCount
			for _, id := range consultedOrder {
				hit, err := resolve(cat, id, meta[id])
				if err != nil {
					return nil, err
				}
				if len(hit.AppliesTo) == 0 {
					rep.NoGlob = append(rep.NoGlob, hit)
					continue
				}
				rec := &record.Record{ID: hit.ID, Title: hit.Title, AppliesTo: hit.AppliesTo}
				if rec.MatchesPaths(opts.Paths) {
					rep.Overlap = append(rep.Overlap, hit)
				}
			}
			if cat != nil {
				rules, err := cat.Rules()
				if err != nil {
					return nil, err
				}
				for _, r := range rules {
					if !currentRule(r) {
						continue
					}
					if _, ok := consulted[r.ID]; ok {
						continue
					}
					if !r.MatchesPaths(opts.Paths) {
						continue
					}
					rep.MissedRules = append(rep.MissedRules, RecordHit{
						ID: r.ID, Title: r.Title, AppliesTo: append([]string(nil), r.AppliesTo...),
					})
				}
				slices.SortFunc(rep.MissedRules, func(a, b RecordHit) int {
					return cmp.Compare(a.ID, b.ID)
				})
			}
		}
	}
	return rep, nil
}

func resolve(cat Catalog, id string, fallback idMeta) (RecordHit, error) {
	hit := RecordHit{ID: id, Title: fallback.title, AppliesTo: fallback.applies}
	if cat == nil {
		return hit, nil
	}
	rec, err := cat.Record(id)
	if err != nil {
		return RecordHit{}, err
	}
	if rec == nil {
		return hit, nil
	}
	return RecordHit{
		ID: rec.ID, Title: rec.Title, AppliesTo: append([]string(nil), rec.AppliesTo...),
	}, nil
}

func currentRule(r *record.Record) bool {
	return r != nil && r.Type == record.TypeRule && !r.Retired()
}

func withinFollow(searchAt, at time.Time) bool {
	return !at.Before(searchAt) && at.Sub(searchAt) <= followWindow
}

func followed(searches []searchEvent, at time.Time, id string) bool {
	for _, s := range searches {
		if !withinFollow(s.at, at) {
			continue
		}
		if _, ok := s.ids[id]; ok {
			return true
		}
	}
	return false
}

func searchInWindow(searches []searchEvent, at time.Time) bool {
	for _, s := range searches {
		if withinFollow(s.at, at) {
			return true
		}
	}
	return false
}

type paired struct {
	name   string
	in     cmdlog.Entry
	out    cmdlog.Entry
	hasOut bool
}

func (c paired) at() time.Time {
	if c.hasOut {
		if t := entryTime(c.out); !t.IsZero() {
			return t
		}
	}
	return entryTime(c.in)
}

func pairCalls(entries []cmdlog.Entry) []paired {
	pending := map[string][]int{}
	var calls []paired
	for _, e := range entries {
		name := baseCommand(e.Command)
		switch e.Dir {
		case "in":
			calls = append(calls, paired{name: name, in: e})
			pending[name] = append(pending[name], len(calls)-1)
		case "out":
			q := pending[name]
			if len(q) == 0 {
				calls = append(calls, paired{name: name, out: e, hasOut: true})
				continue
			}
			i := q[0]
			pending[name] = q[1:]
			calls[i].out = e
			calls[i].hasOut = true
		}
	}
	return calls
}

func entryTime(e cmdlog.Entry) time.Time {
	t, _ := parseTime(e.TS)
	return t
}

func parseTime(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

func baseCommand(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, " "); i >= 0 {
		s = s[i+1:]
	}
	return s
}

func parseSearch(result any) ([]hit, string) {
	if isTruncated(result) {
		return nil, "truncated"
	}
	arr, ok := asSlice(result)
	if !ok || len(arr) == 0 {
		return nil, "empty"
	}
	var hits []hit
	for _, item := range arr {
		m, ok := asMap(item)
		if !ok {
			continue
		}
		h := recordHit(mapField(m, "Record", "record"))
		h.source = strField(m, "Source", "source")
		if h.id != "" {
			hits = append(hits, h)
		}
	}
	if len(hits) == 0 {
		return nil, "empty"
	}
	return hits, "hits"
}

func checkMatches(result any) []hit {
	m, ok := asMap(result)
	if !ok {
		return nil
	}
	arr, ok := asSlice(mapField(m, "Matches", "matches"))
	if !ok {
		return nil
	}
	var hits []hit
	for _, item := range arr {
		mm, ok := asMap(item)
		if !ok {
			continue
		}
		h := recordHit(mapField(mm, "Record", "record"))
		h.note = strField(mm, "Reason", "reason")
		if sev := strField(mm, "Severity", "severity"); sev != "" {
			h.status = sev
		}
		if h.id != "" {
			hits = append(hits, h)
		}
	}
	return hits
}

func recordHit(v any) hit {
	m, ok := asMap(v)
	if !ok {
		return hit{}
	}
	return hit{
		id:      strField(m, "ID", "id"),
		title:   strField(m, "Title", "title"),
		status:  strField(m, "Status", "status"),
		applies: stringSlice(mapField(m, "AppliesTo", "applies_to")),
	}
}

func resultID(v any) string {
	return strFieldMap(v, "id", "ID")
}

func citeFields(call paired) (string, string) {
	id := strFieldMap(call.out.Result, "id", "ID")
	effect := strFieldMap(call.out.Result, "effect", "Effect")
	args, _ := asMap(call.in.Args)
	if id == "" {
		id = strField(args, "id", "ID")
		if id == "" {
			id = firstString(mapField(args, "args"))
		}
	}
	if effect == "" {
		effect = strField(args, "effect", "Effect")
		if effect == "" {
			if flags, ok := asMap(mapField(args, "flags")); ok {
				effect = strField(flags, "effect", "Effect")
			}
		}
	}
	return id, effect
}

func rememberTitle(args any) string {
	m, ok := asMap(args)
	if !ok {
		return ""
	}
	if title := strField(m, "title", "Title"); title != "" {
		return title
	}
	if flags, ok := asMap(mapField(m, "flags")); ok {
		return strField(flags, "title", "Title")
	}
	return ""
}

func queryOf(args any) string {
	m, ok := asMap(args)
	if !ok {
		return ""
	}
	if q := strField(m, "query", "Query"); q != "" {
		return q
	}
	parts := stringSlice(mapField(m, "args"))
	return strings.Join(parts, " ")
}

func isTruncated(v any) bool {
	m, ok := asMap(v)
	if !ok {
		return false
	}
	t, _ := m["truncated"].(bool)
	return t
}

func asMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func asSlice(v any) ([]any, bool) {
	s, ok := v.([]any)
	return s, ok
}

func mapField(m map[string]any, keys ...string) any {
	if m == nil {
		return nil
	}
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v
		}
	}
	return nil
}

func strField(m map[string]any, keys ...string) string {
	v := mapField(m, keys...)
	s, _ := v.(string)
	return s
}

func strFieldMap(v any, keys ...string) string {
	m, ok := asMap(v)
	if !ok {
		return ""
	}
	return strField(m, keys...)
}

func firstString(v any) string {
	switch t := v.(type) {
	case []any:
		if len(t) == 0 {
			return ""
		}
		s, _ := t[0].(string)
		return s
	case []string:
		if len(t) == 0 {
			return ""
		}
		return t[0]
	default:
		return ""
	}
}

func stringSlice(v any) []string {
	switch t := v.(type) {
	case []string:
		return append([]string(nil), t...)
	case []any:
		var out []string
		for _, x := range t {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// Format renders a digest for stdout.
func Format(r *Report) string {
	if r == nil {
		return ""
	}
	var b strings.Builder
	if r.LoggingOff {
		b.WriteString("log_commands is false; no commands.log\n")
		return b.String()
	}
	if r.Empty {
		b.WriteString("log_commands is true; no commands logged yet\n")
		return b.String()
	}
	fmt.Fprintf(&b, "%s\n", formatCounts(r.Counts))
	writeRefs(&b, "Empty searches", r.EmptySearches, func(q QueryRef) string {
		return strings.TrimSpace(q.At.Format(time.RFC3339) + " " + q.Query)
	})
	writeRefs(&b, "Keyword-only searches", r.KeywordOnly, func(q QueryRef) string {
		return strings.TrimSpace(q.At.Format(time.RFC3339) + " " + q.Query)
	})
	writeRefs(&b, "Vector-only searches", r.VectorOnly, func(q QueryRef) string {
		return strings.TrimSpace(q.At.Format(time.RFC3339) + " " + q.Query)
	})
	writeRefs(&b, "Superseded top hits", r.SupersededTop, func(q QueryRef) string {
		return strings.TrimSpace(q.ID + " " + q.Title)
	})
	writeRefs(&b, "Incomplete searches", r.Incomplete, func(q QueryRef) string {
		return strings.TrimSpace(q.At.Format(time.RFC3339) + " " + q.Query)
	})
	writeRefs(&b, "Follow-through gets", r.FollowGets, func(q QueryRef) string {
		return strings.TrimSpace(q.ID + " " + q.Title)
	})
	writeRefs(&b, "Gets with no prior hit", r.UnmatchedGets, func(q QueryRef) string {
		return strings.TrimSpace(q.ID + " " + q.Title)
	})
	writeRefs(&b, "Remembers with no prior search", r.RemembersNoSearch, func(q QueryRef) string {
		return strings.TrimSpace(q.ID + " " + q.Title)
	})
	writeRefs(&b, "Check matches", r.CheckMatches, func(q QueryRef) string {
		return fmt.Sprintf("[%s] %s — %s (%s)", q.Status, q.Title, q.Note, q.ID)
	})
	if len(r.Cites) > 0 {
		fmt.Fprintf(&b, "Cites: %d\n", len(r.Cites))
		for _, c := range r.Cites {
			fmt.Fprintf(&b, "  %s %s\n", c.ID, c.Effect)
		}
	}
	fmt.Fprintf(&b, "Cited / retrieved: %d/%d\n", r.Cited, r.Retrieved)
	if r.Since != "" {
		fmt.Fprintf(&b, "Since: %s\n", r.Since)
		if r.GitSkipped != "" {
			fmt.Fprintf(&b, "Git join skipped: %s\n", r.GitSkipped)
		} else {
			fmt.Fprintf(&b, "Changed paths: %d\n", len(r.ChangedPaths))
			if r.WideDiff {
				fmt.Fprintf(&b, "Wide diff: %d paths; overlap is coarse\n", len(r.ChangedPaths))
			}
			for _, p := range r.ChangedPaths {
				fmt.Fprintf(&b, "  %s\n", p)
			}
			writeHits(&b, "Overlap", r.Overlap)
			writeHits(&b, "Consulted with no path glob", r.NoGlob)
			writeHits(&b, "Rules matching the diff and absent from the log", r.MissedRules)
		}
	}
	return b.String()
}

func formatCounts(counts map[string]int) string {
	seen := map[string]bool{}
	var parts []string
	for _, name := range countOrder {
		seen[name] = true
		parts = append(parts, fmt.Sprintf("%s %d", name, counts[name]))
	}
	for _, name := range slices.Sorted(maps.Keys(counts)) {
		if seen[name] {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s %d", name, counts[name]))
	}
	return strings.Join(parts, ", ")
}

func writeRefs(b *strings.Builder, title string, rows []QueryRef, line func(QueryRef) string) {
	if len(rows) == 0 {
		return
	}
	fmt.Fprintf(b, "%s: %d\n", title, len(rows))
	for _, row := range rows {
		fmt.Fprintf(b, "  %s\n", line(row))
	}
}

func writeHits(b *strings.Builder, title string, rows []RecordHit) {
	if len(rows) == 0 {
		return
	}
	fmt.Fprintf(b, "%s: %d\n", title, len(rows))
	for _, row := range rows {
		fmt.Fprintf(b, "  %s %s\n", row.ID, row.Title)
	}
}
