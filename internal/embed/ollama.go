package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bwireman/archivist/internal/config"
)

type OllamaClient struct {
	baseURL    string
	embedModel string
	numCtx     int
	httpClient *http.Client
}

func NewOllamaClientFromConfig(cfg config.OllamaConfig) *OllamaClient {
	c := NewOllamaClientWithTimeout(cfg.ResolvedBaseURL(), cfg.EmbedModel, cfg.EmbedTimeoutDuration())
	c.numCtx = cfg.EmbedNumCtx
	return c
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
		return readAPIError(resp, "ollama health check failed")
	}
	return nil
}

type embedOptions struct {
	NumCtx int `json:"num_ctx,omitempty"`
}

type embedRequest struct {
	Model   string        `json:"model"`
	Prompt  string        `json:"prompt"`
	Options *embedOptions `json:"options,omitempty"`
}

type embedResponse struct {
	Embedding []float32 `json:"embedding"`
}

func (c *OllamaClient) Embed(ctx context.Context, text string) ([]float32, error) {
	reqBody := embedRequest{Model: c.embedModel, Prompt: text}
	if c.numCtx > 0 {
		reqBody.Options = &embedOptions{NumCtx: c.numCtx}
	}
	body, err := json.Marshal(reqBody)
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
		return nil, readAPIError(resp, "embed failed")
	}
	var out embedResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if len(out.Embedding) == 0 {
		return nil, errors.New("embed returned an empty vector")
	}
	return out.Embedding, nil
}

func readAPIError(resp *http.Response, prefix string) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s: %w", prefix, err)
	}
	return fmt.Errorf("%s: %s", prefix, string(body))
}

// FakeEmbedder is used in tests.
type FakeEmbedder struct {
	Dim int
}

func (f *FakeEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
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
