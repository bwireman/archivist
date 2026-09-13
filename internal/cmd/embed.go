package cmd

import (
	"fmt"

	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/store"
	"github.com/spf13/cobra"
)

func newEmbedCmd() *cobra.Command {
	var worker bool
	var once bool
	var concurrency int
	cmd := &cobra.Command{
		Use:   "embed",
		Short: "Embed queued records via Ollama",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !worker {
				return fmt.Errorf("use --worker to drain the embed queue")
			}
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
			opts := embed.WorkerOptions{Once: once, Concurrency: concurrency}
			n, err := w.Run(cmd.Context(), opts)
			fmt.Printf("Embedded %d records\n", n)
			return err
		},
	}
	cmd.Flags().BoolVar(&worker, "worker", false, "drain the embed queue")
	cmd.Flags().BoolVar(&once, "once", false, "process every queued item once, then exit")
	cmd.Flags().IntVar(&concurrency, "concurrency", 2, "parallel embed workers")
	return cmd
}
