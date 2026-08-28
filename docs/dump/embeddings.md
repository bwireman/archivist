# Archivist context dump

Retrieved chunks from a local code index. Use only facts present here.

- Query: ollama embeddings
- Chunks: 20

## `docs/decisions/001-use-qwen3-embedding.md` (adr, lines 1-5, score 0.744)

```md
# Use qwen3-embedding:0.6b for this repository

- Status: accepted
- Date: 2026-08-28
```


## `docs/decisions/001-use-qwen3-embedding.md` (adr, lines 9-11, score 0.735)

```md
## Decision
Pin `.archivist.json` to `qwen3-embedding:0.6b` on `http://localhost:11434`. Leave the tool default in `config.Default()` as `nomic-embed-text`.
```


## `.archivist.json` (code, lines 1-25, score 0.707)

```json
{
  "ollama": {
    "base_url": "http://localhost:11434",
    "embed_model": "qwen3-embedding:0.6b",
    "embed_timeout": "2m"
  },
  "index": {
    "skip_dirs": [
      ".git",
      "vendor",
      "node_modules",
      ".archivist"
    ],
    "skip_globs": [],
    "adr_paths": [
      "**/adr/**",
      "docs/decisions/**",
      "**/ADR*.md"
    ]
  },
  "store": {
    "path": ".archivist/index.db"
  }
}
```


## `docs/decisions/001-use-qwen3-embedding.md` (adr, lines 6-8, score 0.658)

```md
## Context
`archivist init` writes `nomic-embed-text` as the default Ollama embed model. This repository needed a model that is actually installed locally so `make index` can run.
```


## `87bb4a48dc76cfc76681db7fc61a58171700050f` (commit, no line range, score 0.644)

```
Commit: 87bb4a48dc76cfc76681db7fc61a58171700050f
Author: bwireman
Date: 2026-08-28T18:39:14-05:00
Subject: Pin this repo to qwen3-embedding:0.6b.

The local Ollama instance does not have nomic-embed-text; keep that as the init default and override it here.

Files:
.archivist.json
docs/decisions/001-use-qwen3-embedding.md
```


## `internal/config/testdata/full.json` (code, lines 1-31, score 0.640)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```json
{
  "ollama": {
    "base_url": "http://ollama.example:11434",
    "embed_model": "mxbai-embed-large",
    "embed_timeout": "5m"
  },
  "index": {
    "skip_dirs": [
      ".git",
      "vendor",
      "node_modules",
      ".archivist",
      "dist",
      "build"
    ],
    "skip_globs": [
      "*.min.js",
      "*.pb.go"
    ],
    "adr_paths": [
      "**/adr/**",
      "docs/decisions/**",
      "**/ADR*.md",
      "architecture/decisions/**"
    ]
  },
  "store": {
    "path": ".archivist/custom-index.db"
  }
}
```


## `.cursor/rules/archivist-dump.mdc` (doc, lines 1-5, score 0.614)

```
---
description: Dump indexed decisions into docs/dump for later LLM sessions
alwaysApply: true
---
```


## `internal/embed/ollama.go` (code, lines 103-120, score 0.578)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/embed/ollama.go

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
```


## `internal/embed/ollama.go` (code, lines 64-92, score 0.576)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/embed/ollama.go

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
	if len(out.Embedding) == 0 {
		return nil, fmt.Errorf("embed returned an empty vector")
	}
	c.dimensions = len(out.Embedding)
	return out.Embedding, nil
}
```


## `docs/decisions/001-use-qwen3-embedding.md` (adr, lines 12-16, score 0.546)

```md
## Consequences
- Indexing this repo works without pulling `nomic-embed-text`.
- Other repos created with `archivist init` still get `nomic-embed-text` unless they override it.
- Changing the embed model later requires a full reindex; existing vectors are not comparable across models.
```


## `internal/embed/ollama.go` (code, lines 122-127, score 0.539)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/embed/ollama.go

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

func (f *FakeEmbedder) Dimensions() int {
	if f.Dim == 0 {
		return 8
	}
	return f.Dim
}
```


## `internal/embed/ollama.go` (code, lines 60-62, score 0.536)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/embed/ollama.go

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

type embedResponse struct {
	Embedding []float32 `json:"embedding"`
}
```


## `internal/embed/ollama.go` (code, lines 26-28, score 0.527)

Blame: bwireman (7d00fd949b40aee2f1044597f67392458ce705a1)

```go
File: internal/embed/ollama.go

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

func NewOllamaClientFromConfig(cfg config.OllamaConfig) *OllamaClient {
	return NewOllamaClientWithTimeout(cfg.BaseURL, cfg.EmbedModel, cfg.EmbedTimeoutDuration())
}
```


## `internal/embed/ollama.go` (code, lines 30-36, score 0.523)

Blame: bwireman (0f8e04b32575af25336945a0a2df91593c91b8e9)

```go
File: internal/embed/ollama.go

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

func NewOllamaClientWithTimeout(baseURL, embedModel string, timeout time.Duration) *OllamaClient {
	return &OllamaClient{
		baseURL:    baseURL,
		embedModel: embedModel,
		httpClient: &http.Client{Timeout: timeout},
	}
}
```


## `internal/embed/ollama.go` (code, lines 15-20, score 0.523)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/embed/ollama.go

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

type OllamaClient struct {
	baseURL    string
	embedModel string
	httpClient *http.Client
	dimensions int
}
```


## `internal/embed/ollama.go` (code, lines 22-24, score 0.521)

Blame: bwireman (0f8e04b32575af25336945a0a2df91593c91b8e9)

```go
File: internal/embed/ollama.go

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

func NewOllamaClient(baseURL, embedModel string) *OllamaClient {
	return NewOllamaClientWithTimeout(baseURL, embedModel, config.DefaultEmbedTimeout)
}
```


## `internal/embed/ollama.go` (code, lines 38-53, score 0.518)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/embed/ollama.go

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
```


## `internal/embed/ollama.go` (code, lines 94-96, score 0.518)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/embed/ollama.go

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

func (c *OllamaClient) Dimensions() int {
	return c.dimensions
}
```


## `internal/embed/ollama.go` (code, lines 55-58, score 0.517)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/embed/ollama.go

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

type embedRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}
```


## `internal/embed/ollama.go` (code, lines 99-101, score 0.512)

Blame: bwireman (bbf2c5e85191e594b809f3ccd6d0c40dc6d620f8)

```go
File: internal/embed/ollama.go

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

type FakeEmbedder struct {
	Dim int
}
```

