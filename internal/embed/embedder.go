package embed

import (
	"context"

	"github.com/bwireman/archivist/internal/config"
)

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Dimensions() int
}

func OptionalFromConfig(ctx context.Context, cfg config.OllamaConfig) Embedder {
	client := NewOllamaClientFromConfig(cfg)
	if err := client.Healthy(ctx); err != nil {
		return nil
	}
	return client
}

type HealthStatus struct {
	EmbedderOK    bool   `json:"embedder_ok"`
	EmbedderError string `json:"embedder_error,omitempty"`
}

func CheckHealth(ctx context.Context, cfg *config.Config) HealthStatus {
	var status HealthStatus
	client := NewOllamaClientFromConfig(cfg.Ollama)
	if err := client.Healthy(ctx); err != nil {
		status.EmbedderError = err.Error()
	} else {
		status.EmbedderOK = true
	}
	return status
}
