package tui

import (
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/index"
)

func TestFormatSummary(t *testing.T) {
	upToDate := FormatSummary(index.Progress{}, 215)
	if upToDate != "Index up to date (215 chunks)" {
		t.Fatalf("got %q", upToDate)
	}
	changed := FormatSummary(index.Progress{FilesIndexed: 3, CommitsNew: 1}, 40)
	if changed != "Indexed 3 files, 1 commit (40 chunks)" {
		t.Fatalf("got %q", changed)
	}
}

func TestIndexViewShowsProgress(t *testing.T) {
	m := newModel(nil, func() {})
	m.progress = index.Progress{
		Phase:        index.PhaseFiles,
		Path:         "internal/store/store.go",
		Detail:       "embed 2/5",
		FilesSeen:    4,
		FilesTotal:   10,
		FilesIndexed: 3,
	}
	out := m.View()
	for _, want := range []string{"archivist index", "files", "4/10", "internal/store/store.go", "embed 2/5", "3 indexed"} {
		if !strings.Contains(out, want) {
			t.Fatalf("view missing %q:\n%s", want, out)
		}
	}
}

func TestPhaseString(t *testing.T) {
	if index.PhaseFiles.String() != "files" {
		t.Fatalf("got %q", index.PhaseFiles.String())
	}
}
