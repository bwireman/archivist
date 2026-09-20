package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/skills"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize archivist config and data directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := repoRoot()
			if err != nil {
				return err
			}
			existed := configExists(root)
			cfg := config.Default()
			if err := applyInit(root, cfg); err != nil {
				return err
			}
			verb := "Created"
			if existed {
				verb = "Updated"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s .archivist.json and .archivist/\nEmbeddings use Ollama:\n  ollama pull %s\nGit hook (optional): .githooks/post-commit runs archivist index after each commit when enabled:\n  git config core.hooksPath .githooks\n", verb, cfg.Ollama.EmbedModel)
			return nil
		},
	}
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
	if err := os.MkdirAll(filepath.Dir(config.StorePath(root)), 0o755); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(root, cfg.Records.Repo), 0o755); err != nil {
		return err
	}
	if cfg.Records.GlobalInRepo() {
		if err := os.MkdirAll(filepath.Join(root, cfg.Records.Global), 0o755); err != nil {
			return err
		}
	} else if dir := cfg.Records.GlobalDir(root); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	if cfg.Records.WriteDocs {
		if err := os.MkdirAll(filepath.Join(root, cfg.Records.Export), 0o755); err != nil {
			return err
		}
	}
	if dir := cfg.DevRecordsDir(); dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	if err := appendGitignore(filepath.Join(root, ".gitignore"), ".archivist/\n"); err != nil {
		return err
	}
	return skills.InstallGithooks(root)
}
