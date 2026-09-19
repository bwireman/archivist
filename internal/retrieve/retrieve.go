package retrieve

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
)

const DefaultTopK = 20

type Options struct {
	TopK  int
	Type  record.Type
	Scope record.Scope
	Query string
}

type Result struct {
	Record *record.Record
	Score  float64
	Source string // "fts", "vector", "hybrid"
}

type Engine struct {
	Repo *store.Store
	Home *store.Store
}

func (e *Engine) Search(ctx context.Context, embedder embed.Embedder, opts Options) ([]Result, error) {
	if opts.TopK <= 0 {
		opts.TopK = DefaultTopK
	}
	filter := store.RecordFilter{Type: opts.Type, Scope: opts.Scope}
	rankLimit := opts.TopK * 3

	ftsRank := map[string]int{}
	for _, st := range []*store.Store{e.Repo, e.Home} {
		if st == nil {
			continue
		}
		ftsResults, err := st.SearchFTS(opts.Query, rankLimit, filter)
		if err != nil {
			return nil, err
		}
		for i, fr := range ftsResults {
			if _, ok := ftsRank[fr.RecordID]; !ok {
				ftsRank[fr.RecordID] = i + 1
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
		for _, st := range []*store.Store{e.Repo, e.Home} {
			if st == nil {
				continue
			}
			ranked, err := st.RankEmbeddings(qEmb, filter, rankLimit)
			if err != nil {
				return nil, err
			}
			for i, row := range ranked {
				if _, ok := vectorRank[row.RecordID]; !ok {
					vectorRank[row.RecordID] = i + 1
				}
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
	recs, err := e.lookupRecords(ids)
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
		candidate := Result{
			Record: rec,
			Score:  rrfScore(ftsRank[id], vectorRank[id]),
			Source: hitSource(ftsRank[id], vectorRank[id]),
		}
		key := string(rec.Type) + "/" + rec.Slug
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
	sort.Slice(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		return results[i].Record.ID < results[j].Record.ID
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

func hitSource(ftsRank, vectorRank int) string {
	switch {
	case vectorRank == 0:
		return "fts"
	case ftsRank == 0:
		return "vector"
	default:
		return "hybrid"
	}
}

func (e *Engine) lookupRecords(ids []string) (map[string]*record.Record, error) {
	out := make(map[string]*record.Record, len(ids))
	remaining := ids
	if e.Repo != nil {
		found, err := e.Repo.GetRecordsByIDs(remaining)
		if err != nil {
			return nil, err
		}
		next := make([]string, 0, len(remaining))
		for _, id := range remaining {
			if rec, ok := found[id]; ok {
				out[id] = rec
			} else {
				next = append(next, id)
			}
		}
		remaining = next
	}
	if e.Home != nil && len(remaining) > 0 {
		found, err := e.Home.GetRecordsByIDs(remaining)
		if err != nil {
			return nil, err
		}
		for id, rec := range found {
			out[id] = rec
		}
	}
	return out, nil
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
	}
	if len(results) == 0 {
		b.WriteString("No results.\n")
	}
	return b.String()
}
