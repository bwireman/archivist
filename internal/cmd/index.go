package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/bwireman/archivist/internal/index"
	"github.com/bwireman/archivist/internal/tui"
	"github.com/mattn/go-isatty"
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

			useTUI := !plain && isatty.IsTerminal(os.Stdout.Fd())
			var (
				progress index.Progress
				runErr   error
			)
			if useTUI {
				progress, runErr = tui.RunIndex(cmd.Context(), func(ctx context.Context, r index.Reporter) error {
					idx.Reporter = r
					return idx.Index(ctx, scope)
				})
			} else {
				idx.Reporter = func(p index.Progress) { progress = p }
				runErr = idx.Index(cmd.Context(), scope)
			}
			if runErr != nil {
				if errors.Is(runErr, context.Canceled) {
					return fmt.Errorf("indexing cancelled")
				}
				return runErr
			}
			count, _ := repo.RecordCount()
			fmt.Println(tui.FormatSummary(progress, count))
			return nil
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "", "limit indexing to a subdirectory")
	cmd.Flags().BoolVar(&plain, "plain", false, "print a one-line summary instead of the TUI")
	return cmd
}
