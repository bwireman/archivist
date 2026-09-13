package cmd

import (
	"github.com/bwireman/archivist/internal/embed"
	mcpsrv "github.com/bwireman/archivist/internal/mcp"
	"github.com/spf13/cobra"
)

func newMCPCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Start the MCP server over stdio",
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
			embedder := embed.OptionalFromConfig(cmd.Context(), cfg.Ollama)
			srv := mcpsrv.New(root, cfg, repo, home, embedder)
			return mcpsrv.ServeStdio(srv)
		},
	}
}
