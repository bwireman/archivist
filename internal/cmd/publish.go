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
			repo, home, err := openExistingStores(root)
			if err != nil {
				return err
			}
			defer closeStore(repo)
			defer closeStore(home)
			noteResult(cmd, map[string]string{"destination": args[0]})
			return publish.Publish(root, cfg, repo, home, args[0])
		},
	}
}
