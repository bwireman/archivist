// Package config loads and saves ~/.archivist/config.json.
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
	DisabledMarker            = ".archivist-disabled"
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
	Index       IndexConfig   `json:"index,omitzero"`
	Records     RecordsConfig `json:"records"`
	Publish     PublishConfig `json:"publish,omitzero"`
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
	EmbedNumCtx  int    `json:"embed_num_ctx,omitempty"`
	CriticModel  string `json:"critic_model,omitempty"`
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
	Repo      string `json:"repo,omitempty"`       // import drop folder + logical source_path prefix (default docs/decisions)
	Global    string `json:"global,omitempty"`     // empty => ~/.archivist; checkout-relative is an in-repo import folder
	Dev       string `json:"dev,omitempty"`        // import drop folder for dev scope (default ~/.archivist/records)
	Export    string `json:"export,omitempty"`     // generated tree when WriteDocs (default docs/archive)
	WriteDocs bool   `json:"write_docs,omitempty"` // if true, archivist export writes Export
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

// Canonical returns path as a cleaned absolute path with symlinks resolved.
// A path that does not exist yet is returned cleaned and absolute so init can create it.
func Canonical(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		if os.IsNotExist(err) {
			return abs, nil
		}
		return "", err
	}
	return resolved, nil
}

// CheckoutAliases is path plus its canonical path when those differ.
func CheckoutAliases(path string) []string {
	if path == "" {
		return nil
	}
	out := []string{path}
	if canon, err := Canonical(path); err == nil && canon != path {
		out = append(out, canon)
	}
	return out
}

// SameCheckout reports whether two checkout paths are the same after canonicalization.
func SameCheckout(a, b string) bool {
	if a == b {
		return true
	}
	ca, errA := Canonical(a)
	cb, errB := Canonical(b)
	if errA != nil || errB != nil {
		return false
	}
	return ca == cb
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
	cfg := &Config{}
	globalPath := GlobalConfigPath()
	legacyShape := false
	if globalPath != "" {
		data, err := os.ReadFile(globalPath)
		if err == nil {
			legacyShape, err = applyConfigBytes(cfg, data)
			if err != nil {
				return nil, err
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}
	locals, err := readLocalConfigs(repoRoot)
	if err != nil {
		return nil, err
	}
	for _, local := range locals {
		extra := &Config{}
		if _, err := applyConfigBytes(extra, local.data); err != nil {
			return nil, err
		}
		mergeMissing(cfg, extra)
	}
	cfg = finishLoad(cfg)
	if globalPath != "" && (legacyShape || len(locals) > 0) {
		if err := writeConfigFile(globalPath, cfg); err != nil {
			if !os.IsPermission(err) {
				return nil, fmt.Errorf("write config: %w", err)
			}
		} else if err := removeConfigs(locals); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

// applyConfigBytes overlays data onto cfg. A top-level Ollama object is the
// old ~/.archivist/config.json shape.
func applyConfigBytes(cfg *Config, data []byte) (legacyOllama bool, err error) {
	legacyOllama, err = isLegacyOllama(data)
	if err != nil {
		return false, fmt.Errorf("parse config: %w", err)
	}
	if legacyOllama {
		var ollama OllamaConfig
		if err := json.Unmarshal(data, &ollama); err != nil {
			return true, fmt.Errorf("parse config: %w", err)
		}
		cfg.Ollama = ollama
		return true, nil
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return false, fmt.Errorf("parse config: %w", err)
	}
	return false, nil
}

func isLegacyOllama(data []byte) (bool, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return false, err
	}
	for _, key := range []string{"ollama", "records", "index", "publish", "log_commands"} {
		if _, ok := raw[key]; ok {
			return false, nil
		}
	}
	for _, key := range []string{"base_url", "embed_model", "embed_timeout", "embed_num_ctx"} {
		if _, ok := raw[key]; ok {
			return true, nil
		}
	}
	return false, nil
}

type localConfig struct {
	path string
	data []byte
}

func localConfigPaths(repoRoot string) []string {
	canon, err := Canonical(repoRoot)
	if err != nil || strings.TrimSpace(canon) == "" {
		canon = repoRoot
	}
	var paths []string
	if dir := CheckoutDir(canon); dir != "" {
		paths = append(paths, filepath.Join(dir, "config.json"))
	}
	if strings.TrimSpace(canon) != "" {
		paths = append(paths, filepath.Join(canon, DefaultConfigName))
	}
	return paths
}

func readLocalConfigs(repoRoot string) ([]localConfig, error) {
	var out []localConfig
	for _, path := range localConfigPaths(repoRoot) {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read config: %w", err)
		}
		out = append(out, localConfig{path: path, data: data})
	}
	return out, nil
}

func removeConfigs(files []localConfig) error {
	for _, file := range files {
		if err := os.Remove(file.path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// mergeMissing copies set fields from src when dst does not have them.
// Empty strings, a zero embed_num_ctx, and false do not replace a set value.
func mergeMissing(dst, src *Config) {
	if dst == nil || src == nil {
		return
	}
	if strings.TrimSpace(dst.Ollama.BaseURL) == "" && strings.TrimSpace(src.Ollama.BaseURL) != "" {
		dst.Ollama.BaseURL = src.Ollama.BaseURL
	}
	if strings.TrimSpace(dst.Ollama.EmbedModel) == "" && strings.TrimSpace(src.Ollama.EmbedModel) != "" {
		dst.Ollama.EmbedModel = src.Ollama.EmbedModel
	}
	if strings.TrimSpace(dst.Ollama.CriticModel) == "" && strings.TrimSpace(src.Ollama.CriticModel) != "" {
		dst.Ollama.CriticModel = src.Ollama.CriticModel
	}
	if strings.TrimSpace(dst.Ollama.EmbedTimeout) == "" && strings.TrimSpace(src.Ollama.EmbedTimeout) != "" {
		dst.Ollama.EmbedTimeout = src.Ollama.EmbedTimeout
	}
	if dst.Ollama.EmbedNumCtx <= 0 && src.Ollama.EmbedNumCtx > 0 {
		dst.Ollama.EmbedNumCtx = src.Ollama.EmbedNumCtx
	}
	if len(dst.Index.SkipGlobs) == 0 && len(src.Index.SkipGlobs) > 0 {
		dst.Index.SkipGlobs = append([]string(nil), src.Index.SkipGlobs...)
	}
	if strings.TrimSpace(dst.Records.Repo) == "" && strings.TrimSpace(src.Records.Repo) != "" {
		dst.Records.Repo = src.Records.Repo
	}
	if strings.TrimSpace(dst.Records.Global) == "" && strings.TrimSpace(src.Records.Global) != "" {
		dst.Records.Global = src.Records.Global
	}
	if strings.TrimSpace(dst.Records.Dev) == "" && strings.TrimSpace(src.Records.Dev) != "" {
		dst.Records.Dev = src.Records.Dev
	}
	if strings.TrimSpace(dst.Records.Export) == "" && strings.TrimSpace(src.Records.Export) != "" {
		dst.Records.Export = src.Records.Export
	}
	if !dst.Records.WriteDocs && src.Records.WriteDocs {
		dst.Records.WriteDocs = true
	}
	if len(dst.Publish.Destinations) == 0 && len(src.Publish.Destinations) > 0 {
		dst.Publish.Destinations = src.Publish.Destinations
	}
	if !dst.LogCommands && src.LogCommands {
		dst.LogCommands = true
	}
}

func finishLoad(cfg *Config) *Config {
	if strings.TrimSpace(cfg.Ollama.EmbedModel) == "" {
		cfg.Ollama.EmbedModel = DefaultEmbedModel
	}
	if strings.TrimSpace(cfg.Ollama.EmbedTimeout) == "" {
		cfg.Ollama.EmbedTimeout = DefaultEmbedTimeoutStr
	}
	cfg.Records = cfg.Records.withDefaults()
	return cfg
}

func HasConfig(root string) bool {
	if path := GlobalConfigPath(); path != "" {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	for _, path := range localConfigPaths(root) {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

func Save(repoRoot string, cfg *Config) error {
	path := GlobalConfigPath()
	if path == "" {
		if strings.TrimSpace(repoRoot) == "" {
			return fmt.Errorf("cannot resolve config path (set $HOME)")
		}
		return writeConfigFile(filepath.Join(repoRoot, DefaultConfigName), cfg)
	}
	if err := writeConfigFile(path, cfg); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	for _, local := range localConfigPaths(repoRoot) {
		if local == "" || local == path {
			continue
		}
		if err := os.Remove(local); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func writeConfigFile(path string, cfg *Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func DataDir(repoRoot string) string {
	return filepath.Join(repoRoot, DefaultDataDir)
}

// DisabledBy returns the marker file in path or an ancestor, or "" when none.
func DisabledBy(path string) string {
	dir := path
	for {
		marker := filepath.Join(dir, DisabledMarker)
		if _, err := os.Stat(marker); err == nil {
			return marker
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// StorePath is the one archive database, ~/.archivist/archive.db.
// repoRoot is unused; checkout identity is a column in that file.
func StorePath(repoRoot string) string {
	return HomeStorePath()
}

// LegacyIndexPath is a checkout's old database, merged by store.MergeCheckout.
func LegacyIndexPath(repoRoot string) string {
	return filepath.Join(repoRoot, DefaultDataDir, DefaultIndexDB)
}

// ResolveCheckout cleans a checkout path: trim, expand ~/, and resolve a
// relative path against repoRoot.
func ResolveCheckout(repoRoot, raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("archives entry is empty")
	}
	p := raw
	if expanded, ok := expandHomePath(p); ok {
		p = expanded
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(repoRoot, p)
	}
	root, err := Canonical(p)
	if err != nil {
		return "", fmt.Errorf("archives path %s: %w", raw, err)
	}
	return root, nil
}

func CommandsLogPath(repoRoot string) string {
	canon, err := Canonical(repoRoot)
	if err != nil {
		canon = repoRoot
	}
	if dir := CheckoutDir(canon); dir != "" {
		return filepath.Join(dir, DefaultCommandsLog)
	}
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
	return PathUnder(path, UserGlobalPrefix)
}

func IsHomeGlobalPath(path string) bool {
	return PathUnder(path, HomeGlobalPrefix)
}
