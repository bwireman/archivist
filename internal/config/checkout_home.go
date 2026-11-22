package config

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
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

// GlobalConfigPath is ~/.archivist/config.json, the settings file for every checkout.
func GlobalConfigPath() string {
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

// RelocateCheckoutFiles folds a repo .archivist.json or checkout config.json
// into ~/.archivist/config.json and moves commands.log under the home
// checkout directory. The log moves only when the home log is not already there.
func RelocateCheckoutFiles(root string) error {
	if _, err := Load(root); err != nil {
		return err
	}
	canon, err := Canonical(root)
	if err != nil {
		return err
	}
	return moveCommandsLog(canon)
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
