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
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			repo, home, extras, err := openReadArchives(root, cfg)
			if err != nil {
				return err
			}
			defer repo.Close()
			defer home.Close()
			defer closeExtras(extras)
			embedder := embed.OptionalFromConfig(cmd.Context(), cfg.Ollama)
			srv := mcpsrv.New(root, cfg, repo, home, embedder)
			srv.Engine.Extras = retrieveExtras(extras)
			srv.Archive.Extras = extras
			srv.Archive.ExtraRoots = extraRoots(extras)
			return mcpsrv.ServeStdio(srv)
		},
	}
}
