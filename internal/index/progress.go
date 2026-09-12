package index

import "fmt"

// Phase is the stage of an Index run.
type Phase int

const (
	PhaseScan Phase = iota
	PhaseFiles
	PhasePrune
	PhaseGit
	PhaseDone
)

func (p Phase) String() string {
	switch p {
	case PhaseScan:
		return "scan"
	case PhaseFiles:
		return "files"
	case PhasePrune:
		return "prune"
	case PhaseGit:
		return "git"
	case PhaseDone:
		return "done"
	default:
		return ""
	}
}

// Progress is a snapshot of an in-flight Index run.
type Progress struct {
	Phase  Phase
	Path   string
	Detail string

	FilesTotal     int
	FilesSeen      int
	FilesIndexed   int
	FilesUnchanged int
	FilesRemoved   int

	CommitsTotal int
	CommitsNew   int

	ChunksEmbedded int
}

// Reporter receives Progress snapshots. Calls may come from the indexer
// goroutine; implementations must be safe for that.
type Reporter func(Progress)

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
