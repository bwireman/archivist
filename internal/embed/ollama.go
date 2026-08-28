package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bwireman/archivist/internal/config"
)

type OllamaClient struct {
	baseURL    string
	embedModel string
	httpClient *http.Client
	dimensions int
}

func NewOllamaClient(baseURL, embedModel string) *OllamaClient {
	return NewOllamaClientWithTimeout(baseURL, embedModel, config.DefaultEmbedTimeout)
}

func NewOllamaClientFromConfig(cfg config.OllamaConfig) *OllamaClient {
	return NewOllamaClientWithTimeout(cfg.BaseURL, cfg.EmbedModel, cfg.EmbedTimeoutDuration())
}

func NewOllamaClientWithTimeout(baseURL, embedModel string, timeout time.Duration) *OllamaClient {
	return &OllamaClient{
		baseURL:    baseURL,
		embedModel: embedModel,
		httpClient: &http.Client{Timeout: timeout},
	}
}

func (c *OllamaClient) Healthy(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ollama unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ollama health check failed: %s", string(body))
	}
	return nil
}

type embedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type embedResponse struct {
	Embedding []float32 `json:"embedding"`
}

func (c *OllamaClient) Embed(ctx context.Context, text string) ([]float32, error) {
	body, err := json.Marshal(embedRequest{Model: c.embedModel, Prompt: text})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embed failed: %s", string(b))
	}
	var out embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Embedding) > 0 {
		c.dimensions = len(out.Embedding)
	}
	return out.Embedding, nil
}

func (c *OllamaClient) Dimensions() int {
	return c.dimensions
}

// FakeEmbedder is used in tests.
type FakeEmbedder struct {
	Dim int
}

func (f *FakeEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	dim := f.Dim
	if dim == 0 {
		dim = 8
	}
	vec := make([]float32, dim)
	var sum float32
	for i, r := range text {
		vec[i%dim] += float32(int(r)%97) / 97
		sum += vec[i%dim]
	}
	if sum > 0 {
		for i := range vec {
			vec[i] /= sum
		}
	}
	return vec, nil
}

func (f *FakeEmbedder) Dimensions() int {
	if f.Dim == 0 {
		return 8
	}
	return f.Dim
}

func (f *FakeEmbedder) Healthy(ctx context.Context) error { return nil }
