package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/bwireman/archivist/internal/export"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	var bundle string
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Generate docs/archive/ grouped by record type",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			repo, home, err := openStores(root)
			if err != nil {
				return err
			}
			defer repo.Close()
			defer home.Close()
			opts := export.Options{RepoRoot: root}
			if bundle != "" {
				opts.OutDir = bundle
			} else {
				opts.OutDir = filepath.Join(root, cfg.Records.Export)
			}
			if err := export.Run(repo, home, opts); err != nil {
				return err
			}
			if bundle != "" {
				fmt.Printf("Wrote bundle to %s\n", bundle)
			} else {
				fmt.Println("Wrote docs/archive/")
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&bundle, "bundle", "", "write portable bundle to directory")
	return cmd
}
