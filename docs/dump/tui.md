# Archivist context dump

Retrieved chunks from a local code index. Use only facts present here.

- Query: init TUI huh bubbletea
- Chunks: 20

## `docs/global-decisions/001-bubbletea-index-tui.md` (adr/global, lines 1-6, score 0.837)

```md
# Use Bubble Tea for the index TUI

- Status: accepted
- Date: 2026-08-28
- Scope: global
```


## `docs/global-decisions/001-bubbletea-index-tui.md` (adr/global, lines 10-12, score 0.754)

```md
## Decision
Show a Bubble Tea TUI when stdout is a TTY. Fall back to a one-line summary when it is not, or when `--plain` is set (`make index` uses `--plain`). The indexer reports progress through a callback so tests do not need a terminal.
```


## `docs/global-decisions/002-huh-init-tui.md` (adr/global, lines 1-6, score 0.728)

```md
# Use Huh for the init TUI

- Status: accepted
- Date: 2026-08-28
- Scope: global
```


## `docs/global-decisions/002-huh-init-tui.md` (adr/global, lines 10-12, score 0.560)

```md
## Decision
Show a Huh form on a TTY so `archivist init` can edit Ollama, index, and store settings before writing `.archivist.json`. `--plain` writes `config.Default()` with no prompt.
```


## `76677f6b1f213af652be027800ac149e83c5e510` (commit, no line range, score 0.559)

```
Commit: 76677f6b1f213af652be027800ac149e83c5e510
Author: bwireman
Date: 2026-08-28T18:50:52-05:00
Subject: Add TUIs for index progress and init setup.

Interactive runs get a live index view and a Huh form for writing .archivist.json; --plain keeps make and scripts as one-line logs.

Files:
.archivist.json
Makefile
docs/decisions/002-bubbletea-index-tui.md
docs/decisions/003-huh-init-tui.md
docs/dump/decisions.md
docs/dump/tui.md
go.mod
go.sum
internal/cmd/index.go
internal/cmd/init.go
internal/cmd/root.go
internal/index/indexer.go
internal/index/indexer_test.go
internal/index/progress.go
internal/tui/index.go
internal/tui/index_test.go
internal/tui/init.go
internal/tui/init_test.go
```


## `go.sum` (code, lines 1-21, score 0.543)

Blame: Not Committed Yet (0000000000000000000000000000000000000000)

```
github.com/MakeNowJust/heredoc v1.0.0 h1:cXCdzVdstXyiTqTvfqk9SDHpKNjxuom+DOlyEeQ4pzQ=
github.com/MakeNowJust/heredoc v1.0.0/go.mod h1:mG5amYoWBHf8vpLOuehzbGGw0EHxpZZ6lCpQ4fNJ8LE=
github.com/atotto/clipboard v0.1.4 h1:EH0zSVneZPSuFR11BlR9YppQTVDbh5+16AmcJi4g1z4=
github.com/atotto/clipboard v0.1.4/go.mod h1:ZY9tmq7sm5xIbd9bOK4onWV4S6X0u6GY7Vn0Yu86PYI=
github.com/aymanbagabas/go-osc52/v2 v2.0.1 h1:HwpRHbFMcZLEVr42D4p7XBqjyuxQH5SMiErDT4WkJ2k=
github.com/aymanbagabas/go-osc52/v2 v2.0.1/go.mod h1:uYgXzlJ7ZpABp8OJ+exZzJJhRNQ2ASbcXHWsFqH8hp8=
github.com/aymanbagabas/go-udiff v0.3.1 h1:LV+qyBQ2pqe0u42ZsUEtPiCaUoqgA9gYRDs3vj1nolY=
github.com/aymanbagabas/go-udiff v0.3.1/go.mod h1:G0fsKmG+P6ylD0r6N/KgQD/nWzgfnl8ZBcNLgcbrw8E=
github.com/catppuccin/go v0.3.0 h1:d+0/YicIq+hSTo5oPuRi5kOpqkVA5tAsU6dNhvRu+aY=
github.com/catppuccin/go v0.3.0/go.mod h1:8IHJuMGaUUjQM82qBrGNBv7LFq6JI3NnQCF6MOlZjpc=
github.com/charmbracelet/bubbles v1.0.0 h1:12J8/ak/uCZEMQ6KU7pcfwceyjLlWsDLAxB5fXonfvc=
github.com/charmbracelet/bubbles v1.0.0/go.mod h1:9d/Zd5GdnauMI5ivUIVisuEm3ave1XwXtD1ckyV6r3E=
github.com/charmbracelet/bubbletea v1.3.10 h1:otUDHWMMzQSB0Pkc87rm691KZ3SWa4KUlvF9nRvCICw=
github.com/charmbracelet/bubbletea v1.3.10/go.mod h1:ORQfo0fk8U+po9VaNvnV95UPWA1BitP1E0N6xJPlHr4=
github.com/charmbracelet/colorprofile v0.4.1 h1:a1lO03qTrSIRaK8c3JRxJDZOvhvIeSco3ej+ngLk1kk=
github.com/charmbracelet/colorprofile v0.4.1/go.mod h1:U1d9Dljmdf9DLegaJ0nGZNJvoXAhayhmidOdcBwAvKk=
github.com/charmbracelet/harmonica v0.2.0 h1:8NxJWRWg/bzKqqEaaeFNipOu77YR5t8aSwG4pgaUBiQ=
github.com/charmbracelet/harmonica v0.2.0/go.mod h1:KSri/1RMQOZLbw7AHqgcBycp8pgJnQMYYT8QZRqZ1Ao=
github.com/charmbracelet/huh v1.0.0 h1:wOnedH8G4qzJbmhftTqrpppyqHakl/zbbNdXIWJyIxw=
github.com/charmbracelet/huh v1.0.0/go.mod h1:5YVc+SlZ1IhQALxRPpkGwwEKftN/+OlJlnJYlDRFqN4=
github.com/charmbracelet/lipgloss v1.1.0 h1:vYXsiLHVkK7fp74RkV7b2kq9+zDLoEU4MZoFqR/noCY=
```


## `go.sum` (code, lines 22-38, score 0.496)

Blame: Not Committed Yet (0000000000000000000000000000000000000000)

```
github.com/charmbracelet/huh v1.0.0/go.mod h1:5YVc+SlZ1IhQALxRPpkGwwEKftN/+OlJlnJYlDRFqN4=
github.com/charmbracelet/lipgloss v1.1.0 h1:vYXsiLHVkK7fp74RkV7b2kq9+zDLoEU4MZoFqR/noCY=
github.com/charmbracelet/lipgloss v1.1.0/go.mod h1:/6Q8FR2o+kj8rz4Dq0zQc3vYf7X+B0binUUBwA0aL30=
github.com/charmbracelet/x/ansi v0.11.6 h1:GhV21SiDz/45W9AnV2R61xZMRri5NlLnl6CVF7ihZW8=
github.com/charmbracelet/x/ansi v0.11.6/go.mod h1:2JNYLgQUsyqaiLovhU2Rv/pb8r6ydXKS3NIttu3VGZQ=
github.com/charmbracelet/x/cellbuf v0.0.15 h1:ur3pZy0o6z/R7EylET877CBxaiE1Sp1GMxoFPAIztPI=
github.com/charmbracelet/x/cellbuf v0.0.15/go.mod h1:J1YVbR7MUuEGIFPCaaZ96KDl5NoS0DAWkskup+mOY+Q=
github.com/charmbracelet/x/conpty v0.1.0 h1:4zc8KaIcbiL4mghEON8D72agYtSeIgq8FSThSPQIb+U=
github.com/charmbracelet/x/conpty v0.1.0/go.mod h1:rMFsDJoDwVmiYM10aD4bH2XiRgwI7NYJtQgl5yskjEQ=
github.com/charmbracelet/x/errors v0.0.0-20240508181413-e8d8b6e2de86 h1:JSt3B+U9iqk37QUU2Rvb6DSBYRLtWqFqfxf8l5hOZUA=
github.com/charmbracelet/x/errors v0.0.0-20240508181413-e8d8b6e2de86/go.mod h1:2P0UgXMEa6TsToMSuFqKFQR+fZTO9CNGUNokkPatT/0=
github.com/charmbracelet/x/exp/golden v0.0.0-20241011142426-46044092ad91 h1:payRxjMjKgx2PaCWLZ4p3ro9y97+TVLZNaRZgJwSVDQ=
github.com/charmbracelet/x/exp/golden v0.0.0-20241011142426-46044092ad91/go.mod h1:wDlXFlCrmJ8J+swcL/MnGUuYnqgQdW9rhSD61oNMb6U=
github.com/charmbracelet/x/exp/strings v0.0.0-20240722160745-212f7b056ed0 h1:qko3AQ4gK1MTS/de7F5hPGx6/k1u0w4TeYmBFwzYVP4=
github.com/charmbracelet/x/exp/strings v0.0.0-20240722160745-212f7b056ed0/go.mod h1:pBhA0ybfXv6hDjQUZ7hk1lVxBiUbupdw5R31yPUViVQ=
github.com/charmbracelet/x/term v0.2.2 h1:xVRT/S2ZcKdhhOuSP4t5cLi5o+JxklsoEObBSgfgZRk=
github.com/charmbracelet/x/term v0.2.2/go.mod h1:kF8CY5RddLWrsgVwpw4kAa6TESp6EB5y3uxGLeCqzAI=
github.com/charmbracelet/x/termios v0.1.1 h1:o3Q2bT8eqzGnGPOYheoYS8eEleT5ZVNYNy8JawjaNZY=
github.com/charmbracelet/x/termios v0.1.1/go.mod h1:rB7fnv1TgOPOyyKRJ9o+AsTU/vK5WHJ2ivHeut/Pcwo=
```


## `go.sum` (code, lines 39-57, score 0.494)

Blame: Not Committed Yet (0000000000000000000000000000000000000000)

```
github.com/charmbracelet/x/termios v0.1.1 h1:o3Q2bT8eqzGnGPOYheoYS8eEleT5ZVNYNy8JawjaNZY=
github.com/charmbracelet/x/termios v0.1.1/go.mod h1:rB7fnv1TgOPOyyKRJ9o+AsTU/vK5WHJ2ivHeut/Pcwo=
github.com/charmbracelet/x/xpty v0.1.2 h1:Pqmu4TEJ8KeA9uSkISKMU3f+C1F6OGBn8ABuGlqCbtI=
github.com/charmbracelet/x/xpty v0.1.2/go.mod h1:XK2Z0id5rtLWcpeNiMYBccNNBrP2IJnzHI0Lq13Xzq4=
github.com/clipperhouse/displaywidth v0.9.0 h1:Qb4KOhYwRiN3viMv1v/3cTBlz3AcAZX3+y9OLhMtAtA=
github.com/clipperhouse/displaywidth v0.9.0/go.mod h1:aCAAqTlh4GIVkhQnJpbL0T/WfcrJXHcj8C0yjYcjOZA=
github.com/clipperhouse/stringish v0.1.1 h1:+NSqMOr3GR6k1FdRhhnXrLfztGzuG+VuFDfatpWHKCs=
github.com/clipperhouse/stringish v0.1.1/go.mod h1:v/WhFtE1q0ovMta2+m+UbpZ+2/HEXNWYXQgCt4hdOzA=
github.com/clipperhouse/uax29/v2 v2.5.0 h1:x7T0T4eTHDONxFJsL94uKNKPHrclyFI0lm7+w94cO8U=
github.com/clipperhouse/uax29/v2 v2.5.0/go.mod h1:Wn1g7MK6OoeDT0vL+Q0SQLDz/KpfsVRgg6W7ihQeh4g=
github.com/cpuguy83/go-md2man/v2 v2.0.6/go.mod h1:oOW0eioCTA6cOiMLiUPZOpcVxMig6NIQQ7OS05n1F4g=
github.com/creack/pty v1.1.24 h1:bJrF4RRfyJnbTJqzRLHzcGaZK1NeM5kTC9jGgovnR1s=
github.com/creack/pty v1.1.24/go.mod h1:08sCNb52WyoAwi2QDyzUCTgcvVFhUzewun7wtTfvcwE=
github.com/davecgh/go-spew v1.1.1 h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=
github.com/davecgh/go-spew v1.1.1/go.mod h1:J7Y8YcW2NihsgmVo/mv3lAwl/skON4iLHjSsI+c5H38=
github.com/dustin/go-humanize v1.0.1 h1:GzkhY7T5VNhEkwH0PVJgjz+fX1rhBrR7pRT3mDkpeCY=
github.com/dustin/go-humanize v1.0.1/go.mod h1:Mu1zIs6XwVuF/gI1OepvI0qD18qycQx+mFykh5fBlto=
github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f h1:Y/CXytFA4m6baUTXGLOoWe4PQhGxaX0KpnayAqC48p4=
github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f/go.mod h1:vw97MGsxSvLiUE2X8qFplwetxpGLQrlU1Q9AUEIzCaM=
github.com/google/pprof v0.0.0-20260802141513-ef3492d7dac3 h1:LMLX+LgTNWpfvCBdFebv6EsYotImrt/Ppc5cXIriCSo=
github.com/google/pprof v0.0.0-20260802141513-ef3492d7dac3/go.mod h1:jl5iWTm0/hd5PjEYEOuwAJ57L/CibdZfrqZ5XA5GrCk=
```


## `go.mod` (code, lines 1-44, score 0.490)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```
module github.com/bwireman/archivist

go 1.26.7

require (
	github.com/charmbracelet/bubbles v1.0.0
	github.com/charmbracelet/bubbletea v1.3.10
	github.com/charmbracelet/huh v1.0.0
	github.com/charmbracelet/lipgloss v1.1.0
	github.com/mattn/go-isatty v0.0.24
	github.com/smacker/go-tree-sitter v0.0.0-20240827094217-dd81d9e9be82
	github.com/spf13/cobra v1.10.2
	modernc.org/sqlite v1.57.0
)

require (
	github.com/atotto/clipboard v0.1.4 // indirect
	github.com/aymanbagabas/go-osc52/v2 v2.0.1 // indirect
	github.com/catppuccin/go v0.3.0 // indirect
	github.com/charmbracelet/colorprofile v0.4.1 // indirect
	github.com/charmbracelet/harmonica v0.2.0 // indirect
	github.com/charmbracelet/x/ansi v0.11.6 // indirect
	github.com/charmbracelet/x/cellbuf v0.0.15 // indirect
	github.com/charmbracelet/x/exp/strings v0.0.0-20240722160745-212f7b056ed0 // indirect
	github.com/charmbracelet/x/term v0.2.2 // indirect
	github.com/clipperhouse/displaywidth v0.9.0 // indirect
	github.com/clipperhouse/stringish v0.1.1 // indirect
	github.com/clipperhouse/uax29/v2 v2.5.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/erikgeiser/coninput v0.0.0-20211004153227-1c3628e74d0f // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/lucasb-eyer/go-colorful v1.3.0 // indirect
	github.com/mattn/go-localereader v0.0.1 // indirect
	github.com/mattn/go-runewidth v0.0.19 // indirect
	github.com/mitchellh/hashstructure/v2 v2.0.2 // indirect
	github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/muesli/termenv v0.16.0 // indirect
	github.com/ncruces/go-strftime v1.0.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	github.com/xo/terminfo v0.0.0-20220910002029-abceb7e1c41e // indirect
```


## `internal/tui/index.go` (code, lines 93-95, score 0.474)

```go
File: internal/tui/index.go

package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwireman/archivist/internal/index"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Width(7)
	pathStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	phaseStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("177"))
)

type progressMsg index.Progress

type doneMsg struct{}

type model struct {
	ch       <-chan index.Progress

func (m model) Init() tea.Cmd {
	return tea.Batch(m.listen(), m.spin.Tick)
}
```


## `docs/global-decisions/002-huh-init-tui.md` (adr/global, lines 7-9, score 0.469)

```md
## Context
`archivist init` wrote `config.Default()` with no prompts. The embed model is a real choice (`nomic-embed-text` vs a locally installed model). The index command already uses a Charm TUI on a TTY.
```


## `.cursor/rules/archivist-dump.mdc` (doc, lines 1-5, score 0.463)

```
---
description: Dump indexed decisions into docs/dump for later LLM sessions
alwaysApply: true
---
```


## `internal/tui/init.go` (code, lines 177-183, score 0.462)

Blame: bwireman (76677f6b1f213af652be027800ac149e83c5e510)

```go
File: internal/tui/init.go

package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/config"
	"github.com/charmbracelet/huh"
)

// InitForm is the editable string view of a Config used by the init TUI.
type InitForm struct {
	BaseURL        string
	EmbedModel     string
	EmbedTimeout   string
	SkipDirs       string
	SkipGlobs      string
	ADRPaths       string
	GlobalADRPaths string
	StorePath      string
}

// FormFromConfig flattens cfg into form fields.
func FormFromConfig(cfg *config.Config) InitForm {
	if cfg == nil {
		cfg = config.Default()
	}

func FormatInitSummary(cfg *config.Config, existed bool) string {
	verb := "Created"
	if existed {
		verb = "Updated"
	}
	return fmt.Sprintf("%s .archivist.json and .archivist/\nEmbeddings use Ollama:\n  ollama pull %s\n", verb, cfg.Ollama.EmbedModel)
}
```


## `internal/tui/index.go` (code, lines 107-137, score 0.458)

```go
File: internal/tui/index.go

package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwireman/archivist/internal/index"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Width(7)
	pathStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	phaseStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("177"))
)

type progressMsg index.Progress

type doneMsg struct{}

type model struct {
	ch       <-chan index.Progress

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		w := msg.Width - 20
		if w < 12 {
			w = 12
		}
		if w > 40 {
			w = 40
		}
		m.bar.Width = w
		return m, nil
	case progressMsg:
		m.progress = index.Progress(msg)
		return m, m.listen()
	case doneMsg:
		return m, tea.Quit
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			m.cancel()
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	}
	return m, nil
}
```


## `internal/tui/index.go` (code, lines 97-105, score 0.457)

```go
File: internal/tui/index.go

package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwireman/archivist/internal/index"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Width(7)
	pathStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	phaseStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("177"))
)

type progressMsg index.Progress

type doneMsg struct{}

type model struct {
	ch       <-chan index.Progress

func (m model) listen() tea.Cmd {
	return func() tea.Msg {
		p, ok := <-m.ch
		if !ok {
			return doneMsg{}
		}
		return progressMsg(p)
	}
}
```


## `internal/tui/init.go` (code, lines 44-69, score 0.455)

Blame: bwireman (76677f6b1f213af652be027800ac149e83c5e510)

```go
File: internal/tui/init.go

package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/config"
	"github.com/charmbracelet/huh"
)

// InitForm is the editable string view of a Config used by the init TUI.
type InitForm struct {
	BaseURL        string
	EmbedModel     string
	EmbedTimeout   string
	SkipDirs       string
	SkipGlobs      string
	ADRPaths       string
	GlobalADRPaths string
	StorePath      string
}

// FormFromConfig flattens cfg into form fields.
func FormFromConfig(cfg *config.Config) InitForm {
	if cfg == nil {
		cfg = config.Default()
	}

func ConfigFromForm(form InitForm) (*config.Config, error) {
	cfg := config.Default()
	cfg.Ollama.BaseURL = strings.TrimSpace(form.BaseURL)
	cfg.Ollama.EmbedModel = strings.TrimSpace(form.EmbedModel)
	cfg.Ollama.EmbedTimeout = strings.TrimSpace(form.EmbedTimeout)
	cfg.Index.SkipDirs = SplitList(form.SkipDirs)
	cfg.Index.SkipGlobs = SplitList(form.SkipGlobs)
	cfg.Index.ADRPaths = SplitList(form.ADRPaths)
	cfg.Index.GlobalADRPaths = SplitList(form.GlobalADRPaths)
	cfg.Store.Path = strings.TrimSpace(form.StorePath)

	if cfg.Ollama.BaseURL == "" {
		return nil, fmt.Errorf("ollama base URL is required")
	}
	if cfg.Ollama.EmbedModel == "" {
		return nil, fmt.Errorf("embed model is required")
	}
	if cfg.Store.Path == "" {
		return nil, fmt.Errorf("store path is required")
	}
	d, err := time.ParseDuration(cfg.Ollama.EmbedTimeout)
	if err != nil || d <= 0 {
		return nil, fmt.Errorf("embed timeout must be a duration like 2m")
	}
	return cfg, nil
}
```


## `internal/tui/index.go` (code, lines 186-198, score 0.454)

```go
File: internal/tui/index.go

package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwireman/archivist/internal/index"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Width(7)
	pathStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	phaseStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("177"))
)

type progressMsg index.Progress

type doneMsg struct{}

type model struct {
	ch       <-chan index.Progress

func ratio(done, total int) float64 {
	if total <= 0 {
		return 0
	}
	r := float64(done) / float64(total)
	if r > 1 {
		return 1
	}
	if r < 0 {
		return 0
	}
	return r
}
```


## `internal/tui/index.go` (code, lines 218-223, score 0.453)

```go
File: internal/tui/index.go

package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwireman/archivist/internal/index"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().Bold(true)
	labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Width(7)
	pathStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	mutedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	okStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
	helpStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	phaseStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("177"))
)

type progressMsg index.Progress

type doneMsg struct{}

type model struct {
	ch       <-chan index.Progress

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
```


## `internal/chunk/chunk_test.go` (comment, line 22, score 0.452)

```go
content := "package main\n\n// TODO: fix this\nfunc main() {}\n"
```


## `go.sum` (code, lines 58-77, score 0.451)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```
github.com/google/pprof v0.0.0-20260802141513-ef3492d7dac3/go.mod h1:jl5iWTm0/hd5PjEYEOuwAJ57L/CibdZfrqZ5XA5GrCk=
github.com/google/uuid v1.6.0 h1:NIvaJDMOsjHA8n1jAhLSgzrAzy1Hgr+hNrb57e+94F0=
github.com/google/uuid v1.6.0/go.mod h1:TIyPZe4MgqvfeYDBFedMoGGpEw/LqOeaOT+nhxU+yHo=
github.com/hashicorp/golang-lru/v2 v2.0.7 h1:a+bsQ5rvGLjzHuww6tVxozPZFVghXaHOwFs4luLUK2k=
github.com/hashicorp/golang-lru/v2 v2.0.7/go.mod h1:QeFd9opnmA6QUJc5vARoKUSoFhyfM2/ZepoAG6RGpeM=
github.com/inconshreveable/mousetrap v1.1.0 h1:wN+x4NVGpMsO7ErUn/mUI3vEoE6Jt13X2s0bqwp9tc8=
github.com/inconshreveable/mousetrap v1.1.0/go.mod h1:vpF70FUmC8bwa3OWnCshd2FqLfsEA9PFc4w1p2J65bw=
github.com/lucasb-eyer/go-colorful v1.3.0 h1:2/yBRLdWBZKrf7gB40FoiKfAWYQ0lqNcbuQwVHXptag=
github.com/lucasb-eyer/go-colorful v1.3.0/go.mod h1:R4dSotOR9KMtayYi1e77YzuveK+i7ruzyGqttikkLy0=
github.com/mattn/go-isatty v0.0.24 h1:tGZZoVgT/KiqK1c8ocVLeDS8BSWMRd47J3Lbz7vsReI=
github.com/mattn/go-isatty v0.0.24/go.mod h1:nMCL3Zebbrt45jsMDgnfIwz6ydEQApk5oEI3HqDio6A=
github.com/mattn/go-localereader v0.0.1 h1:ygSAOl7ZXTx4RdPYinUpg6W99U8jWvWi9Ye2JC/oIi4=
github.com/mattn/go-localereader v0.0.1/go.mod h1:8fBrzywKY7BI3czFoHkuzRoWE9C+EiG4R1k4Cjx5p88=
github.com/mattn/go-runewidth v0.0.19 h1:v++JhqYnZuu5jSKrk9RbgF5v4CGUjqRfBm05byFGLdw=
github.com/mattn/go-runewidth v0.0.19/go.mod h1:XBkDxAl56ILZc9knddidhrOlY5R/pDhgLpndooCuJAs=
github.com/mitchellh/hashstructure/v2 v2.0.2 h1:vGKWl0YJqUNxE8d+h8f6NJLcCJrgbhC4NcD46KavDd4=
github.com/mitchellh/hashstructure/v2 v2.0.2/go.mod h1:MG3aRVU/N29oo/V/IhBX8GR/zz4kQkprJgF2EVszyDE=
github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6 h1:ZK8zHtRHOkbHy6Mmr5D264iyp3TiX5OmNcI5cIARiQI=
github.com/muesli/ansi v0.0.0-20230316100256-276c6243b2f6/go.mod h1:CJlz5H+gyd6CUWT45Oy4q24RdLyn7Md9Vj2/ldJBSIo=
github.com/muesli/cancelreader v0.2.2 h1:3I4Kt4BQjOR54NavqnDogx/MIoWBFa0StPA8ELUXHmA=
github.com/muesli/cancelreader v0.2.2/go.mod h1:3XuTXfFS2VjM+HTLZY9Ak0l6eUKfijIfMUZ4EgX0QYo=
```

