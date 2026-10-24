package cmd

import (
	"fmt"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/store"
	"github.com/spf13/cobra"
)

func newLinkCmd() *cobra.Command {
	var kind, from, to string
	cmd := &cobra.Command{
		Use:   "link",
		Short: "Store a record, pin, or repo link in this checkout",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			edge, err := writeLink(kind, from, to, false)
			if err != nil {
				return err
			}
			noteResult(cmd, edge)
			fmt.Fprintln(cmd.OutOrStdout(), "Linked")
			return nil
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "", "record, pin, or repo")
	cmd.Flags().StringVar(&from, "from", "", "record id in this checkout")
	cmd.Flags().StringVar(&to, "to", "", "checkout for repo, or checkout#id for record and pin")
	_ = cmd.MarkFlagRequired("kind")
	_ = cmd.MarkFlagRequired("to")
	cmd.AddCommand(newLinkListCmd())
	return cmd
}

func newUnlinkCmd() *cobra.Command {
	var kind, from, to string
	cmd := &cobra.Command{
		Use:   "unlink",
		Short: "Remove a record, pin, or repo link from this checkout",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if _, err := writeLink(kind, from, to, true); err != nil {
				return err
			}
			noteResult(cmd, map[string]string{"status": "ok"})
			fmt.Fprintln(cmd.OutOrStdout(), "Unlinked")
			return nil
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "", "record, pin, or repo")
	cmd.Flags().StringVar(&from, "from", "", "record id in this checkout")
	cmd.Flags().StringVar(&to, "to", "", "checkout for repo, or checkout#id for record and pin")
	_ = cmd.MarkFlagRequired("kind")
	_ = cmd.MarkFlagRequired("to")
	return cmd
}

func newLinkListCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List record links stored in this checkout",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, err := repoRoot()
			if err != nil {
				return err
			}
			db, err := openQueryStore(config.StorePath(root))
			if err != nil {
				return err
			}
			defer db.Close()
			edges, err := db.ListLinks()
			if err != nil {
				return err
			}
			payload := map[string]any{"links": edges}
			noteResult(cmd, payload)
			if asJSON {
				return writeIndentedJSON(cmd.OutOrStdout(), payload)
			}
			for _, edge := range edges {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\n", edge.Kind, edge.FromID, edge.ToRoot, edge.ToID)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	return cmd
}

func writeLink(kind, from, to string, remove bool) (store.Link, error) {
	root, err := repoRoot()
	if err != nil {
		return store.Link{}, err
	}
	edge, err := store.NewLink(root, kind, from, to)
	if err != nil {
		return store.Link{}, err
	}
	db, err := openStore(root)
	if err != nil {
		return store.Link{}, err
	}
	defer db.Close()
	if remove {
		return edge, db.DeleteLink(root, edge)
	}
	if err := db.InsertLink(root, edge); err != nil {
		return store.Link{}, err
	}
	return edge, nil
}
