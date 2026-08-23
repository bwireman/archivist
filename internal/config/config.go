package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultConfigName          = ".archivist.json"
	DefaultDataDir             = ".archivist"
	DefaultIndexDB             = "index.db"
	DefaultEmbedTimeout        = 2 * time.Minute
	DefaultGenerateTimeout     = 30 * time.Minute
	DefaultEmbedTimeoutStr     = "2m"
	DefaultGenerateTimeoutStr  = "30m"
	DefaultProvider            = "ollama"
	DefaultCursorBaseURL       = "https://api.cursor.com"
	DefaultCursorModel         = "composer-2.5"
	DefaultCursorPollInterval  = 2 * time.Second
	DefaultCursorPollIntervalStr = "2s"
	CursorAPIKeyEnv            = "CURSOR_API_KEY"
)

type Config struct {
	Provider string       `json:"provider"`
	Ollama   OllamaConfig `json:"ollama"`
	Cursor   CursorConfig `json:"cursor"`
	Docs     DocsConfig   `json:"docs"`
	Index    IndexConfig  `json:"index"`
	Store    StoreConfig  `json:"store"`
}

type OllamaConfig struct {
	BaseURL         string `json:"base_url"`
	EmbedModel      string `json:"embed_model"`
	GenerateModel   string `json:"generate_model"`
	EmbedTimeout    string `json:"embed_timeout,omitempty"`
	GenerateTimeout string `json:"generate_timeout,omitempty"`
}

func (o OllamaConfig) EmbedTimeoutDuration() time.Duration {
	return parseTimeout(o.EmbedTimeout, DefaultEmbedTimeout)
}

func (o OllamaConfig) GenerateTimeoutDuration() time.Duration {
	return parseTimeout(o.GenerateTimeout, DefaultGenerateTimeout)
}

type CursorConfig struct {
	APIKey          string `json:"api_key,omitempty"`
	BaseURL         string `json:"base_url"`
	Model           string `json:"model"`
	GenerateTimeout string `json:"generate_timeout,omitempty"`
	PollInterval    string `json:"poll_interval,omitempty"`
}

func (c CursorConfig) APIKeyValue() string {
	if c.APIKey != "" {
		return c.APIKey
	}
	return os.Getenv(CursorAPIKeyEnv)
}

func (c CursorConfig) GenerateTimeoutDuration() time.Duration {
	return parseTimeout(c.GenerateTimeout, DefaultGenerateTimeout)
}

func (c CursorConfig) PollIntervalDuration() time.Duration {
	return parseTimeout(c.PollInterval, DefaultCursorPollInterval)
}

func (cfg *Config) UsesCursorGeneration() bool {
	return cfg.Provider == "cursor"
}

func parseTimeout(raw string, fallback time.Duration) time.Duration {
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return d
}

type DocsConfig struct {
	Root  string            `json:"root"`
	Pages map[string]string `json:"pages"`
}

type IndexConfig struct {
	SkipDirs  []string `json:"skip_dirs"`
	SkipGlobs []string `json:"skip_globs"`
	ADRPaths  []string `json:"adr_paths"`
}

type StoreConfig struct {
	Path string `json:"path"`
}

func Default() *Config {
	return &Config{
		Provider: DefaultProvider,
		Ollama: OllamaConfig{
			BaseURL:         "http://localhost:11434",
			EmbedModel:      "nomic-embed-text",
			GenerateModel:   "llama3.1",
			EmbedTimeout:    DefaultEmbedTimeoutStr,
			GenerateTimeout: DefaultGenerateTimeoutStr,
		},
		Cursor: CursorConfig{
			BaseURL:         DefaultCursorBaseURL,
			Model:           DefaultCursorModel,
			GenerateTimeout: DefaultGenerateTimeoutStr,
			PollInterval:    DefaultCursorPollIntervalStr,
		},
		Docs: DocsConfig{
			Root:  "docs",
			Pages: map[string]string{},
		},
		Index: IndexConfig{
			SkipDirs: []string{
				".git", "vendor", "node_modules", ".archivist",
			},
			SkipGlobs: []string{},
			ADRPaths: []string{
				"**/adr/**",
				"docs/decisions/**",
				"**/ADR*.md",
			},
		},
		Store: StoreConfig{
			Path: filepath.Join(DefaultDataDir, DefaultIndexDB),
		},
	}
}

func Load(repoRoot string) (*Config, error) {
	cfg := Default()
	path := filepath.Join(repoRoot, DefaultConfigName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Store.Path == "" {
		cfg.Store.Path = filepath.Join(DefaultDataDir, DefaultIndexDB)
	}
	if cfg.Provider == "" {
		cfg.Provider = DefaultProvider
	}
	if cfg.Cursor.BaseURL == "" {
		cfg.Cursor.BaseURL = DefaultCursorBaseURL
	}
	if cfg.Cursor.Model == "" {
		cfg.Cursor.Model = DefaultCursorModel
	}
	return cfg, nil
}

func Save(repoRoot string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')
	path := filepath.Join(repoRoot, DefaultConfigName)
	return os.WriteFile(path, data, 0o644)
}

func DataDir(repoRoot string) string {
	return filepath.Join(repoRoot, DefaultDataDir)
}

func StorePath(repoRoot string, cfg *Config) string {
	if filepath.IsAbs(cfg.Store.Path) {
		return cfg.Store.Path
	}
	return filepath.Join(repoRoot, cfg.Store.Path)
}
