// Package retrieve searches records with full-text search and embeddings.
package retrieve

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
)

const DefaultTopK = 20

// Measured 2026-10-04: off-topic tops (porter, dungeon, Go style) were at most 0.41.
// A paraphrase of the command-log feature with no FTS hit scored 0.55.
// Hybrid and keyword hits are not filtered.
const vectorOnlyMinCosine = 0.50

const (
	SourceFTS    = "fts"
	SourceVector = "vector"
	SourceHybrid = "hybrid"
)

type Options struct {
	TopK   int
	Type   record.Type
	Scope  record.Scope
	Status record.Status
	Query  string
}

type Result struct {
	Record  *record.Record
	Score   float64
	Source  string // "fts", "vector", "hybrid"
	Archive string // extra checkout root; empty for the primary checkout and the home store
}

// Extra is one listed checkout opened for search.
type Extra struct {
	Root string
	DB   *store.Store
}

type Engine struct {
	Repo   *store.Store
	Home   *store.Store
	Extras []Extra
}

type searchStore struct {
	db   *store.Store
	root string
}

func (e *Engine) searchStores() []searchStore {
	if e == nil {
		return nil
	}
	var out []searchStore
	if e.Repo != nil {
		out = append(out, searchStore{db: e.Repo})
	}
	if e.Home != nil {
		out = append(out, searchStore{db: e.Home})
	}
	for _, ex := range e.Extras {
		if ex.DB == nil {
			continue
		}
		out = append(out, searchStore{db: ex.DB, root: ex.Root})
	}
	return out
}

func (e *Engine) Search(ctx context.Context, embedder embed.Embedder, opts Options) ([]Result, error) {
	if opts.TopK <= 0 {
		opts.TopK = DefaultTopK
	}
	filter, err := searchFilter(opts)
	if err != nil {
		return nil, err
	}
	rankLimit := opts.TopK * 3

	// anyTermRank holds hits from the any-term retry. They only add rank to
	// vector hits that already cleared the floor; on their own they are noise.
	ftsRank := map[string]int{}
	anyTermRank := map[string]int{}
	for _, src := range e.searchStores() {
		st := src.db
		ftsResults, err := st.SearchFTS(opts.Query, rankLimit, filter)
		if err != nil {
			return nil, err
		}
		for i, fr := range ftsResults {
			rank := ftsRank
			if fr.AnyTerm {
				rank = anyTermRank
			}
			if _, ok := rank[fr.RecordID]; !ok {
				rank[fr.RecordID] = i + 1
			}
		}
	}

	var qEmb []float32
	if embedder != nil {
		emb, err := embedder.Embed(ctx, opts.Query)
		if err != nil {
			fmt.Fprintf(os.Stderr, "search embed: %v; using keyword-only\n", err)
		} else {
			qEmb = emb
		}
	}

	vectorRank := map[string]int{}
	if len(qEmb) > 0 {
		for _, src := range e.searchStores() {
			st := src.db
			ranked, err := st.RankEmbeddings(qEmb, filter, rankLimit)
			if err != nil {
				return nil, err
			}
			for i, row := range ranked {
				if _, ok := vectorRank[row.RecordID]; ok {
					continue
				}
				// Keep the original rank index. A weak vector-only hit is omitted
				// rather than compacted, so later hits keep the rank they earned.
				if _, inFTS := ftsRank[row.RecordID]; !inFTS && row.Score < vectorOnlyMinCosine {
					continue
				}
				vectorRank[row.RecordID] = i + 1
			}
		}
	}

	ids := make([]string, 0, len(ftsRank)+len(vectorRank))
	for id := range ftsRank {
		ids = append(ids, id)
	}
	for id := range vectorRank {
		if _, ok := ftsRank[id]; !ok {
			ids = append(ids, id)
		}
	}
	recs, archives, err := e.lookupRecords(ids)
	if err != nil {
		return nil, err
	}

	// A record can exist in both stores under the same type and slug; the
	// narrower scope wins the overlay. Keying on type as well as slug keeps
	// unrelated records that happen to share a slug from evicting each other.
	best := map[string]Result{}
	for _, id := range ids {
		rec, ok := recs[id]
		if !ok {
			continue
		}
		kw := ftsRank[id]
		if kw == 0 {
			kw = anyTermRank[id]
		}
		archiveRoot := archives[id]
		candidate := Result{
			Record:  rec,
			Score:   rrfScore(kw, vectorRank[id]),
			Source:  hitSource(kw, vectorRank[id]),
			Archive: archiveRoot,
		}
		key := overlayKey(archiveRoot, rec)
		if existing, ok := best[key]; ok && !overlayWins(candidate, existing) {
			continue
		}
		best[key] = candidate
	}

	results := make([]Result, 0, len(best))
	for _, r := range best {
		results = append(results, r)
	}
	// Ids come out of map iteration, so break score ties on id to keep the
	// same query returning the same ordering.
	slices.SortFunc(results, func(a, b Result) int {
		return cmp.Or(cmp.Compare(b.Score, a.Score), cmp.Compare(a.Record.ID, b.Record.ID))
	})
	if len(results) > opts.TopK {
		results = results[:opts.TopK]
	}
	return results, nil
}

// overlayWins reports whether candidate should replace existing for the same
// type and slug: narrower scope first, then the better retrieval score.
func overlayWins(candidate, existing Result) bool {
	cp := record.ScopePrecedence(candidate.Record.Scope)
	ep := record.ScopePrecedence(existing.Record.Scope)
	if cp != ep {
		return cp > ep
	}
	return candidate.Score > existing.Score
}

func searchFilter(opts Options) (store.RecordFilter, error) {
	f := store.RecordFilter{Type: opts.Type, Scope: opts.Scope}
	if opts.Status == "" {
		f.ExcludeRetired = true
		return f, nil
	}
	if !record.ValidStatus(opts.Status) {
		return f, fmt.Errorf("invalid status %q", opts.Status)
	}
	f.Status = opts.Status
	return f, nil
}

func hitSource(ftsRank, vectorRank int) string {
	switch {
	case vectorRank == 0:
		return SourceFTS
	case ftsRank == 0:
		return SourceVector
	default:
		return SourceHybrid
	}
}

func overlayKey(root string, rec *record.Record) string {
	base := string(rec.Type) + "/" + rec.Slug
	if root != "" {
		return root + "\x00" + base
	}
	return base
}

func (e *Engine) lookupRecords(ids []string) (map[string]*record.Record, map[string]string, error) {
	out := make(map[string]*record.Record, len(ids))
	archives := map[string]string{}
	remaining := ids
	for _, src := range e.searchStores() {
		if len(remaining) == 0 {
			break
		}
		found, err := src.db.GetRecordsByIDs(remaining)
		if err != nil {
			return nil, nil, err
		}
		next := make([]string, 0, len(remaining))
		for _, id := range remaining {
			rec, ok := found[id]
			if !ok {
				next = append(next, id)
				continue
			}
			out[id] = rec
			if src.root != "" {
				archives[id] = src.root
			}
		}
		remaining = next
	}
	return out, archives, nil
}

func rrfScore(ftsRank, vectorRank int) float64 {
	const k = 60.0
	var score float64
	if ftsRank > 0 {
		score += 1.0 / (k + float64(ftsRank))
	}
	if vectorRank > 0 {
		score += 1.0 / (k + float64(vectorRank))
	}
	return score
}

func FormatResults(results []Result) string {
	var b strings.Builder
	for i, r := range results {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%d. [%.4f] %s/%s %s\n   %s\n",
			i+1, r.Score, r.Record.Type, r.Record.Scope, r.Record.Title, r.Record.SourcePath)
		if r.Archive != "" {
			fmt.Fprintf(&b, "   %s\n", r.Archive)
		}
	}
	if len(results) == 0 {
		b.WriteString("No results.\n")
	}
	return b.String()
}
