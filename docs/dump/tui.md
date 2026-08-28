# Archivist context dump

Retrieved chunks from a local code index. Use only facts present here.

- Query: init TUI huh setup form --plain
- Chunks: 20

## `docs/decisions/003-huh-init-tui.md` (adr, lines 1-5, score 0.699)

```md
# Use Huh for the init TUI

- Status: accepted
- Date: 2026-08-28
```


## `docs/decisions/003-huh-init-tui.md` (adr, lines 9-11, score 0.652)

```md
## Decision
Show a Huh form on a TTY so `archivist init` can edit Ollama, index, and store settings before writing `.archivist.json`. `--plain` writes `config.Default()` with no prompt.
```


## `internal/tui/init.go` (code, lines 15-23, score 0.616)

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
	BaseURL      string
	EmbedModel   string
	EmbedTimeout string
	SkipDirs     string
	SkipGlobs    string
	ADRPaths     string
	StorePath    string
}

// FormFromConfig flattens cfg into form fields.
func FormFromConfig(cfg *config.Config) InitForm {
	if cfg == nil {
		cfg = config.Default()
	}
	return InitForm{

type InitForm struct {
	BaseURL      string
	EmbedModel   string
	EmbedTimeout string
	SkipDirs     string
	SkipGlobs    string
	ADRPaths     string
	StorePath    string
}
```


## `internal/tui/init.go` (code, lines 74-84, score 0.614)

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
	BaseURL      string
	EmbedModel   string
	EmbedTimeout string
	SkipDirs     string
	SkipGlobs    string
	ADRPaths     string
	StorePath    string
}

// FormFromConfig flattens cfg into form fields.
func FormFromConfig(cfg *config.Config) InitForm {
	if cfg == nil {
		cfg = config.Default()
	}
	return InitForm{

func SplitList(s string) []string {
	s = strings.ReplaceAll(s, "\n", ",")
	out := []string{}
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
```


## `internal/tui/init.go` (code, lines 170-176, score 0.612)

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
	BaseURL      string
	EmbedModel   string
	EmbedTimeout string
	SkipDirs     string
	SkipGlobs    string
	ADRPaths     string
	StorePath    string
}

// FormFromConfig flattens cfg into form fields.
func FormFromConfig(cfg *config.Config) InitForm {
	if cfg == nil {
		cfg = config.Default()
	}
	return InitForm{

func FormatInitSummary(cfg *config.Config, existed bool) string {
	verb := "Created"
	if existed {
		verb = "Updated"
	}
	return fmt.Sprintf("%s .archivist.json and .archivist/\nEmbeddings use Ollama:\n  ollama pull %s\n", verb, cfg.Ollama.EmbedModel)
}
```


## `docs/decisions/002-bubbletea-index-tui.md` (adr, lines 9-11, score 0.612)

```md
## Decision
Show a Bubble Tea TUI when stdout is a TTY. Fall back to a one-line summary when it is not, or when `--plain` is set (`make index` uses `--plain`). The indexer reports progress through a callback so tests do not need a terminal.
```


## `internal/tui/init.go` (code, lines 42-66, score 0.611)

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
	BaseURL      string
	EmbedModel   string
	EmbedTimeout string
	SkipDirs     string
	SkipGlobs    string
	ADRPaths     string
	StorePath    string
}

// FormFromConfig flattens cfg into form fields.
func FormFromConfig(cfg *config.Config) InitForm {
	if cfg == nil {
		cfg = config.Default()
	}
	return InitForm{

func ConfigFromForm(form InitForm) (*config.Config, error) {
	cfg := config.Default()
	cfg.Ollama.BaseURL = strings.TrimSpace(form.BaseURL)
	cfg.Ollama.EmbedModel = strings.TrimSpace(form.EmbedModel)
	cfg.Ollama.EmbedTimeout = strings.TrimSpace(form.EmbedTimeout)
	cfg.Index.SkipDirs = SplitList(form.SkipDirs)
	cfg.Index.SkipGlobs = SplitList(form.SkipGlobs)
	cfg.Index.ADRPaths = SplitList(form.ADRPaths)
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


## `internal/tui/init.go` (code, lines 26-39, score 0.606)

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
	BaseURL      string
	EmbedModel   string
	EmbedTimeout string
	SkipDirs     string
	SkipGlobs    string
	ADRPaths     string
	StorePath    string
}

// FormFromConfig flattens cfg into form fields.
func FormFromConfig(cfg *config.Config) InitForm {
	if cfg == nil {
		cfg = config.Default()
	}
	return InitForm{

func FormFromConfig(cfg *config.Config) InitForm {
	if cfg == nil {
		cfg = config.Default()
	}
	return InitForm{
		BaseURL:      cfg.Ollama.BaseURL,
		EmbedModel:   cfg.Ollama.EmbedModel,
		EmbedTimeout: cfg.Ollama.EmbedTimeout,
		SkipDirs:     JoinList(cfg.Index.SkipDirs),
		SkipGlobs:    JoinList(cfg.Index.SkipGlobs),
		ADRPaths:     JoinList(cfg.Index.ADRPaths),
		StorePath:    cfg.Store.Path,
	}
}
```


## `internal/tui/init.go` (code, lines 152-159, score 0.592)

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
	BaseURL      string
	EmbedModel   string
	EmbedTimeout string
	SkipDirs     string
	SkipGlobs    string
	ADRPaths     string
	StorePath    string
}

// FormFromConfig flattens cfg into form fields.
func FormFromConfig(cfg *config.Config) InitForm {
	if cfg == nil {
		cfg = config.Default()
	}
	return InitForm{

func required(name string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s is required", name)
		}
		return nil
	}
}
```


## `internal/tui/init.go` (code, lines 69-71, score 0.592)

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
	BaseURL      string
	EmbedModel   string
	EmbedTimeout string
	SkipDirs     string
	SkipGlobs    string
	ADRPaths     string
	StorePath    string
}

// FormFromConfig flattens cfg into form fields.
func FormFromConfig(cfg *config.Config) InitForm {
	if cfg == nil {
		cfg = config.Default()
	}
	return InitForm{

func JoinList(items []string) string {
	return strings.Join(items, ", ")
}
```


## `internal/tui/init.go` (code, lines 161-167, score 0.589)

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
	BaseURL      string
	EmbedModel   string
	EmbedTimeout string
	SkipDirs     string
	SkipGlobs    string
	ADRPaths     string
	StorePath    string
}

// FormFromConfig flattens cfg into form fields.
func FormFromConfig(cfg *config.Config) InitForm {
	if cfg == nil {
		cfg = config.Default()
	}
	return InitForm{

func validDuration(s string) error {
	d, err := time.ParseDuration(strings.TrimSpace(s))
	if err != nil || d <= 0 {
		return fmt.Errorf("use a duration like 2m")
	}
	return nil
}
```


## `docs/decisions/002-bubbletea-index-tui.md` (adr, lines 1-5, score 0.581)

```md
# Use Bubble Tea for the index TUI

- Status: accepted
- Date: 2026-08-28
```


## `internal/tui/init.go` (code, lines 87-150, score 0.574)

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
	BaseURL      string
	EmbedModel   string
	EmbedTimeout string
	SkipDirs     string
	SkipGlobs    string
	ADRPaths     string
	StorePath    string
}

// FormFromConfig flattens cfg into form fields.
func FormFromConfig(cfg *config.Config) InitForm {
	if cfg == nil {
		cfg = config.Default()
	}
	return InitForm{

func RunInit(ctx context.Context, seed *config.Config) (*config.Config, error) {
	formVals := FormFromConfig(seed)
	confirm := true

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Ollama URL").
				Description("Local Ollama API").
				Value(&formVals.BaseURL).
				Validate(required("Ollama URL")),
			huh.NewInput().
				Title("Embed model").
				Description("Must be pulled in Ollama").
				Suggestions([]string{"nomic-embed-text", "qwen3-embedding:0.6b", "mxbai-embed-large"}).
				Value(&formVals.EmbedModel).
				Validate(required("embed model")),
			huh.NewInput().
				Title("Embed timeout").
				Description("Go duration, e.g. 2m").
				Value(&formVals.EmbedTimeout).
				Validate(validDuration),
		).Title("Ollama").Description("How Archivist embeds chunks"),
		huh.NewGroup(
			huh.NewInput().
				Title("Skip directories").
				Description("Comma-separated directory names").
				Value(&formVals.SkipDirs),
			huh.NewInput().
				Title("Skip globs").
				Description("Comma-separated, e.g. *.pb.go").
				Value(&formVals.SkipGlobs),
			huh.NewInput().
				Title("ADR paths").
				Description("Comma-separated globs").
				Value(&formVals.ADRPaths),
		).Title("Index").Description("What to walk and how to classify ADRs"),
		huh.NewGroup(
			huh.NewInput().
				Title("Store path").
				Description("Relative to the repo, or absolute").
				Value(&formVals.StorePath).
				Validate(required("store path")),
		).Title("Store"),
		huh.NewGroup(
			huh.NewConfirm().
				Title("Write .archivist.json?").
				Affirmative("Write").
				Negative("Cancel").
				Value(&confirm),
		),
	).WithTheme(huh.ThemeCharm()).WithShowHelp(true)

	if err := form.RunWithContext(ctx); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			return nil, context.Canceled
		}
		return nil, err
	}
	if !confirm {
		return nil, context.Canceled
	}
	return ConfigFromForm(formVals)
}
```


## `docs/decisions/003-huh-init-tui.md` (adr, lines 6-8, score 0.507)

```md
## Context
`archivist init` wrote `config.Default()` with no prompts. The embed model is a real choice (`nomic-embed-text` vs a locally installed model). The index command already uses a Charm TUI on a TTY.
```


## `internal/tui/init_test.go` (code, lines 73-83, score 0.471)

```go
File: internal/tui/init_test.go

package tui

import (
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/config"
)

func TestSplitJoinList(t *testing.T) {
	items := []string{".git", "vendor", "node_modules"}
	if got := SplitList(JoinList(items)); strings.Join(got, ",") != strings.Join(items, ",") {
		t.Fatalf("round-trip: %v", got)
	}
	got := SplitList(" *.pb.go , foo.go\n bar.go ")
	if strings.Join(got, ",") != "*.pb.go,foo.go,bar.go" {
		t.Fatalf("split mixed: %v", got)
	}
	if len(SplitList("")) != 0 {
		t.Fatalf("empty: %v", SplitList(""))
	}
}

func TestConfigFromFormRoundTrip(t *testing.T) {
	want := config.Default()
	got, err := ConfigFromForm(FormFromConfig(want))
	if err != nil {
		t.Fatal(err)
	}
	if got.Ollama.BaseURL != want.Ollama.BaseURL || got.Ollama.EmbedModel != want.Ollama.EmbedModel {

func TestFormatInitSummary(t *testing.T) {
	cfg := config.Default()
	created := FormatInitSummary(cfg, false)
	if !strings.Contains(created, "Created .archivist.json") || !strings.Contains(created, "ollama pull nomic-embed-text") {
		t.Fatalf("created:\n%s", created)
	}
	updated := FormatInitSummary(cfg, true)
	if !strings.Contains(updated, "Updated .archivist.json") {
		t.Fatalf("updated:\n%s", updated)
	}
}
```


## `internal/cmd/init.go` (code, lines 16-58, score 0.457)

```go
File: internal/cmd/init.go

package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/tui"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var plain bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize archivist config and data directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := repoRoot()
			if err != nil {
				return err
			}
			existed := configExists(root)
			cfg := config.Default()
			if existed {
				loaded, err := config.Load(root)
				if err != nil {

func newInitCmd() *cobra.Command {
	var plain bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize archivist config and data directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := repoRoot()
			if err != nil {
				return err
			}
			existed := configExists(root)
			cfg := config.Default()
			if existed {
				loaded, err := config.Load(root)
				if err != nil {
					return err
				}
				cfg = loaded
			}

			useTUI := !plain && isatty.IsTerminal(os.Stdout.Fd())
			if useTUI {
				cfg, err = tui.RunInit(cmd.Context(), cfg)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						return fmt.Errorf("init cancelled")
					}
					return err
				}
			} else {
				cfg = config.Default()
			}

			if err := applyInit(root, cfg); err != nil {
				return err
			}
			fmt.Print(tui.FormatInitSummary(cfg, existed))
			return nil
		},
	}
	cmd.Flags().BoolVar(&plain, "plain", false, "write default config without the setup TUI")
	return cmd
}
```


## `internal/tui/init_test.go` (code, lines 24-39, score 0.453)

```go
File: internal/tui/init_test.go

package tui

import (
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/config"
)

func TestSplitJoinList(t *testing.T) {
	items := []string{".git", "vendor", "node_modules"}
	if got := SplitList(JoinList(items)); strings.Join(got, ",") != strings.Join(items, ",") {
		t.Fatalf("round-trip: %v", got)
	}
	got := SplitList(" *.pb.go , foo.go\n bar.go ")
	if strings.Join(got, ",") != "*.pb.go,foo.go,bar.go" {
		t.Fatalf("split mixed: %v", got)
	}
	if len(SplitList("")) != 0 {
		t.Fatalf("empty: %v", SplitList(""))
	}
}

func TestConfigFromFormRoundTrip(t *testing.T) {
	want := config.Default()
	got, err := ConfigFromForm(FormFromConfig(want))
	if err != nil {
		t.Fatal(err)
	}
	if got.Ollama.BaseURL != want.Ollama.BaseURL || got.Ollama.EmbedModel != want.Ollama.EmbedModel {

func TestConfigFromFormRoundTrip(t *testing.T) {
	want := config.Default()
	got, err := ConfigFromForm(FormFromConfig(want))
	if err != nil {
		t.Fatal(err)
	}
	if got.Ollama.BaseURL != want.Ollama.BaseURL || got.Ollama.EmbedModel != want.Ollama.EmbedModel {
		t.Fatalf("ollama mismatch: %#v", got.Ollama)
	}
	if strings.Join(got.Index.SkipDirs, ",") != strings.Join(want.Index.SkipDirs, ",") {
		t.Fatalf("skip dirs: %v", got.Index.SkipDirs)
	}
	if got.Store.Path != want.Store.Path {
		t.Fatalf("store: %s", got.Store.Path)
	}
}
```


## `internal/tui/init_test.go` (code, lines 49-71, score 0.452)

```go
File: internal/tui/init_test.go

package tui

import (
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/config"
)

func TestSplitJoinList(t *testing.T) {
	items := []string{".git", "vendor", "node_modules"}
	if got := SplitList(JoinList(items)); strings.Join(got, ",") != strings.Join(items, ",") {
		t.Fatalf("round-trip: %v", got)
	}
	got := SplitList(" *.pb.go , foo.go\n bar.go ")
	if strings.Join(got, ",") != "*.pb.go,foo.go,bar.go" {
		t.Fatalf("split mixed: %v", got)
	}
	if len(SplitList("")) != 0 {
		t.Fatalf("empty: %v", SplitList(""))
	}
}

func TestConfigFromFormRoundTrip(t *testing.T) {
	want := config.Default()
	got, err := ConfigFromForm(FormFromConfig(want))
	if err != nil {
		t.Fatal(err)
	}
	if got.Ollama.BaseURL != want.Ollama.BaseURL || got.Ollama.EmbedModel != want.Ollama.EmbedModel {

func TestConfigFromFormCustom(t *testing.T) {
	got, err := ConfigFromForm(InitForm{
		BaseURL:      "http://ollama.example:11434",
		EmbedModel:   "qwen3-embedding:0.6b",
		EmbedTimeout: "5m",
		SkipDirs:     ".git, vendor",
		SkipGlobs:    "*.pb.go",
		ADRPaths:     "docs/decisions/**",
		StorePath:    ".archivist/custom.db",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Ollama.EmbedModel != "qwen3-embedding:0.6b" {
		t.Fatalf("model: %s", got.Ollama.EmbedModel)
	}
	if got.Store.Path != ".archivist/custom.db" {
		t.Fatalf("store: %s", got.Store.Path)
	}
	if strings.Join(got.Index.SkipGlobs, ",") != "*.pb.go" {
		t.Fatalf("globs: %v", got.Index.SkipGlobs)
	}
}
```


## `internal/tui/index.go` (code, lines 93-95, score 0.450)

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


## `docs/decisions/003-huh-init-tui.md` (adr, lines 12-16, score 0.447)

```md
## Consequences
- Interactive setup can set the embed model without editing JSON.
- Scripts and `--plain` still get the tool defaults.
- Huh is an additional Charm dependency.
```

