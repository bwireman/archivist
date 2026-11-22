package cmd

import (
	"encoding/json"

	"github.com/bwireman/archivist/internal/critic"
	"github.com/spf13/cobra"
)

func newDecideCmd() *cobra.Command {
	var title, body string
	cmd := &cobra.Command{
		Use:   "decide",
		Short: "Score a candidate record without writing",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			adv := critic.RememberAdvisory(cmd.Context(), cfg.Ollama.ResolvedBaseURL(), cfg.Ollama.CriticModel, cfg.Ollama.EmbedModel, nil, title, body)
			enc := json.NewEncoder(cmd.OutOrStdout())
			if adv == nil {
				return enc.Encode(map[string]any{})
			}
			return enc.Encode(adv)
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "candidate title")
	cmd.Flags().StringVar(&body, "body", "", "candidate body")
	_ = cmd.MarkFlagRequired("title")
	_ = cmd.MarkFlagRequired("body")
	return cmd
}
