package retrieve

import "github.com/bwireman/archivist/internal/record"

// SearchCard is the default JSON hit for MCP search and CLI search --json.
// Source is the retrieval channel (fts, vector, or hybrid), not a file path.
type SearchCard struct {
	ID      string        `json:"id"`
	Slug    string        `json:"slug"`
	Type    record.Type   `json:"type"`
	Scope   record.Scope  `json:"scope"`
	Title   string        `json:"title"`
	Status  record.Status `json:"status"`
	Score   float64       `json:"score"`
	Source  string        `json:"source"`
	Archive string        `json:"archive,omitempty"`
}

// RecordCard is the default JSON body for MCP get.
type RecordCard struct {
	ID           string          `json:"id"`
	Slug         string          `json:"slug"`
	Type         record.Type     `json:"type"`
	Scope        record.Scope    `json:"scope"`
	Title        string          `json:"title"`
	Status       record.Status   `json:"status"`
	Severity     record.Severity `json:"severity"`
	Body         string          `json:"body"`
	Tags         []string        `json:"tags"`
	AppliesTo    []string        `json:"applies_to"`
	SupersededBy string          `json:"superseded_by"`
}

// ProjectSearch copies hits into cards. A nil slice becomes an empty slice so
// JSON encodes as [] rather than null.
func ProjectSearch(results []Result) []SearchCard {
	cards := make([]SearchCard, 0, len(results))
	for _, r := range results {
		cards = append(cards, SearchCard{
			ID: r.Record.ID, Slug: r.Record.Slug, Type: r.Record.Type, Scope: r.Record.Scope,
			Title: r.Record.Title, Status: r.Record.Status, Score: r.Score, Source: r.Source,
			Archive: r.Archive,
		})
	}
	return cards
}

func ProjectRecord(r *record.Record) RecordCard {
	return RecordCard{
		ID: r.ID, Slug: r.Slug, Type: r.Type, Scope: r.Scope,
		Title: r.Title, Status: r.Status, Severity: r.Severity, Body: r.Body,
		Tags: r.Tags, AppliesTo: r.AppliesTo, SupersededBy: r.SupersededBy,
	}
}
