package cmd

import (
	"fmt"
	"strings"

	"github.com/bwireman/archivist/internal/store"
	"github.com/spf13/cobra"
)

// DefaultMapLimit is the number of rows per section printed by `archivist map`.
const DefaultMapLimit = store.DefaultExploreLimit

func newMapCmd() *cobra.Command {
	var limit int
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "map <query>",
		Short: "Explore the code map: symbols, imports, importers, and commits",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			query := strings.Join(args, " ")
			primary, err := repo.ExploreCode(query, limit)
			if err != nil {
				return err
			}
			res := store.MapResult{CodeSearch: primary}
			for _, ex := range extras {
				section, err := ex.DB.ExploreCode(query, limit)
				if err != nil {
					return err
				}
				res.Archives = append(res.Archives, store.RootedCodeSearch{Root: ex.Root, CodeSearch: section})
			}
			noteResult(cmd, res)
			if asJSON {
				return writeIndentedJSON(cmd.OutOrStdout(), res)
			}
			fmt.Fprint(cmd.OutOrStdout(), formatMap(res))
			return nil
		},
	}
	cmd.Flags().IntVar(&limit, "limit", DefaultMapLimit, "max rows per section")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	return cmd
}

func formatMap(res store.MapResult) string {
	var b strings.Builder
	b.WriteString(formatCodeSearch(res.CodeSearch))
	for _, extra := range res.Archives {
		fmt.Fprintf(&b, "\n# %s\n", extra.Root)
		b.WriteString(formatCodeSearch(extra.CodeSearch))
	}
	return b.String()
}

func formatCodeSearch(res store.CodeSearch) string {
	var b strings.Builder
	for _, sym := range res.Symbols {
		fmt.Fprintf(&b, "%s:%d %s (%s)", sym.FilePath, sym.Line, sym.Name, sym.Kind)
		if sym.DocLine != "" {
			fmt.Fprintf(&b, " — %s", sym.DocLine)
		}
		b.WriteByte('\n')
	}
	writeEdges(&b, "Imports", res.Imports)
	writeEdges(&b, "Imported by", res.Importers)
	if len(res.Commits) > 0 {
		b.WriteString("\nCommits\n")
		for _, c := range res.Commits {
			fmt.Fprintf(&b, "  %s %s %s\n", c.Hash[:min(len(c.Hash), 8)],
				c.AuthoredAt.Format("2006-01-02"), c.Subject)
		}
	}
	if b.Len() == 0 {
		return "No matches. Run `archivist index` if the code map is empty.\n"
	}
	return b.String()
}

func writeEdges(b *strings.Builder, heading string, edges []store.SymbolEdge) {
	if len(edges) == 0 {
		return
	}
	fmt.Fprintf(b, "\n%s\n", heading)
	for _, e := range edges {
		fmt.Fprintf(b, "  %s -> %s\n", e.FromFile, e.ToPath)
	}
}
