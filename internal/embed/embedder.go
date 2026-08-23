package embed

import "context"

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	Dimensions() int
}

type Generator interface {
	Generate(ctx context.Context, prompt string) (string, error)
}

type Client interface {
	Embedder
	Generator
	Healthy(ctx context.Context) error
}
