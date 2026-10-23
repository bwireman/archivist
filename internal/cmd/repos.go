package cmd

import (
	"fmt"
	"strings"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/repolinks"
	"github.com/spf13/cobra"
)

func newReposCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repos",
		Short: "Record checkout links in the home directory",
	}
	cmd.AddCommand(newReposLinkCmd(), newReposUnlinkCmd(), newReposListCmd())
	return cmd
}

func newReposLinkCmd() *cobra.Command {
	var kind string
	var records []string
	cmd := &cobra.Command{
		Use:   "link <checkout>",
		Short: "Record a related or depends-on edge in ~/.archivist/repo-links.json",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, _, err := loadEnv()
			if err != nil {
				return err
			}
			path, err := repoLinksPath()
			if err != nil {
				return err
			}
			refs, err := parseRecordFlags(root, records)
			if err != nil {
				return err
			}
			edge, err := repolinks.Link(path, root, kind, args[0], refs)
			if err != nil {
				return err
			}
			noteResult(cmd, edge)
			fmt.Fprintln(cmd.OutOrStdout(), "Linked")
			return nil
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "", "related or depends-on")
	cmd.Flags().StringArrayVar(&records, "record", nil, "record to attach, as home#id or checkout#id")
	_ = cmd.MarkFlagRequired("kind")
	return cmd
}

func newReposUnlinkCmd() *cobra.Command {
	var kind string
	cmd := &cobra.Command{
		Use:   "unlink <checkout>",
		Short: "Remove a related or depends-on edge",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, _, err := loadEnv()
			if err != nil {
				return err
			}
			path, err := repoLinksPath()
			if err != nil {
				return err
			}
			if err := repolinks.Unlink(path, root, kind, args[0]); err != nil {
				return err
			}
			noteResult(cmd, map[string]string{"status": "ok"})
			fmt.Fprintln(cmd.OutOrStdout(), "Unlinked")
			return nil
		},
	}
	cmd.Flags().StringVar(&kind, "kind", "", "related or depends-on")
	_ = cmd.MarkFlagRequired("kind")
	return cmd
}

func newReposListCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List checkout links stored in the home directory",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := repoLinksPath()
			if err != nil {
				return err
			}
			edges, err := repolinks.List(path)
			if err != nil {
				return err
			}
			if edges == nil {
				edges = []repolinks.Edge{}
			}
			noteResult(cmd, edges)
			if asJSON {
				return writeIndentedJSON(cmd.OutOrStdout(), map[string]any{"links": edges})
			}
			for _, edge := range edges {
				fmt.Fprintln(cmd.OutOrStdout(), formatEdge(edge))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	return cmd
}

func repoLinksPath() (string, error) {
	path := config.RepoLinksPath()
	if path == "" {
		return "", fmt.Errorf("cannot resolve home archive path (set $HOME)")
	}
	return path, nil
}

func parseRecordFlags(primary string, raw []string) ([]repolinks.RecordRef, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	out := make([]repolinks.RecordRef, 0, len(raw))
	for _, item := range raw {
		ref, err := repolinks.ParseRecord(primary, item)
		if err != nil {
			return nil, err
		}
		out = append(out, ref)
	}
	return out, nil
}

func formatEdge(edge repolinks.Edge) string {
	var b strings.Builder
	switch edge.Kind {
	case repolinks.KindRelated:
		fmt.Fprintf(&b, "%s\t%s\t%s", edge.Kind, edge.A, edge.B)
	case repolinks.KindDependsOn:
		fmt.Fprintf(&b, "%s\t%s\t%s", edge.Kind, edge.From, edge.To)
	default:
		b.WriteString(edge.Kind)
	}
	for _, ref := range edge.Records {
		root := ref.Root
		if root == "" {
			root = "home"
		}
		fmt.Fprintf(&b, "\t%s#%s", root, ref.ID)
	}
	return b.String()
}
