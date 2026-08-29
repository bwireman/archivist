package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultConfigName         = ".archivist.json"
	DefaultDataDir            = ".archivist"
	DefaultIndexDB            = "index.db"
	DefaultEmbedTimeout       = 2 * time.Minute
	DefaultEmbedTimeoutStr    = "2m"
	DefaultDecisionsDir       = "docs/decisions"
	DefaultGlobalDecisionsDir = "docs/global-decisions"
	DefaultDumpDir            = "docs/dump"
	DefaultGlobalDB           = "global.db"
	UserGlobalPrefix          = "user"
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
	SkipDirs       []string  `json:"skip_dirs"`
	SkipGlobs      []string  `json:"skip_globs"`
	HonorGitignore bool      `json:"honor_gitignore"`
	ADR            ADRConfig `json:"adr"`
}

// ADRConfig is the globs that classify markdown as ADRs.
// Repo vs global is which list matches; global wins if both do.
type ADRConfig struct {
	Repo   []string `json:"repo"`
	Global []string `json:"global"`
}

type StoreConfig struct {
	Path       string `json:"path"`
	GlobalPath string `json:"global_path,omitempty"`
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
			SkipGlobs:      []string{},
			HonorGitignore: true,
			ADR: ADRConfig{
				Repo: []string{
					"docs/decisions/**",
					"**/adr/**",
					"**/ADR*.md",
				},
				Global: []string{
					"docs/global-decisions/**",
				},
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
	cfg.Index.applyADRDefaults(Default().Index.ADR)
	return cfg, nil
}

func (c *IndexConfig) applyADRDefaults(defaults ADRConfig) {
	if c.ADR.Repo == nil {
		c.ADR.Repo = defaults.Repo
	}
	if c.ADR.Global == nil {
		c.ADR.Global = defaults.Global
	}
}

// UnmarshalJSON accepts nested index.adr and the older adr_paths / global_adr_paths keys.
func (c *IndexConfig) UnmarshalJSON(data []byte) error {
	var w struct {
		SkipDirs       []string   `json:"skip_dirs"`
		SkipGlobs      []string   `json:"skip_globs"`
		HonorGitignore *bool      `json:"honor_gitignore"`
		ADR            *ADRConfig `json:"adr"`
		ADRPaths       []string   `json:"adr_paths"`
		GlobalADRPaths []string   `json:"global_adr_paths"`
	}
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	c.SkipDirs = w.SkipDirs
	c.SkipGlobs = w.SkipGlobs
	if w.HonorGitignore != nil {
		c.HonorGitignore = *w.HonorGitignore
	} else {
		c.HonorGitignore = true
	}
	c.ADR = ADRConfig{}
	if w.ADR != nil {
		c.ADR = *w.ADR
	}
	if c.ADR.Repo == nil && w.ADRPaths != nil {
		c.ADR.Repo = w.ADRPaths
	}
	if c.ADR.Global == nil && w.GlobalADRPaths != nil {
		c.ADR.Global = w.GlobalADRPaths
	}
	return nil
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

// ArchivistHome is ~/.archivist.
func ArchivistHome() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, DefaultDataDir)
}

// GlobalStorePath is the machine-wide ADR index. Empty store.global_path
// means ~/.archivist/global.db. A relative path is resolved under
// ~/.archivist/, not the repo, so it cannot become per-checkout.
func GlobalStorePath(cfg *Config) string {
	var p string
	if cfg != nil {
		p = strings.TrimSpace(cfg.Store.GlobalPath)
	}
	if filepath.IsAbs(p) {
		return p
	}
	home := ArchivistHome()
	if home == "" {
		return ""
	}
	if p == "" {
		return filepath.Join(home, DefaultGlobalDB)
	}
	resolved := filepath.Clean(filepath.Join(home, p))
	if !pathUnderDir(home, resolved) {
		return ""
	}
	return resolved
}

func pathUnderDir(dir, path string) bool {
	dir = filepath.Clean(dir)
	path = filepath.Clean(path)
	sep := string(filepath.Separator)
	return path == dir || strings.HasPrefix(path, dir+sep)
}

// UserDecisionsDir is ~/.archivist/decisions — user-global ADRs in the
// machine-wide global index, not the repo DB.
func UserDecisionsDir() string {
	home := ArchivistHome()
	if home == "" {
		return ""
	}
	return filepath.Join(home, "decisions")
}

// VirtualUserADRPath is the index path for a file under UserDecisionsDir.
func VirtualUserADRPath(rel string) string {
	rel = filepath.ToSlash(strings.TrimPrefix(rel, "/"))
	if rel == "" || rel == "." {
		rel = "unknown.md"
	}
	return UserGlobalPrefix + "/" + rel
}

// IsUserGlobalPath reports whether path is a virtual user-ADR path (user/…).
func IsUserGlobalPath(path string) bool {
	path = filepath.ToSlash(path)
	if path == UserGlobalPrefix {
		return true
	}
	return strings.HasPrefix(path, UserGlobalPrefix+"/")
}
