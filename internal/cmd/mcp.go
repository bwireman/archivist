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
			db, err := openReadOnlyArchive(root)
			if err != nil {
				return err
			}
			defer db.Close()
			embedder := embed.OptionalFromConfig(cmd.Context(), cfg.Ollama)
			srv := mcpsrv.New(root, cfg, db, embedder)
			if err := srv.RegistrationError(); err != nil {
				return err
			}
			return mcpsrv.ServeStdio(srv)
		},
	}
}
