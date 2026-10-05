// Package cmd implements the archivist CLI.
package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
				fmt.Fprintf(cmd.OutOrStdout(), "archivist version %s\n", version.String())
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
		Short: "Print the version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "archivist version %s\n", version.String())
		},
	}
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
	abs, err := filepath.Abs(repoPath)
	if err != nil {
		return "", err
	}
	return abs, nil
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
	var scope string
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
			repo, home, err := openStores(root)
			if err != nil {
				return err
			}
			defer repo.Close()
			defer home.Close()

			embedder := embed.OptionalFromConfig(cmd.Context(), cfg.Ollama)
			engine := &retrieve.Engine{Repo: repo, Home: home}
			opts := retrieve.Options{TopK: topK, Query: strings.Join(args, " "), Status: record.Status(status)}
			if recType != "" {
				opts.Type = record.Type(recType)
			}
			if scope != "" {
				opts.Scope = record.Scope(scope)
			}
			results, err := engine.Search(cmd.Context(), embedder, opts)
			if err != nil {
				return err
			}
			logged := any(results)
			if !full {
				logged = retrieve.ProjectSearch(results)
			}
			noteResult(cmd, logged)
			return writeSearchOutput(cmd.OutOrStdout(), results, asJSON, full)
		},
	}
	cmd.Flags().IntVar(&topK, "top", retrieve.DefaultTopK, "number of results")
	cmd.Flags().StringVar(&recType, "type", "", "filter by record type: decision, rule, feature, guide, map, pitfall")
	cmd.Flags().StringVar(&scope, "scope", "", "filter by scope: dev|repo|global")
	cmd.Flags().StringVar(&status, "status", "", "record status: proposed, accepted, deprecated, or superseded; default omits deprecated and superseded")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	cmd.Flags().BoolVar(&full, "full", false, "with --json, print every stored field instead of the search card")
	return cmd
}

func writeSearchOutput(w io.Writer, results []retrieve.Result, asJSON, full bool) error {
	if !asJSON {
		_, err := io.WriteString(w, retrieve.FormatResults(results))
		return err
	}
	if full {
		return writeIndentedJSON(w, results)
	}
	return writeIndentedJSON(w, retrieve.ProjectSearch(results))
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
			repo, home, err := openStores(root)
			if err != nil {
				return err
			}
			defer repo.Close()
			defer home.Close()

			health := embed.CheckHealth(cmd.Context(), cfg)
			recCount, _ := repo.RecordCount()
			homeCount, _ := home.RecordCount()
			fileCount, _ := repo.FileCount()
			queue, _ := repo.QueueDepth()
			homeQueue, _ := home.QueueDepth()
			lastIdx, hasIdx, _ := repo.LastIndexedAt()

			type status struct {
				Version       string `json:"version"`
				EmbedderOK    bool   `json:"embedder_ok"`
				EmbedderError string `json:"embedder_error,omitempty"`
				RecordCount   int    `json:"record_count"`
				FileCount     int    `json:"file_count"`
				QueueDepth    int    `json:"queue_depth"`
				HomeQueue     int    `json:"home_queue_depth"`
				LastIndexedAt string `json:"last_indexed_at,omitempty"`
			}
			s := status{
				Version:       version.Version,
				EmbedderOK:    health.EmbedderOK,
				EmbedderError: health.EmbedderError,
				RecordCount:   recCount + homeCount,
				FileCount:     fileCount,
				QueueDepth:    queue + homeQueue,
				HomeQueue:     homeQueue,
			}
			if hasIdx {
				s.LastIndexedAt = lastIdx.Format("2006-01-02 15:04:05 UTC")
			}
			noteResult(cmd, s)
			if asJSON {
				return writeIndentedJSON(os.Stdout, s)
			}
			fmt.Printf("Version: %s\n", version.String())
			fmt.Printf("Embeddings (Ollama): ")
			if s.EmbedderOK {
				fmt.Println("ok")
			} else {
				fmt.Println("unavailable -", s.EmbedderError)
			}
			fmt.Printf("Records: %d, code files: %d\n", s.RecordCount, s.FileCount)
			fmt.Printf("Embed queue: %d\n", s.QueueDepth)
			if hasIdx {
				fmt.Printf("Last indexed: %s\n", s.LastIndexedAt)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	return cmd
}

func writeIndentedJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
