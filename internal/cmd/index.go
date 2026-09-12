package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/bwireman/archivist/internal/index"
	"github.com/spf13/cobra"
)

func newIndexCmd() *cobra.Command {
	var scope string
	var plain bool
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Index records and code structure (no Ollama required)",
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

			idx := &index.Indexer{
				RepoRoot: root,
				Cfg:      cfg,
				Store:    repo,
				Home:     home,
			}

			var progress index.Progress
			idx.Reporter = func(p index.Progress) { progress = p }
			if err := idx.Index(cmd.Context(), scope); err != nil {
				if errors.Is(err, context.Canceled) {
					return fmt.Errorf("indexing cancelled")
				}
				return err
			}
			count, _ := repo.RecordCount()
			fmt.Fprintln(cmd.OutOrStdout(), index.FormatSummary(progress, count))
			return nil
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "", "limit indexing to a subdirectory")
	cmd.Flags().BoolVar(&plain, "plain", false, "ignored; index always prints a one-line summary")
	_ = cmd.Flags().MarkHidden("plain")
	return cmd
}
