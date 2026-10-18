// Package cmd implements the archivist CLI.
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/bwireman/archivist/internal/archive"
	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/retrieve"
	"github.com/bwireman/archivist/internal/skills"
	"github.com/bwireman/archivist/internal/store"
	"github.com/bwireman/archivist/internal/version"
	"github.com/spf13/cobra"
)

var repoPath string

func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "archivist",
		Short:         "Knowledge archive for design decisions and code structure",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ver, _ := cmd.Flags().GetBool("version")
			if ver {
				printVersion(cmd.OutOrStdout())
				return nil
			}
			return cmd.Help()
		},
	}
	root.Flags().BoolP("version", "v", false, "version for archivist")
	root.PersistentFlags().StringVar(&repoPath, "path", ".", "repository root path")
	root.PersistentPreRunE = warnStaleRules
	defaultHelp := root.HelpFunc()
	root.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		_ = warnStaleRules(cmd, args)
		defaultHelp(cmd, args)
	})
	root.AddCommand(newInitCmd())
	root.AddCommand(newIndexCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newMapCmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newVersionCmd())
	root.AddCommand(newEmbedCmd())
	root.AddCommand(newExportCmd())
	root.AddCommand(newImportCmd())
	root.AddCommand(newPublishCmd())
	root.AddCommand(newMCPCmd())
	root.AddCommand(newRememberCmd())
	root.AddCommand(newUpdateCmd())
	root.AddCommand(newRetireCmd())
	root.AddCommand(newCheckCmd())
	root.AddCommand(newCiteCmd())
	root.AddCommand(newTraceCmd())
	root.AddCommand(newSkillsCmd())
	wrapArchiveCommandLogs(root)
	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version, commit, and build time",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			printVersion(cmd.OutOrStdout())
		},
	}
}

func printVersion(w io.Writer) {
	fmt.Fprintf(w, "archivist version %s\n", version.Report())
}

func warnStaleRules(cmd *cobra.Command, _ []string) error {
	if cmd.Annotations["archivist-rules-checked"] == "1" {
		return nil
	}
	if cmd.Annotations == nil {
		cmd.Annotations = map[string]string{}
	}
	cmd.Annotations["archivist-rules-checked"] = "1"

	root, err := repoRoot()
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "archivist: could not check installed rules: %v\n", err)
		return nil
	}
	stale, target, err := skills.RulesStale(root)
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "archivist: could not check installed rules: %v\n", err)
		return nil
	}
	if stale {
		fmt.Fprint(cmd.ErrOrStderr(), skills.StaleRulesNotice(target))
	}
	return nil
}

func repoRoot() (string, error) {
	return config.Canonical(repoPath)
}

func loadEnv() (string, *config.Config, error) {
	root, err := repoRoot()
	if err != nil {
		return "", nil, err
	}
	cfg, err := config.Load(root)
	if err != nil {
		return "", nil, err
	}
	return root, cfg, nil
}

func openStore(root string) (*store.Store, error) {
	return store.Open(config.StorePath(root))
}

func openHomeStore() (*store.Store, error) {
	path := config.HomeStorePath()
	if path == "" {
		return nil, errors.New("cannot resolve home archive path (set $HOME)")
	}
	return store.Open(path)
}

func openStores(root string) (*store.Store, *store.Store, error) {
	repo, err := openStore(root)
	if err != nil {
		return nil, nil, err
	}
	home, err := openHomeStore()
	if err != nil {
		_ = repo.Close()
		return nil, nil, err
	}
	return repo, home, nil
}

func openQueryStore(path string) (*store.Store, error) {
	db, err := store.OpenReadOnly(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%s does not exist; run archivist init", path)
		}
		return nil, err
	}
	return db, nil
}

func openQueryStores(root string) (*store.Store, *store.Store, error) {
	repo, err := openQueryStore(config.StorePath(root))
	if err != nil {
		return nil, nil, err
	}
	homePath := config.HomeStorePath()
	if homePath == "" {
		_ = repo.Close()
		return nil, nil, errors.New("cannot resolve home archive path (set $HOME)")
	}
	home, err := openQueryStore(homePath)
	if err != nil {
		_ = repo.Close()
		return nil, nil, err
	}
	return repo, home, nil
}

// openReadArchives opens the primary store, the home store, and each configured
// extra with OpenReadOnly. search, map, status, cite, check, trace, and mcp use
// this. It does not create databases, schema, or orphan cleanup.
func openReadArchives(root string, cfg *config.Config) (*store.Store, *store.Store, []archive.Extra, error) {
	repo, home, err := openQueryStores(root)
	if err != nil {
		return nil, nil, nil, err
	}
	roots, err := cfg.ArchiveRoots(root)
	if err != nil {
		_ = repo.Close()
		_ = home.Close()
		return nil, nil, nil, err
	}
	var extras []archive.Extra
	for _, extraRoot := range roots {
		db, err := store.OpenReadOnly(config.StorePath(extraRoot))
		if err != nil {
			_ = repo.Close()
			_ = home.Close()
			closeExtras(extras)
			return nil, nil, nil, fmt.Errorf("archive %s: %w", extraRoot, err)
		}
		extras = append(extras, archive.Extra{Root: extraRoot, DB: db})
	}
	return repo, home, extras, nil
}

func closeExtras(extras []archive.Extra) {
	for _, ex := range extras {
		if ex.DB != nil {
			_ = ex.DB.Close()
		}
	}
}

func extraRoots(extras []archive.Extra) []string {
	roots := make([]string, 0, len(extras))
	for _, ex := range extras {
		roots = append(roots, ex.Root)
	}
	return roots
}

func retrieveExtras(extras []archive.Extra) []retrieve.Extra {
	out := make([]retrieve.Extra, 0, len(extras))
	for _, ex := range extras {
		out = append(out, retrieve.Extra{Root: ex.Root, DB: ex.DB})
	}
	return out
}

func appendGitignore(path, line string) (err error) {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	trimmed := strings.TrimSpace(line)
	if err == nil {
		for l := range strings.SplitSeq(string(data), "\n") {
			if strings.TrimSpace(l) == trimmed {
				return nil
			}
		}
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := f.Close(); err == nil {
			err = cerr
		}
	}()
	if len(data) > 0 && data[len(data)-1] != '\n' {
		if _, err = f.WriteString("\n"); err != nil {
			return err
		}
	}
	_, err = f.WriteString(line)
	return err
}

func newSearchCmd() *cobra.Command {
	var topK int
	var recType string
	var status string
	var asJSON bool
	var full bool
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search the knowledge archive",
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

			embedder := embed.OptionalFromConfig(cmd.Context(), cfg.Ollama)
			engine := &retrieve.Engine{Repo: repo, Home: home, Extras: retrieveExtras(extras), QueryModel: cfg.Ollama.EmbedModel}
			opts := retrieve.Options{TopK: topK, Query: strings.Join(args, " "), Status: record.Status(status)}
			if recType != "" {
				opts.Type = record.Type(recType)
			}
			results, err := engine.Search(cmd.Context(), embedder, opts)
			if err != nil {
				return err
			}
			payload := any(results)
			if !full {
				payload = retrieve.ProjectSearch(results)
			}
			noteResult(cmd, payload)
			if !asJSON {
				_, err := fmt.Fprint(cmd.OutOrStdout(), retrieve.FormatResults(results))
				return err
			}
			return writeIndentedJSON(cmd.OutOrStdout(), payload)
		},
	}
	cmd.Flags().IntVar(&topK, "top", retrieve.DefaultTopK, "number of results")
	cmd.Flags().StringVar(&recType, "type", "", "filter by record type: decision, rule, feature, guide, map, pitfall")
	cmd.Flags().StringVar(&status, "status", "", "record status: proposed, accepted, deprecated, or superseded; default omits deprecated and superseded")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	cmd.Flags().BoolVar(&full, "full", false, "with --json, print every stored field instead of the search card")
	return cmd
}

func newStatusCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show archive and embedder status",
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

			health := embed.CheckHealth(cmd.Context(), cfg)
			recCount, _ := repo.RecordCount()
			homeCount, _ := home.RecordCount()
			fileCount, _ := repo.FileCount()
			queue, _ := repo.QueueDepth()
			homeQueue, _ := home.QueueDepth()
			lastIdx, hasIdx, _ := repo.LastIndexedAt()

			type extraCount struct {
				Root        string `json:"root"`
				RecordCount int    `json:"record_count"`
			}
			type status struct {
				Version        string           `json:"version"`
				EmbedderOK     bool             `json:"embedder_ok"`
				EmbedderError  string           `json:"embedder_error,omitempty"`
				RecordCount    int              `json:"record_count"`
				FileCount      int              `json:"file_count"`
				QueueDepth     int              `json:"queue_depth"`
				HomeQueue      int              `json:"home_queue_depth"`
				LastIndexedAt  string           `json:"last_indexed_at,omitempty"`
				Extras         []extraCount     `json:"extras,omitempty"`
				MissingVectors []store.EmbedGap `json:"missing_vectors"`
				EmbedGiveups   []store.EmbedGap `json:"embed_giveups"`
			}
			s := status{
				Version:        version.Version,
				EmbedderOK:     health.EmbedderOK,
				EmbedderError:  health.EmbedderError,
				RecordCount:    recCount + homeCount,
				FileCount:      fileCount,
				QueueDepth:     queue + homeQueue,
				HomeQueue:      homeQueue,
				MissingVectors: []store.EmbedGap{},
				EmbedGiveups:   []store.EmbedGap{},
			}
			var gapErr error
			s.MissingVectors, s.EmbedGiveups, gapErr = collectEmbedGaps(repo, home, extras)
			if gapErr != nil {
				return gapErr
			}
			for _, ex := range extras {
				n, _ := ex.DB.RecordCount()
				s.Extras = append(s.Extras, extraCount{Root: ex.Root, RecordCount: n})
			}
			if hasIdx {
				s.LastIndexedAt = lastIdx.Format("2006-01-02 15:04:05 UTC")
			}
			noteResult(cmd, s)
			out := cmd.OutOrStdout()
			if asJSON {
				return writeIndentedJSON(out, s)
			}
			fmt.Fprintf(out, "Version: %s\n", version.String())
			fmt.Fprintf(out, "Embeddings (Ollama): ")
			if s.EmbedderOK {
				fmt.Fprintln(out, "ok")
			} else {
				fmt.Fprintln(out, "unavailable -", s.EmbedderError)
			}
			fmt.Fprintf(out, "Records: %d, code files: %d\n", s.RecordCount, s.FileCount)
			fmt.Fprintf(out, "Embed queue: %d\n", s.QueueDepth)
			for _, g := range s.MissingVectors {
				fmt.Fprintf(out, "No vector: %s %s\n", g.ID, g.Title)
				if g.Archive != "" {
					fmt.Fprintf(out, "  %s\n", g.Archive)
				}
			}
			for _, g := range s.EmbedGiveups {
				fmt.Fprintf(out, "Embed gave up: %s %s\n", g.ID, g.Title)
				if g.Archive != "" {
					fmt.Fprintf(out, "  %s\n", g.Archive)
				}
			}
			if hasIdx {
				fmt.Fprintf(out, "Last indexed: %s\n", s.LastIndexedAt)
			}
			for _, ex := range s.Extras {
				fmt.Fprintf(out, "Extra archive: %s (%d records)\n", ex.Root, ex.RecordCount)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	return cmd
}

func collectEmbedGaps(repo, home *store.Store, extras []archive.Extra) (missing, giveups []store.EmbedGap, err error) {
	missing = []store.EmbedGap{}
	giveups = []store.EmbedGap{}
	add := func(st *store.Store, root string) error {
		if st == nil {
			return nil
		}
		miss, err := st.MissingVectors()
		if err != nil {
			return err
		}
		gave, err := st.EmbedGiveups()
		if err != nil {
			return err
		}
		for _, g := range miss {
			g.Archive = root
			missing = append(missing, g)
		}
		for _, g := range gave {
			g.Archive = root
			giveups = append(giveups, g)
		}
		return nil
	}
	if err = add(repo, ""); err != nil {
		return nil, nil, err
	}
	if err = add(home, ""); err != nil {
		return nil, nil, err
	}
	for _, ex := range extras {
		if err = add(ex.DB, ex.Root); err != nil {
			return nil, nil, err
		}
	}
	return missing, giveups, nil
}

func writeIndentedJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
