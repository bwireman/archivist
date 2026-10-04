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
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Index code structure and git history (no Ollama required)",
		RunE: func(cmd *cobra.Command, _ []string) error {
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

			idx := &index.Indexer{
				RepoRoot: root,
				Cfg:      cfg,
				Store:    repo,
				Home:     home,
			}

			progress, err := idx.Index(cmd.Context(), scope)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return fmt.Errorf("indexing cancelled: %w", err)
				}
				return err
			}
			count, _ := repo.RecordCount()
			if home != nil {
				n, _ := home.RecordCount()
				count += n
			}
			noteResult(cmd, map[string]any{
				"files_indexed": progress.FilesIndexed,
				"files_removed": progress.FilesRemoved,
				"commits_new":   progress.CommitsNew,
				"records":       count,
			})
			fmt.Fprintln(cmd.OutOrStdout(), index.FormatSummary(progress, count))
			return nil
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "", "limit indexing to a subdirectory")
	return cmd
}
