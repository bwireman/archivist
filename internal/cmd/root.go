package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/retrieve"
	"github.com/bwireman/archivist/internal/store"
	"github.com/bwireman/archivist/internal/version"
	"github.com/spf13/cobra"
)

var repoPath string

func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "archivist",
		Short:         "Knowledge archive for design decisions and code structure",
		Version:       version.String(),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetVersionTemplate("{{.Name}} version {{.Version}}\n")
	root.PersistentFlags().StringVar(&repoPath, "path", ".", "repository root path")
	root.AddCommand(newInitCmd())
	root.AddCommand(newIndexCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newVersionCmd())
	root.AddCommand(newEmbedCmd())
	root.AddCommand(newExportCmd())
	root.AddCommand(newPublishCmd())
	root.AddCommand(newMCPCmd())
	root.AddCommand(newRememberCmd())
	root.AddCommand(newUpdateCmd())
	root.AddCommand(newRetireCmd())
	root.AddCommand(newCheckCmd())
	root.AddCommand(newMigrateCmd())
	root.AddCommand(newSkillsCmd())
	return root
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "archivist version %s\n", version.String())
		},
	}
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
	path, err := resolveHomeStorePath()
	if err != nil {
		return nil, err
	}
	return store.Open(path)
}

func resolveHomeStorePath() (string, error) {
	path := config.HomeStorePath()
	if path != "" {
		return path, nil
	}
	return "", fmt.Errorf("cannot resolve home archive path (set $HOME)")
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

func appendGitignore(path, line string) error {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	trimmed := strings.TrimSpace(line)
	if err == nil {
		for _, l := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(l) == trimmed {
				return nil
			}
		}
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if len(data) > 0 && data[len(data)-1] != '\n' {
		if _, err := f.WriteString("\n"); err != nil {
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
	var asJSON bool
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
			opts := retrieve.Options{TopK: topK, Query: strings.Join(args, " ")}
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
			if asJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(results)
			}
			fmt.Print(retrieve.FormatResults(results))
			return nil
		},
	}
	cmd.Flags().IntVar(&topK, "top", retrieve.DefaultTopK, "number of results")
	cmd.Flags().StringVar(&recType, "type", "", "filter by record type: decision, rule, feature, guide, map, pitfall")
	cmd.Flags().StringVar(&scope, "scope", "", "filter by scope: dev|repo|global")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	return cmd
}

func newStatusCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show archive and embedder status",
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

			health := embed.CheckHealth(cmd.Context(), cfg)
			recCount, _ := repo.RecordCount()
			homeCount, _ := home.RecordCount()
			fileCount, _ := repo.FileCount()
			queue, _ := repo.QueueDepth()
			homeQueue, _ := home.QueueDepth()
			lastIdx, hasIdx, _ := repo.LastIndexedAt()

			type status struct {
				Version       string `json:"version"`
				Schema        int    `json:"schema"`
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
				Schema:        version.Schema,
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
			if asJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(s)
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
