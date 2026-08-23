package embed

import (
	"context"
	"fmt"
	"time"

	"github.com/bwireman/archivist/internal/config"
)

type Providers struct {
	Embedder  Embedder
	Generator Generator
}

type HealthStatus struct {
	Provider       string `json:"provider"`
	EmbedderOK     bool   `json:"embedder_ok"`
	EmbedderError  string `json:"embedder_error,omitempty"`
	GeneratorOK    bool   `json:"generator_ok"`
	GeneratorError string `json:"generator_error,omitempty"`
}

func NewProviders(cfg *config.Config) (Providers, error) {
	ollama := NewOllamaClientFromConfig(cfg.Ollama)

	switch cfg.Provider {
	case "cursor":
		cursor, err := NewCursorClient(cfg.Cursor)
		if err != nil {
			return Providers{}, err
		}
		return Providers{
			Embedder:  ollama,
			Generator: &cursorGenerator{cursor: cursor, timeout: cfg.Cursor.GenerateTimeoutDuration()},
		}, nil
	case "ollama", "":
		return Providers{
			Embedder:  ollama,
			Generator: ollama,
		}, nil
	default:
		return Providers{}, fmt.Errorf("unknown provider %q (use ollama or cursor)", cfg.Provider)
	}
}

func CheckHealth(ctx context.Context, cfg *config.Config) HealthStatus {
	status := HealthStatus{Provider: cfg.Provider}
	if status.Provider == "" {
		status.Provider = config.DefaultProvider
	}

	providers, err := NewProviders(cfg)
	if err != nil {
		status.GeneratorError = err.Error()
		return status
	}

	if err := checkEmbedder(ctx, providers.Embedder); err != nil {
		status.EmbedderError = err.Error()
	} else {
		status.EmbedderOK = true
	}

	if err := checkGenerator(ctx, providers.Generator); err != nil {
		status.GeneratorError = err.Error()
	} else {
		status.GeneratorOK = true
	}
	return status
}

func checkEmbedder(ctx context.Context, e Embedder) error {
	if h, ok := e.(interface{ Healthy(context.Context) error }); ok {
		return h.Healthy(ctx)
	}
	return nil
}

func checkGenerator(ctx context.Context, g Generator) error {
	if h, ok := g.(interface{ Healthy(context.Context) error }); ok {
		return h.Healthy(ctx)
	}
	return nil
}

type cursorGenerator struct {
	cursor  *CursorClient
	timeout time.Duration
}

func (g *cursorGenerator) Generate(ctx context.Context, prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, g.timeout)
	defer cancel()
	return g.cursor.Generate(ctx, prompt)
}

func (g *cursorGenerator) Healthy(ctx context.Context) error {
	return g.cursor.Healthy(ctx)
}
