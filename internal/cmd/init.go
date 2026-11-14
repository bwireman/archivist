package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bwireman/archivist/internal/config"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize archivist config and data directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := repoRoot()
			if err != nil {
				return err
			}
			existed := config.HasConfig(root)
			cfg := config.Default()
			if existed {
				loaded, err := config.Load(root)
				if err != nil {
					return err
				}
				cfg = loaded
			}
			if err := applyInit(root, cfg, !existed); err != nil {
				return err
			}
			verb := "Created"
			if existed {
				verb = "Updated"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s checkout config in %s\nEmbeddings use Ollama:\n  ollama pull %s\nSeed the archive from this repo with the init-archive skill (scan decisions, rules, and features, then embed).\n", verb, config.CheckoutDir(root), cfg.Ollama.EmbedModel)
			return nil
		},
	}
}

func applyInit(root string, cfg *config.Config, writeConfig bool) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	if writeConfig {
		if err := config.Save(root, cfg); err != nil {
			return err
		}
		if err := config.RelocateCheckoutFiles(root); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(filepath.Join(root, cfg.Records.Repo), 0o755); err != nil {
		return err
	}
	if cfg.Records.GlobalInRepo() {
		if err := os.MkdirAll(filepath.Join(root, cfg.Records.Global), 0o755); err != nil {
			return err
		}
	} else if dir := cfg.Records.GlobalDir(root); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	if cfg.Records.WriteDocs {
		if err := os.MkdirAll(filepath.Join(root, cfg.Records.Export), 0o755); err != nil {
			return err
		}
	}
	if dir := cfg.DevRecordsDir(); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	repo, home, err := openStores(root)
	if err != nil {
		return err
	}
	canon, err := config.Canonical(root)
	if err != nil {
		_ = repo.Close()
		_ = home.Close()
		return err
	}
	if err := repo.RegisterCheckout(canon, config.CheckoutHex(canon)); err != nil {
		_ = repo.Close()
		_ = home.Close()
		return err
	}
	err = repo.Close()
	if cerr := home.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return err
	}
	return appendGitignore(filepath.Join(root, ".gitignore"), ".archivist/\n")
}
