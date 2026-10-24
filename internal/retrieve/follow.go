package retrieve

import (
	"context"
	"fmt"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
)

// SearchLink is one followed edge on a search response.
// record and pin carry a search card. repo carries search cards in hits.
type SearchLink struct {
	Kind    string       `json:"kind"`
	FromID  string       `json:"from_id,omitempty"`
	Archive string       `json:"archive,omitempty"`
	Record  *SearchCard  `json:"record,omitempty"`
	Hits    []SearchCard `json:"hits,omitempty"`
	Missing string       `json:"missing,omitempty"`
}

// GetLink is one edge attached to a record found in this checkout.
// record edges carry the full record. repo edges carry only the checkout.
type GetLink struct {
	Kind    string      `json:"kind"`
	FromID  string      `json:"from_id,omitempty"`
	Archive string      `json:"archive,omitempty"`
	Record  *RecordCard `json:"record,omitempty"`
	Missing string      `json:"missing,omitempty"`
}

type SearchEnvelope struct {
	Results any          `json:"results"`
	Links   []SearchLink `json:"links"`
}

// Envelope wraps ranked hits. links is never null.
func Envelope(results any, links []SearchLink) SearchEnvelope {
	if links == nil {
		links = []SearchLink{}
	}
	return SearchEnvelope{Results: results, Links: links}
}

// Follow loads this checkout's links and attaches the ones the ranked hits trigger.
// It does not merge those hits into results. A missing target is recorded on the
// link and does not fail the search. Linked checkouts are not searched for their own links.
func Follow(ctx context.Context, e *Engine, embedder embed.Embedder, opts Options, results []Result) ([]SearchLink, error) {
	if e == nil || e.Repo == nil {
		return []SearchLink{}, nil
	}
	edges, err := e.Repo.ListLinks()
	if err != nil {
		return nil, err
	}
	if len(edges) == 0 {
		return []SearchLink{}, nil
	}
	repoHits, err := repoHitIDs(e.Repo, results)
	if err != nil {
		return nil, err
	}
	opened := map[string]*store.Store{}
	defer func() {
		for _, db := range opened {
			_ = db.Close()
		}
	}()
	repoSearch := map[string]SearchLink{}
	out := []SearchLink{}
	for _, edge := range edges {
		switch edge.Kind {
		case store.LinkPin:
			out = append(out, searchRecordLink(edge, opts.Query, opened))
		case store.LinkRecord:
			if _, ok := repoHits[edge.FromID]; !ok {
				continue
			}
			out = append(out, searchRecordLink(edge, opts.Query, opened))
		case store.LinkRepo:
			if _, ok := repoHits[edge.FromID]; !ok {
				continue
			}
			out = append(out, searchRepoLink(ctx, e, embedder, opts, edge, opened, repoSearch))
		}
	}
	return out, nil
}

// RepoEngine is the search engine for one linked checkout.
// Home and extras stay empty so that checkout's own links are not followed.
func RepoEngine(parent *Engine, db *store.Store) *Engine {
	model := ""
	if parent != nil {
		model = parent.QueryModel
	}
	return &Engine{Repo: db, QueryModel: model}
}

// LinksForRecord lists record and repo edges whose from_id is rec.
// Pins are omitted. Repo edges do not run a search.
func LinksForRecord(repo *store.Store, rec *record.Record) ([]GetLink, error) {
	if repo == nil || rec == nil {
		return nil, nil
	}
	edges, err := repo.ListLinks()
	if err != nil {
		return nil, err
	}
	opened := map[string]*store.Store{}
	defer func() {
		for _, db := range opened {
			_ = db.Close()
		}
	}()
	out := []GetLink{}
	for _, edge := range edges {
		if edge.FromID != rec.ID {
			continue
		}
		switch edge.Kind {
		case store.LinkRepo:
			out = append(out, GetLink{Kind: edge.Kind, FromID: edge.FromID, Archive: edge.ToRoot})
		case store.LinkRecord:
			out = append(out, getRecordLink(edge, opened))
		}
	}
	return out, nil
}

func repoHitIDs(repo *store.Store, results []Result) (map[string]struct{}, error) {
	ids := make([]string, 0, len(results))
	for _, r := range results {
		if r.Archive != "" || r.Record == nil || r.Record.ID == "" {
			continue
		}
		ids = append(ids, r.Record.ID)
	}
	found, err := repo.GetRecordsByIDs(ids)
	if err != nil {
		return nil, err
	}
	out := make(map[string]struct{}, len(found))
	for id := range found {
		out[id] = struct{}{}
	}
	return out, nil
}

func searchRecordLink(edge store.Link, query string, opened map[string]*store.Store) SearchLink {
	link := SearchLink{Kind: edge.Kind, FromID: edge.FromID, Archive: edge.ToRoot}
	rec, err := recordIn(edge.ToRoot, edge.ToID, opened)
	if err != nil {
		link.Missing = err.Error()
		return link
	}
	cards := ProjectSearch([]Result{{Record: rec, Snippet: Snippet(rec.Body, query)}})
	link.Record = &cards[0]
	return link
}

func searchRepoLink(ctx context.Context, parent *Engine, embedder embed.Embedder, opts Options, edge store.Link, opened map[string]*store.Store, cache map[string]SearchLink) SearchLink {
	if prev, ok := cache[edge.ToRoot]; ok {
		prev.FromID = edge.FromID
		return prev
	}
	link := SearchLink{Kind: edge.Kind, FromID: edge.FromID, Archive: edge.ToRoot}
	db, err := openLinked(edge.ToRoot, opened)
	if err != nil {
		link.Missing = err.Error()
		cache[edge.ToRoot] = link
		return link
	}
	hits, err := RepoEngine(parent, db).Search(ctx, embedder, opts)
	if err != nil {
		link.Missing = err.Error()
		cache[edge.ToRoot] = link
		return link
	}
	link.Hits = ProjectSearch(hits)
	cache[edge.ToRoot] = link
	return link
}

func getRecordLink(edge store.Link, opened map[string]*store.Store) GetLink {
	link := GetLink{Kind: edge.Kind, FromID: edge.FromID, Archive: edge.ToRoot}
	rec, err := recordIn(edge.ToRoot, edge.ToID, opened)
	if err != nil {
		link.Missing = err.Error()
		return link
	}
	card := ProjectRecord(rec)
	link.Record = &card
	return link
}

func recordIn(root, id string, opened map[string]*store.Store) (*record.Record, error) {
	db, err := openLinked(root, opened)
	if err != nil {
		return nil, err
	}
	rec, ok, err := db.GetRecordByID(id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("record links: record %s not found", id)
	}
	return rec, nil
}

func openLinked(root string, opened map[string]*store.Store) (*store.Store, error) {
	if db, ok := opened[root]; ok {
		return db, nil
	}
	db, err := store.OpenReadOnly(config.StorePath(root))
	if err != nil {
		return nil, err
	}
	opened[root] = db
	return db, nil
}
