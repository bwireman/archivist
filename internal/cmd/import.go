package cmd

import (
	"fmt"

	"github.com/bwireman/archivist/internal/archive"
	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import",
		Short: "Upsert typed markdown into SQLite (no prune)",
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
			svc := archive.New(root, cfg, repo, home)
			res, err := svc.Import()
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Imported %d, updated %d, skipped %d\n", res.Imported, res.Updated, res.Skipped)
			return nil
		},
	}
}
