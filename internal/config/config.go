package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultConfigName      = ".archivist.json"
	DefaultDataDir         = ".archivist"
	DefaultIndexDB         = "index.db"
	DefaultEmbedTimeout    = 2 * time.Minute
	DefaultEmbedTimeoutStr = "2m"
	DefaultDecisionsDir    = "docs/decisions"
	DefaultDumpDir         = "docs/dump"
)

type Config struct {
	Ollama OllamaConfig `json:"ollama"`
	Index  IndexConfig  `json:"index"`
	Store  StoreConfig  `json:"store"`
}

type OllamaConfig struct {
	BaseURL      string `json:"base_url"`
	EmbedModel   string `json:"embed_model"`
	EmbedTimeout string `json:"embed_timeout,omitempty"`
}

func (o OllamaConfig) EmbedTimeoutDuration() time.Duration {
	return parseTimeout(o.EmbedTimeout, DefaultEmbedTimeout)
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
		Ollama: OllamaConfig{
			BaseURL:      "http://localhost:11434",
			EmbedModel:   "nomic-embed-text",
			EmbedTimeout: DefaultEmbedTimeoutStr,
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
