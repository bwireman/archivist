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
			rows, err := st.ListEmbeddings(filter)
			if err != nil {
				return nil, err
			}
			var scored []struct {
				id    string
				score float64
			}
			for _, row := range rows {
				scored = append(scored, struct {
					id    string
					score float64
				}{row.RecordID, store.CosineSimilarity(qEmb, row.Embedding)})
			}
			sort.Slice(scored, func(i, j int) bool { return scored[i].score > scored[j].score })
			for i, s := range scored {
				if i >= rankLimit {
					break
				}
				if _, ok := vectorRank[s.id]; !ok {
					vectorRank[s.id] = i + 1
				}
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

	merged := map[string]Result{}
	for id := range allIDs {
		rec, ok, err := e.lookupRecord(id)
		if err != nil {
			return nil, err
		}
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
