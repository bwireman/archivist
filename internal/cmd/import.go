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
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			db, err := openExistingStore(root)
			if err != nil {
				return err
			}
			defer closeStore(db)
			svc := archive.New(root, cfg, db)
			res, err := svc.Import()
			if err != nil {
				return err
			}
			noteResult(cmd, res)
			fmt.Fprintf(cmd.OutOrStdout(), "Imported %d, updated %d, skipped %d\n", res.Imported, res.Updated, res.Skipped)
			return nil
		},
	}
}
