package index

import "fmt"

// Progress is the result of an Index run.
type Progress struct {
	FilesIndexed int
	FilesRemoved int
	CommitsNew   int
}

// FormatSummary is the one-line result printed after indexing.
func FormatSummary(p Progress, recordCount int) string {
	if p.FilesIndexed == 0 && p.CommitsNew == 0 && p.FilesRemoved == 0 {
		return fmt.Sprintf("Index up to date (%d records)", recordCount)
	}
	return fmt.Sprintf("Indexed %d %s, %d %s (%d records)",
		p.FilesIndexed, plural(p.FilesIndexed, "file", "files"),
		p.CommitsNew, plural(p.CommitsNew, "commit", "commits"),
		recordCount)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
