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
	DefaultCommandsLog        = "commands.log"
	DefaultEmbedModel         = "qwen3-embedding:0.6b"
	DefaultEmbedTimeout       = 5 * time.Minute
	DefaultEmbedTimeoutStr    = "5m"
	DefaultOllamaURL          = "http://localhost:11434"
	DefaultDecisionsDir       = "docs/decisions"
	DefaultGlobalDecisionsDir = "docs/global-decisions" // in-repo layout when records.global is set relative to the checkout
	DefaultArchiveDir         = "docs/archive"
	DefaultGlobalDB           = "archive.db"
	UserGlobalPrefix          = "user"
	HomeGlobalPrefix          = "global"
)

type Config struct {
	Ollama      OllamaConfig  `json:"ollama"`
	Index       IndexConfig   `json:"index,omitempty"`
	Records     RecordsConfig `json:"records"`
	Publish     PublishConfig `json:"publish,omitempty"`
	LogCommands bool          `json:"log_commands,omitempty"`
}

type PublishConfig struct {
	Destinations map[string]PublishDestination `json:"destinations,omitempty"`
}

type PublishDestination struct {
	Command []string `json:"command"`
}

type OllamaConfig struct {
	BaseURL      string `json:"base_url"`
	EmbedModel   string `json:"embed_model"`
	EmbedTimeout string `json:"embed_timeout,omitempty"`
}

func (o OllamaConfig) EmbedTimeoutDuration() time.Duration {
	return parseTimeout(o.EmbedTimeout, DefaultEmbedTimeout)
}

// ResolvedBaseURL is ollama.base_url, else $OLLAMA_HOST, else localhost.
func (o OllamaConfig) ResolvedBaseURL() string {
	if u := strings.TrimSpace(o.BaseURL); u != "" {
		return normalizeOllamaURL(u)
	}
	if u := strings.TrimSpace(os.Getenv("OLLAMA_HOST")); u != "" {
		return normalizeOllamaURL(u)
	}
	return DefaultOllamaURL
}

func normalizeOllamaURL(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, "/")
	if s == "" {
		return DefaultOllamaURL
	}
	if strings.Contains(s, "://") {
		return s
	}
	return "http://" + s
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
	SkipGlobs []string `json:"skip_globs,omitempty"`
}

type RecordsConfig struct {
	Repo      string `json:"repo,omitempty"`
	Global    string `json:"global,omitempty"`
	Dev       string `json:"dev,omitempty"`
	Export    string `json:"export,omitempty"`
	WriteDocs bool   `json:"write_docs,omitempty"`
}

func (r RecordsConfig) withDefaults() RecordsConfig {
	if strings.TrimSpace(r.Repo) == "" {
		r.Repo = DefaultDecisionsDir
	}
	if strings.TrimSpace(r.Export) == "" {
		r.Export = DefaultArchiveDir
	}
	r.Repo = filepath.ToSlash(strings.Trim(r.Repo, "/"))
	r.Export = filepath.ToSlash(strings.Trim(r.Export, "/"))
	r.Global = strings.TrimSpace(r.Global)
	if r.GlobalInRepo() {
		r.Global = filepath.ToSlash(strings.Trim(r.Global, "/"))
	} else {
		r.Global = filepath.ToSlash(strings.TrimRight(r.Global, "/"))
	}
	return r
}

func expandHomePath(p string) (string, bool) {
	p = strings.TrimSpace(p)
	if p != "~" && !strings.HasPrefix(p, "~/") {
		return p, false
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return p, false
	}
	if p == "~" {
		return home, true
	}
	return filepath.Join(home, strings.TrimPrefix(p, "~/")), true
}

// GlobalInRepo reports whether records.global is a checkout-relative directory.
// Empty, absolute, and ~/… values live on the machine (default ~/.archivist).
func (r RecordsConfig) GlobalInRepo() bool {
	p := strings.TrimSpace(r.Global)
	if p == "" || filepath.IsAbs(p) {
		return false
	}
	if _, ok := expandHomePath(p); ok {
		return false
	}
	return true
}

// GlobalDir is the directory for scope=global records.
// Empty records.global is ~/.archivist.
func (r RecordsConfig) GlobalDir(repoRoot string) string {
	p := strings.TrimSpace(r.Global)
	if p == "" {
		return ArchivistHome()
	}
	if expanded, ok := expandHomePath(p); ok {
		return expanded
	}
	if filepath.IsAbs(p) {
		return p
	}
	if repoRoot == "" {
		return p
	}
	return filepath.Join(repoRoot, p)
}

func PathUnder(rel, dir string) bool {
	rel = filepath.ToSlash(rel)
	dir = filepath.ToSlash(strings.Trim(dir, "/"))
	if dir == "" {
		return false
	}
	return rel == dir || strings.HasPrefix(rel, dir+"/")
}

func Default() *Config {
	return &Config{
		Ollama: OllamaConfig{
			BaseURL:      DefaultOllamaURL,
			EmbedModel:   DefaultEmbedModel,
			EmbedTimeout: DefaultEmbedTimeoutStr,
		},
		Records: RecordsConfig{
			Repo:   DefaultDecisionsDir,
			Export: DefaultArchiveDir,
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
	if strings.TrimSpace(cfg.Ollama.BaseURL) == "" {
		cfg.Ollama.BaseURL = DefaultOllamaURL
	}
	if strings.TrimSpace(cfg.Ollama.EmbedModel) == "" {
		cfg.Ollama.EmbedModel = DefaultEmbedModel
	}
	if strings.TrimSpace(cfg.Ollama.EmbedTimeout) == "" {
		cfg.Ollama.EmbedTimeout = DefaultEmbedTimeoutStr
	}
	cfg.Records = cfg.Records.withDefaults()
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

func StorePath(repoRoot string) string {
	return filepath.Join(repoRoot, DefaultDataDir, DefaultIndexDB)
}

func CommandsLogPath(repoRoot string) string {
	return filepath.Join(repoRoot, DefaultDataDir, DefaultCommandsLog)
}

func ArchivistHome() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, DefaultDataDir)
}

func HomeStorePath() string {
	home := ArchivistHome()
	if home == "" {
		return ""
	}
	return filepath.Join(home, DefaultGlobalDB)
}

func UserRecordsDir() string {
	home := ArchivistHome()
	if home == "" {
		return ""
	}
	return filepath.Join(home, "records")
}

func (c *Config) DevRecordsDir() string {
	if c != nil {
		if p := strings.TrimSpace(c.Records.Dev); p != "" {
			if filepath.IsAbs(p) {
				return p
			}
			home := ArchivistHome()
			if home == "" {
				return p
			}
			return filepath.Join(home, p)
		}
	}
	return UserRecordsDir()
}

func VirtualUserADRPath(rel string) string {
	return prefixedHomePath(UserGlobalPrefix, rel)
}

func VirtualHomeGlobalPath(rel string) string {
	return prefixedHomePath(HomeGlobalPrefix, rel)
}

func prefixedHomePath(prefix, rel string) string {
	rel = filepath.ToSlash(strings.TrimPrefix(rel, "/"))
	if rel == "" || rel == "." {
		rel = "unknown.md"
	}
	return prefix + "/" + rel
}

func IsUserGlobalPath(path string) bool {
	return hasHomePrefix(path, UserGlobalPrefix)
}

func IsHomeGlobalPath(path string) bool {
	return hasHomePrefix(path, HomeGlobalPrefix)
}

func hasHomePrefix(path, prefix string) bool {
	path = filepath.ToSlash(path)
	if path == prefix {
		return true
	}
	return strings.HasPrefix(path, prefix+"/")
}
