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
	GlobalPath     string
	HonorGitignore bool
}

// FormFromConfig flattens cfg into form fields.
func FormFromConfig(cfg *config.Config) InitForm {
	if cfg == nil {
		cfg = config.Default()
	}
	return InitForm{
		BaseURL:        cfg.Ollama.BaseURL,
		EmbedModel:     cfg.Ollama.EmbedModel,
		EmbedTimeout:   cfg.Ollama.EmbedTimeout,
		SkipDirs:       JoinList(cfg.Index.SkipDirs),
		SkipGlobs:      JoinList(cfg.Index.SkipGlobs),
		ADRPaths:       JoinList(cfg.Index.ADR.Repo),
		GlobalADRPaths: JoinList(cfg.Index.ADR.Global),
		StorePath:      cfg.Store.Path,
		GlobalPath:     cfg.Store.GlobalPath,
		HonorGitignore: cfg.Index.HonorGitignore,
	}
}

// ConfigFromForm builds a Config from form fields. Lists are comma-separated.
func ConfigFromForm(form InitForm) (*config.Config, error) {
	cfg := config.Default()
	cfg.Ollama.BaseURL = strings.TrimSpace(form.BaseURL)
	cfg.Ollama.EmbedModel = strings.TrimSpace(form.EmbedModel)
	cfg.Ollama.EmbedTimeout = strings.TrimSpace(form.EmbedTimeout)
	cfg.Index.SkipDirs = SplitList(form.SkipDirs)
	cfg.Index.SkipGlobs = SplitList(form.SkipGlobs)
	cfg.Index.ADR.Repo = SplitList(form.ADRPaths)
	cfg.Index.ADR.Global = SplitList(form.GlobalADRPaths)
	cfg.Index.HonorGitignore = form.HonorGitignore
	cfg.Store.Path = strings.TrimSpace(form.StorePath)
	cfg.Store.GlobalPath = strings.TrimSpace(form.GlobalPath)

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

// JoinList formats items as a comma-separated string.
func JoinList(items []string) string {
	return strings.Join(items, ", ")
}

// SplitList splits a comma- or newline-separated list and trims blanks.
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

// RunInit shows a setup form seeded from cfg. Cancel returns context.Canceled.
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
				Suggestions([]string{"qwen3-embedding:0.6b", "nomic-embed-text", "mxbai-embed-large"}).
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
			huh.NewConfirm().
				Title("Honor .gitignore").
				Description("Skip files and directories listed in .gitignore").
				Value(&formVals.HonorGitignore),
			huh.NewInput().
				Title("Repo ADR globs").
				Description("index.adr.repo — docs/decisions/**").
				Value(&formVals.ADRPaths),
			huh.NewInput().
				Title("Global ADR globs").
				Description("index.adr.global — docs/global-decisions/**").
				Value(&formVals.GlobalADRPaths),
		).Title("Index").Description("What to walk, and which paths are ADRs"),
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

func required(name string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s is required", name)
		}
		return nil
	}
}

func validDuration(s string) error {
	d, err := time.ParseDuration(strings.TrimSpace(s))
	if err != nil || d <= 0 {
		return fmt.Errorf("use a duration like 2m")
	}
	return nil
}

// FormatInitSummary is printed after init writes config.
func FormatInitSummary(cfg *config.Config, existed bool) string {
	verb := "Created"
	if existed {
		verb = "Updated"
	}
	return fmt.Sprintf("%s .archivist.json and .archivist/\nEmbeddings use Ollama:\n  ollama pull %s\n", verb, cfg.Ollama.EmbedModel)
}
