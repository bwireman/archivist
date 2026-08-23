package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/config"
)

type CursorClient struct {
	apiKey       string
	baseURL      string
	model        string
	httpClient   *http.Client
	pollInterval time.Duration
}

func NewCursorClient(cfg config.CursorConfig) (*CursorClient, error) {
	apiKey := cfg.APIKeyValue()
	if apiKey == "" {
		return nil, fmt.Errorf("cursor api key not set (config cursor.api_key or %s)", config.CursorAPIKeyEnv)
	}
	return &CursorClient{
		apiKey:       apiKey,
		baseURL:      strings.TrimRight(cfg.BaseURL, "/"),
		model:        cfg.Model,
		httpClient:   &http.Client{Timeout: 60 * time.Second},
		pollInterval: cfg.PollIntervalDuration(),
	}, nil
}

func (c *CursorClient) Healthy(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/me", nil)
	if err != nil {
		return err
	}
	c.setAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cursor api unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cursor health check failed (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *CursorClient) Generate(ctx context.Context, prompt string) (string, error) {
	deadline, hasDeadline := ctx.Deadline()
	if !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.DefaultGenerateTimeout)
		defer cancel()
		deadline, _ = ctx.Deadline()
	}

	agentID, runID, err := c.createAgent(ctx, prompt)
	if err != nil {
		return "", err
	}
	defer func() { _ = c.archiveAgent(context.Background(), agentID) }()

	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		run, err := c.getRun(ctx, agentID, runID)
		if err != nil {
			return "", err
		}
		switch run.Status {
		case "FINISHED":
			if strings.TrimSpace(run.Result) == "" {
				return "", fmt.Errorf("cursor run finished without result text")
			}
			return run.Result, nil
		case "ERROR", "CANCELLED", "EXPIRED":
			return "", fmt.Errorf("cursor run %s", strings.ToLower(run.Status))
		}
		if time.Now().After(deadline) {
			return "", fmt.Errorf("cursor generation timed out")
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(c.pollInterval):
		}
	}
}

type cursorCreateRequest struct {
	Prompt cursorPrompt      `json:"prompt"`
	Model  *cursorModel      `json:"model,omitempty"`
	Name   string            `json:"name,omitempty"`
}

type cursorPrompt struct {
	Text string `json:"text"`
}

type cursorModel struct {
	ID string `json:"id"`
}

type cursorCreateResponse struct {
	Agent cursorAgent `json:"agent"`
	Run   cursorRun   `json:"run"`
}

type cursorAgent struct {
	ID string `json:"id"`
}

type cursorRun struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Result string `json:"result,omitempty"`
}

func (c *CursorClient) createAgent(ctx context.Context, prompt string) (string, string, error) {
	body, err := json.Marshal(cursorCreateRequest{
		Prompt: cursorPrompt{Text: prompt},
		Model:  &cursorModel{ID: c.model},
		Name:   "archivist-doc-generation",
	})
	if err != nil {
		return "", "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/agents", bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	c.setAuth(req)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return "", "", fmt.Errorf("cursor create agent failed (%d): %s", resp.StatusCode, string(respBody))
	}
	var out cursorCreateResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		return "", "", err
	}
	if out.Agent.ID == "" || out.Run.ID == "" {
		return "", "", fmt.Errorf("cursor create agent returned incomplete response")
	}
	return out.Agent.ID, out.Run.ID, nil
}

func (c *CursorClient) getRun(ctx context.Context, agentID, runID string) (cursorRun, error) {
	url := fmt.Sprintf("%s/v1/agents/%s/runs/%s", c.baseURL, agentID, runID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return cursorRun{}, err
	}
	c.setAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return cursorRun{}, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		return cursorRun{}, fmt.Errorf("cursor get run failed (%d): %s", resp.StatusCode, string(respBody))
	}
	var run cursorRun
	if err := json.Unmarshal(respBody, &run); err != nil {
		return cursorRun{}, err
	}
	return run, nil
}

func (c *CursorClient) archiveAgent(ctx context.Context, agentID string) error {
	url := fmt.Sprintf("%s/v1/agents/%s/archive", c.baseURL, agentID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return err
	}
	c.setAuth(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("cursor archive agent failed (%d): %s", resp.StatusCode, string(body))
	}
	return nil
}

func (c *CursorClient) setAuth(req *http.Request) {
	req.SetBasicAuth(c.apiKey, "")
}
