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
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			repo, err := openExistingStore(root)
			if err != nil {
				return err
			}
			defer repo.Close()
			client := embed.NewOllamaClientFromConfig(cfg.Ollama)
			if err := client.Healthy(cmd.Context()); err != nil {
				return fmt.Errorf("%w (run: ollama serve)", err)
			}
			stores := []*store.Store{repo}
			home, err := openHomeIfPresent()
			if err != nil {
				return err
			}
			if home != nil {
				defer home.Close()
				stores = append(stores, home)
			}
			w := &embed.Worker{
				Stores:   stores,
				Embedder: client,
				Model:    cfg.Ollama.EmbedModel,
			}
			n, err := w.Run(cmd.Context(), embed.WorkerOptions{Once: once, Concurrency: concurrency})
			noteResult(cmd, map[string]int{"embedded": n})
			fmt.Fprintf(cmd.OutOrStdout(), "Embedded %d records\n", n)
			return err
		},
	}
	cmd.Flags().BoolVar(&once, "once", false, "process every queued item once, then exit")
	cmd.Flags().IntVar(&concurrency, "concurrency", 2, "parallel embed workers")
	return cmd
}
