package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/bwireman/archivist/internal/archive"
	"github.com/bwireman/archivist/internal/trace"
	"github.com/spf13/cobra"
)

func newCiteCmd() *cobra.Command {
	var effect string
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "cite <id>",
		Short: "Record that a retrieved record changed the work",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			effect, err = trace.PrepareCite(cfg.LogCommands, effect)
			if err != nil {
				return err
			}
			repo, home, err := openStores(root)
			if err != nil {
				return err
			}
			defer repo.Close()
			defer home.Close()
			rec, err := archive.New(root, cfg, repo, home).Get(args[0])
			if err != nil {
				return err
			}
			noted := map[string]string{"id": rec.ID, "effect": effect}
			noteResult(cmd, noted)
			if asJSON {
				return json.NewEncoder(os.Stdout).Encode(noted)
			}
			fmt.Printf("Cited %s\n", rec.ID)
			return nil
		},
	}
	cmd.Flags().StringVar(&effect, "effect", "", "one line: what the record changed")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	_ = cmd.MarkFlagRequired("effect")
	return cmd
}
