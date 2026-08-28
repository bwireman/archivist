package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/bwireman/archivist/internal/embed"
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
		Short: "Incrementally index code, docs, git history, and ADRs",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			st, err := openStore(root, cfg)
			if err != nil {
				return err
			}
			defer st.Close()

			client := embed.NewOllamaClientFromConfig(cfg.Ollama)
			if err := client.Healthy(cmd.Context()); err != nil {
				return fmt.Errorf("%w (run: ollama serve)", err)
			}

			idx := &index.Indexer{
				RepoRoot: root,
				Cfg:      cfg,
				Store:    st,
				Embedder: client,
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
			count, _ := st.ChunkCount()
			fmt.Println(tui.FormatSummary(progress, count))
			return nil
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "", "limit indexing to a subdirectory")
	cmd.Flags().BoolVar(&plain, "plain", false, "print a one-line summary instead of the TUI")
	return cmd
}
