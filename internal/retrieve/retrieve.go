package retrieve

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
)

const DefaultTopK = 20

type Options struct {
	TopK      int
	Type      record.Type
	Scope     record.Scope
	Query     string
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
	merged := map[string]Result{}

	ftsRank := map[string]int{}
	vectorRank := map[string]int{}

	for _, st := range []*store.Store{e.Repo, e.Home} {
		if st == nil {
			continue
		}
		ftsResults, err := st.SearchFTS(opts.Query, opts.TopK*3)
		if err != nil {
			return nil, err
		}
		for i, fr := range ftsResults {
			ftsRank[fr.RecordID] = i + 1
		}
	}

	var qEmb []float32
	if embedder != nil {
		qEmb, _ = embedder.Embed(ctx, opts.Query)
	}
	if len(qEmb) > 0 {
		for _, st := range []*store.Store{e.Repo, e.Home} {
			if st == nil {
				continue
			}
			vectors, err := st.AllVectorRecords()
			if err != nil {
				return nil, err
			}
			var scored []struct {
				id    string
				score float64
			}
			for _, vr := range vectors {
				score := store.CosineSimilarity(qEmb, vr.Embedding)
				scored = append(scored, struct {
					id    string
					score float64
				}{vr.Record.ID, score})
			}
			sort.Slice(scored, func(i, j int) bool { return scored[i].score > scored[j].score })
			for i, s := range scored {
				if i >= opts.TopK*3 {
					break
				}
				vectorRank[s.id] = i + 1
			}
		}
	}

	allIDs := map[string]struct{}{}
	for id := range ftsRank {
		allIDs[id] = struct{}{}
	}
	for id := range vectorRank {
		allIDs[id] = struct{}{}
	}

	for id := range allIDs {
		score := rrfScore(ftsRank[id], vectorRank[id])
		rec, ok, err := e.lookupRecord(id)
		if err != nil {
			return nil, err
		}
		if !ok || !matchFilters(rec, opts) {
			continue
		}
		source := "hybrid"
		if ftsRank[id] > 0 && vectorRank[id] == 0 {
			source = "fts"
		} else if vectorRank[id] > 0 && ftsRank[id] == 0 {
			source = "vector"
		}
		merged[id] = Result{Record: rec, Score: score, Source: source}
	}

	results := make([]Result, 0, len(merged))
	for _, r := range merged {
		results = append(results, r)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })

	// scope overlay: repo > global > dev for same slug
	bySlug := map[string]Result{}
	for _, r := range results {
		existing, ok := bySlug[r.Record.Slug]
		if !ok || record.ScopePrecedence(r.Record.Scope) > record.ScopePrecedence(existing.Record.Scope) {
			bySlug[r.Record.Slug] = r
		}
	}
	results = make([]Result, 0, len(bySlug))
	for _, r := range bySlug {
		results = append(results, r)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if len(results) > opts.TopK {
		results = results[:opts.TopK]
	}

	if e.Repo != nil {
		_ = e.Repo.StampSearch(opts.Query, time.Now())
	}
	return results, nil
}

func (e *Engine) lookupRecord(id string) (*record.Record, bool, error) {
	if e.Repo != nil {
		rec, ok, err := e.Repo.GetRecordByID(id)
		if err != nil || ok {
			return rec, ok, err
		}
	}
	if e.Home != nil {
		return e.Home.GetRecordByID(id)
	}
	return nil, false, nil
}

func matchFilters(rec *record.Record, opts Options) bool {
	if opts.Type != "" && rec.Type != opts.Type {
		return false
	}
	if opts.Scope != "" && rec.Scope != opts.Scope {
		return false
	}
	return true
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
