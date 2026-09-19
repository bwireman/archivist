package retrieve

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

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

	idSet := map[string]struct{}{}
	for id := range ftsRank {
		idSet[id] = struct{}{}
	}
	for id := range vectorRank {
		idSet[id] = struct{}{}
	}
	ids := make([]string, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	recs, err := e.lookupRecords(ids)
	if err != nil {
		return nil, err
	}

	merged := map[string]Result{}
	for id := range idSet {
		rec, ok := recs[id]
		if !ok {
			continue
		}
		source := "hybrid"
		if ftsRank[id] > 0 && vectorRank[id] == 0 {
			source = "fts"
		} else if vectorRank[id] > 0 && ftsRank[id] == 0 {
			source = "vector"
		}
		merged[id] = Result{
			Record: rec,
			Score:  rrfScore(ftsRank[id], vectorRank[id]),
			Source: source,
		}
	}

	results := make([]Result, 0, len(merged))
	for _, r := range merged {
		results = append(results, r)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Score > results[j].Score })

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
