package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/tui"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var plain bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize archivist config and data directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := repoRoot()
			if err != nil {
				return err
			}
			existed := configExists(root)
			cfg := config.Default()
			if existed {
				loaded, err := config.Load(root)
				if err != nil {
					return err
				}
				cfg = loaded
			}

			useTUI := !plain && isatty.IsTerminal(os.Stdout.Fd())
			if useTUI {
				cfg, err = tui.RunInit(cmd.Context(), cfg)
				if err != nil {
					if errors.Is(err, context.Canceled) {
						return fmt.Errorf("init cancelled")
					}
					return err
				}
			} else {
				cfg = config.Default()
			}

			if err := applyInit(root, cfg); err != nil {
				return err
			}
			fmt.Print(tui.FormatInitSummary(cfg, existed))
			return nil
		},
	}
	cmd.Flags().BoolVar(&plain, "plain", false, "write default config without the setup TUI")
	return cmd
}

func configExists(root string) bool {
	_, err := os.Stat(filepath.Join(root, config.DefaultConfigName))
	return err == nil
}

func applyInit(root string, cfg *config.Config) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	if err := config.Save(root, cfg); err != nil {
		return err
	}
	if err := os.MkdirAll(config.DataDir(root), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(config.StorePath(root, cfg)), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, config.DefaultDecisionsDir), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, config.DefaultGlobalDecisionsDir), 0o755); err != nil {
		return err
	}
	return appendGitignore(filepath.Join(root, ".gitignore"), ".archivist/\n")
}
