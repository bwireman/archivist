package cmd

import (
	"fmt"
	"time"

	"github.com/bwireman/archivist/internal/review"
	"github.com/spf13/cobra"
)

func newReviewCmd() *cobra.Command {
	var due bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "review",
		Short: "List disagreements to read, stale paths, and missed rules",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			now := time.Now()
			if due {
				rep, err := review.Clock(root, cfg, now)
				if err != nil {
					return err
				}
				if asJSON {
					return writeIndentedJSON(cmd.OutOrStdout(), rep)
				}
				fmt.Fprint(cmd.OutOrStdout(), review.FormatDue(rep))
				return nil
			}
			repo, home, err := openQueryStores(root)
			if err != nil {
				return err
			}
			defer repo.Close()
			defer home.Close()
			rep, err := review.Collect(root, cfg, repo, home, now)
			if err != nil {
				return err
			}
			if asJSON {
				return writeIndentedJSON(cmd.OutOrStdout(), rep)
			}
			fmt.Fprint(cmd.OutOrStdout(), review.Format(rep))
			return nil
		},
	}
	cmd.Flags().BoolVar(&due, "due", false, "report whether a full review is due; this check does not count as a review")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	return cmd
}
