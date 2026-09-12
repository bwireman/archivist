package cmd

import (
	"github.com/bwireman/archivist/internal/embed"
	mcpsrv "github.com/bwireman/archivist/internal/mcp"
	"github.com/spf13/cobra"
)

func newMCPCmd() *cobra.Command {
	var httpAddr string
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start the MCP server (stdio by default)",
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
			client := embed.NewOllamaClientFromConfig(cfg.Ollama)
			var embedder embed.Embedder
			if err := client.Healthy(cmd.Context()); err == nil {
				embedder = client
			}
			srv := mcpsrv.New(root, cfg, repo, home, embedder)
			if httpAddr != "" {
				return mcpsrv.ServeHTTP(srv, httpAddr)
			}
			return mcpsrv.ServeStdio(srv)
		},
	}
	cmd.Flags().StringVar(&httpAddr, "http", "", "listen for streamable HTTP (not yet implemented)")
	return cmd
}
