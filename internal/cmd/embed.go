package cmd

import (
	"fmt"

	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/store"
	"github.com/spf13/cobra"
)

func newEmbedCmd() *cobra.Command {
	var once bool
	var concurrency int
	cmd := &cobra.Command{
		Use:   "embed",
		Short: "Embed queued records via Ollama",
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
			client := embed.NewOllamaClientFromConfig(cfg.Ollama)
			if err := client.Healthy(cmd.Context()); err != nil {
				return fmt.Errorf("%w (run: ollama serve)", err)
			}
			w := &embed.Worker{
				Stores:   []*store.Store{repo, home},
				Embedder: client,
				Model:    cfg.Ollama.EmbedModel,
			}
			n, err := w.Run(cmd.Context(), embed.WorkerOptions{Once: once, Concurrency: concurrency})
			fmt.Fprintf(cmd.OutOrStdout(), "Embedded %d records\n", n)
			return err
		},
	}
	cmd.Flags().BoolVar(&once, "once", false, "process every queued item once, then exit")
	cmd.Flags().IntVar(&concurrency, "concurrency", 2, "parallel embed workers")
	return cmd
}
