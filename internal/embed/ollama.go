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
	baseURL       string
	embedModel    string
	generateModel string
	embedHTTP     *http.Client
	generateHTTP  *http.Client
	dimensions    int
}

func NewOllamaClient(baseURL, embedModel, generateModel string) *OllamaClient {
	return NewOllamaClientWithTimeouts(baseURL, embedModel, generateModel, config.DefaultEmbedTimeout, config.DefaultGenerateTimeout)
}

func NewOllamaClientFromConfig(cfg config.OllamaConfig) *OllamaClient {
	return NewOllamaClientWithTimeouts(
		cfg.BaseURL,
		cfg.EmbedModel,
		cfg.GenerateModel,
		cfg.EmbedTimeoutDuration(),
		cfg.GenerateTimeoutDuration(),
	)
}

func NewOllamaClientWithTimeouts(baseURL, embedModel, generateModel string, embedTimeout, generateTimeout time.Duration) *OllamaClient {
	return &OllamaClient{
		baseURL:       baseURL,
		embedModel:    embedModel,
		generateModel: generateModel,
		embedHTTP:     &http.Client{Timeout: embedTimeout},
		generateHTTP:  &http.Client{Timeout: generateTimeout},
	}
}

func (c *OllamaClient) Healthy(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/tags", nil)
	if err != nil {
		return err
	}
	resp, err := c.embedHTTP.Do(req)
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
	resp, err := c.embedHTTP.Do(req)
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

type generateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type generateResponse struct {
	Response string `json:"response"`
}

func (c *OllamaClient) Generate(ctx context.Context, prompt string) (string, error) {
	body, err := json.Marshal(generateRequest{
		Model:  c.generateModel,
		Prompt: prompt,
		Stream: false,
	})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.generateHTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("generate failed: %s", string(b))
	}
	var out generateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Response, nil
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

type FakeGenerator struct {
	Response string
}

func (f *FakeGenerator) Generate(ctx context.Context, prompt string) (string, error) {
	if f.Response != "" {
		return f.Response, nil
	}
	return "# Generated Doc\n\nContent based on prompt length: " + fmt.Sprint(len(prompt)), nil
}

func (f *FakeGenerator) Healthy(ctx context.Context) error { return nil }

func (f *FakeEmbedder) Healthy(ctx context.Context) error { return nil }
