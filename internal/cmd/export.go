package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/export"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	var bundle string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Generate the markdown archive (off unless records.write_docs)",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			outDir, skip := docsExportTarget(root, cfg, bundle)
			if skip != "" {
				fmt.Fprintln(cmd.OutOrStdout(), skip)
				return nil
			}
			repo, home, err := openStores(root)
			if err != nil {
				return err
			}
			defer repo.Close()
			defer home.Close()
			if err := export.Run(repo, home, export.Options{RepoRoot: root, OutDir: outDir}); err != nil {
				return err
			}
			if bundle != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Wrote bundle to %s\n", bundle)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s/\n", cfg.Records.Export)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&bundle, "bundle", "", "write portable bundle to directory")
	return cmd
}

// docsExportTarget is the directory export should write. An empty skip means write;
// otherwise export is a no-op (records.write_docs is false and --bundle was not set).
func docsExportTarget(root string, cfg *config.Config, bundle string) (outDir string, skip string) {
	if bundle != "" {
		return bundle, ""
	}
	if cfg == nil || !cfg.Records.WriteDocs {
		dir := config.DefaultArchiveDir
		if cfg != nil && strings.TrimSpace(cfg.Records.Export) != "" {
			dir = cfg.Records.Export
		}
		return "", fmt.Sprintf("records.write_docs is false; skipped writing %s/", dir)
	}
	return filepath.Join(root, cfg.Records.Export), ""
}
