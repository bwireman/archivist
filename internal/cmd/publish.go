package cmd

import (
	"github.com/bwireman/archivist/internal/publish"
	"github.com/spf13/cobra"
)

func newPublishCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "publish <destination>",
		Short: "Export bundle and run configured publish command",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			repo, home, err := openStores(root, cfg)
			if err != nil {
				return err
			}
			defer repo.Close()
			defer home.Close()
			return publish.Publish(root, cfg, repo, home, args[0])
		},
	}
}
