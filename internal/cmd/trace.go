package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/bwireman/archivist/internal/cmdlog"
	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/store"
	"github.com/bwireman/archivist/internal/trace"
	"github.com/spf13/cobra"
)

const traceDefaultWindow = 7 * 24 * time.Hour

func newTraceCmd() *cobra.Command {
	var since string
	var all bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "trace",
		Short: "Report whether archive lookups showed up in later work",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			if all && since != "" {
				return fmt.Errorf("--all and --since cannot be combined")
			}
			var after time.Time
			if !all && since == "" {
				after = time.Now().Add(-traceDefaultWindow)
			}
			rep, err := buildTrace(root, cfg, since, after)
			if err != nil {
				return err
			}
			if asJSON {
				return writeIndentedJSON(cmd.OutOrStdout(), rep)
			}
			fmt.Fprint(cmd.OutOrStdout(), trace.Format(rep))
			return nil
		},
	}
	cmd.Flags().StringVar(&since, "since", "", "git rev or RFC3339 time; join retrieved records to paths changed since then")
	cmd.Flags().BoolVar(&all, "all", false, "read the whole log instead of the last 7 days")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	return cmd
}

func buildTrace(root string, cfg *config.Config, since string, after time.Time) (*trace.Report, error) {
	path := config.CommandsLogPath(root)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			if !cfg.LogCommands {
				return &trace.Report{LoggingOff: true}, nil
			}
			return &trace.Report{Empty: true}, nil
		}
		return nil, err
	}
	entries, err := cmdlog.Read(path)
	if err != nil {
		return nil, err
	}
	opts := trace.Options{Since: since, After: after}
	var cat trace.Catalog
	if since != "" {
		join, err := trace.GitChanged(root, since)
		if err != nil {
			return nil, err
		}
		opts.After = join.After
		opts.Paths = join.Paths
		opts.GitSkipped = join.Skipped
		if join.Skipped == "" {
			repo, home, err := openStores(root)
			if err != nil {
				return nil, err
			}
			defer repo.Close()
			defer home.Close()
			cat = storeCatalog{repo: repo, home: home}
		}
	}
	return trace.Build(entries, opts, cat)
}

type storeCatalog struct {
	repo *store.Store
	home *store.Store
}

func (c storeCatalog) Record(id string) (*record.Record, error) {
	for _, st := range []*store.Store{c.repo, c.home} {
		if st == nil {
			continue
		}
		rec, ok, err := st.GetRecordByID(id)
		if err != nil {
			return nil, err
		}
		if ok {
			return rec, nil
		}
		rec, ok, err = st.GetRecordBySlug(id)
		if err != nil {
			return nil, err
		}
		if ok {
			return rec, nil
		}
	}
	return nil, nil
}

func (c storeCatalog) Rules() ([]*record.Record, error) {
	var out []*record.Record
	for _, st := range []*store.Store{c.repo, c.home} {
		if st == nil {
			continue
		}
		rules, err := st.RecordsByType(record.TypeRule)
		if err != nil {
			return nil, err
		}
		out = append(out, rules...)
	}
	return out, nil
}
