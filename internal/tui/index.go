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
	cancel   context.CancelFunc
	progress index.Progress
	spin     spinner.Model
	bar      progress.Model
	width    int
}

// RunIndex runs fn while showing a live index TUI. q or ctrl+c cancels ctx.
func RunIndex(ctx context.Context, fn func(context.Context, index.Reporter) error) (index.Progress, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch := make(chan index.Progress, 256)
	errCh := make(chan error, 1)

	go func() {
		err := fn(ctx, func(p index.Progress) {
			select {
			case ch <- p:
			case <-ctx.Done():
			}
		})
		close(ch)
		errCh <- err
	}()

	m := newModel(ch, cancel)
	final, err := tea.NewProgram(m, tea.WithAltScreen(), tea.WithContext(ctx)).Run()
	runErr := <-errCh
	if err != nil {
		return index.Progress{}, err
	}
	fm, ok := final.(model)
	if !ok {
		return index.Progress{}, runErr
	}
	if ctx.Err() != nil {
		return fm.progress, context.Canceled
	}
	if runErr != nil {
		return fm.progress, runErr
	}
	return fm.progress, nil
}

func newModel(ch <-chan index.Progress, cancel context.CancelFunc) model {
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("177"))
	bar := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(28),
		progress.WithoutPercentage(),
	)
	return model{
		ch:     ch,
		cancel: cancel,
		spin:   sp,
		bar:    bar,
		width:  80,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.listen(), m.spin.Tick)
}

func (m model) listen() tea.Cmd {
	return func() tea.Msg {
		p, ok := <-m.ch
		if !ok {
			return doneMsg{}
		}
		return progressMsg(p)
	}
}

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

func (m model) View() string {
	p := m.progress
	var b strings.Builder

	fmt.Fprintf(&b, "%s  %s %s\n\n",
		titleStyle.Render("archivist index"),
		m.spin.View(),
		phaseStyle.Render(p.Phase.String()),
	)

	switch p.Phase {
	case index.PhaseScan:
		fmt.Fprintf(&b, "%s %s  %d files\n", labelStyle.Render("scan"), m.bar.ViewAs(0), p.FilesTotal)
	case index.PhaseFiles, index.PhasePrune, index.PhaseDone:
		fmt.Fprintf(&b, "%s %s  %d/%d\n", labelStyle.Render("files"), m.bar.ViewAs(ratio(p.FilesSeen, p.FilesTotal)), p.FilesSeen, p.FilesTotal)
	default:
		fmt.Fprintf(&b, "%s %s  %d/%d\n", labelStyle.Render("files"), m.bar.ViewAs(1), p.FilesSeen, p.FilesTotal)
	}

	current := strings.TrimSpace(p.Path + "  " + p.Detail)
	if current != "" {
		fmt.Fprintf(&b, "        %s\n", pathStyle.Render(truncate(current, max(m.width-10, 20))))
	}

	fmt.Fprintln(&b)
	gitPct := 0.0
	if p.Phase == index.PhaseGit || p.Phase == index.PhaseDone {
		gitPct = ratio(p.CommitsNew, p.CommitsTotal)
		if p.CommitsTotal == 0 {
			gitPct = 1
		} else if p.Phase == index.PhaseGit {
			gitPct = ratio(max(p.CommitsNew, 0), p.CommitsTotal)
		}
	}
	fmt.Fprintf(&b, "%s %s  %d new / %d\n", labelStyle.Render("git"), m.bar.ViewAs(gitPct), p.CommitsNew, p.CommitsTotal)

	fmt.Fprintln(&b)
	fmt.Fprintf(&b, "  %s  %s  %s  %s\n\n",
		okStyle.Render(fmt.Sprintf("%d indexed", p.FilesIndexed)),
		mutedStyle.Render(fmt.Sprintf("%d unchanged", p.FilesUnchanged)),
		mutedStyle.Render(fmt.Sprintf("%d pruned", p.FilesRemoved)),
		mutedStyle.Render(fmt.Sprintf("%d embedded", p.ChunksEmbedded)),
	)
	fmt.Fprint(&b, helpStyle.Render("  q cancel"))
	return b.String()
}

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

func truncate(s string, n int) string {
	if n <= 1 || len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

// FormatSummary is the one-line result printed after the TUI (or instead of it).
func FormatSummary(p index.Progress, chunkCount int) string {
	if p.FilesIndexed == 0 && p.CommitsNew == 0 && p.FilesRemoved == 0 {
		return fmt.Sprintf("Index up to date (%d chunks)", chunkCount)
	}
	return fmt.Sprintf("Indexed %d %s, %d %s (%d chunks)",
		p.FilesIndexed, plural(p.FilesIndexed, "file", "files"),
		p.CommitsNew, plural(p.CommitsNew, "commit", "commits"),
		chunkCount)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
