package index

// Phase is the stage of an Index run shown in the TUI.
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
