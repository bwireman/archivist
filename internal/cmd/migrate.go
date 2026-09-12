package cmd

import (
	"fmt"

	"github.com/bwireman/archivist/internal/migrate"
	"github.com/spf13/cobra"
)

func newMigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migration utilities",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "records",
		Short: "Convert legacy ADRs to typed records with front matter",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, _, err := loadEnv()
			if err != nil {
				return err
			}
			if err := migrate.Records(root); err != nil {
				return err
			}
			fmt.Println("Migrated records")
			return nil
		},
	})
	return cmd
}
