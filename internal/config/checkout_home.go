package config

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// CheckoutHex is the directory name for a checkout under ~/.archivist/checkouts.
func CheckoutHex(root string) string {
	sum := sha256.Sum256([]byte(root))
	return hex.EncodeToString(sum[:])
}

// CheckoutDir is ~/.archivist/checkouts/<hex> for a canonical checkout path.
func CheckoutDir(root string) string {
	home := ArchivistHome()
	if home == "" {
		return ""
	}
	return filepath.Join(home, "checkouts", CheckoutHex(root))
}

func homeOllamaPath() string {
	home := ArchivistHome()
	if home == "" {
		return ""
	}
	return filepath.Join(home, "config.json")
}

func checkoutConfigPath(root string) string {
	dir := CheckoutDir(root)
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "config.json")
}

// RelocateCheckoutFiles moves .archivist.json and commands.log under the home
// checkout directory. The old json is removed only after both writes succeed.
// The log moves only when the home log is not already there.
func RelocateCheckoutFiles(root string) error {
	canon, err := Canonical(root)
	if err != nil {
		return err
	}
	dir := CheckoutDir(canon)
	if dir == "" {
		return nil
	}
	legacyJSON := filepath.Join(canon, DefaultConfigName)
	data, err := os.ReadFile(legacyJSON)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		var cfg Config
		if err := json.Unmarshal(data, &cfg); err != nil {
			return err
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		if err := writeHomeOllama(cfg.Ollama); err != nil {
			return err
		}
		cfg.Ollama = OllamaConfig{EmbedNumCtx: cfg.Ollama.EmbedNumCtx}
		body, err := json.MarshalIndent(&cfg, "", "  ")
		if err != nil {
			return err
		}
		body = append(body, '\n')
		if err := os.WriteFile(checkoutConfigPath(canon), body, 0o644); err != nil {
			return err
		}
		if err := os.Remove(legacyJSON); err != nil {
			return err
		}
	}
	return moveCommandsLog(canon)
}

func writeHomeOllama(ollama OllamaConfig) error {
	path := homeOllamaPath()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		var existing OllamaConfig
		if err := json.Unmarshal(data, &existing); err != nil {
			return err
		}
		if strings.TrimSpace(existing.EmbedModel) != "" {
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	body, err := json.MarshalIndent(ollama, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	return os.WriteFile(path, body, 0o644)
}

func moveCommandsLog(root string) error {
	src := filepath.Join(DataDir(root), DefaultCommandsLog)
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	dst := CommandsLogPath(root)
	if _, err := os.Stat(dst); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	} else {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.Rename(src, dst)
}
