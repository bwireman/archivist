package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/docs"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/index"
	"github.com/bwireman/archivist/internal/search"
	"github.com/bwireman/archivist/internal/store"
	"github.com/spf13/cobra"
)

var (
	repoPath string
	jsonOut  bool
)

func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:   "archivist",
		Short: "Local code and documentation indexer",
	}
	root.PersistentFlags().StringVar(&repoPath, "path", ".", "repository root path")
	root.AddCommand(newInitCmd())
	root.AddCommand(newIndexCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newDocsCmd())
	root.AddCommand(newDecisionsCmd())
	return root
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

func openStore(root string, cfg *config.Config) (*store.Store, error) {
	return store.Open(config.StorePath(root, cfg))
}

func loadProviders(cfg *config.Config) (embed.Providers, error) {
	return embed.NewProviders(cfg)
}

func requireEmbedder(ctx context.Context, providers embed.Providers) error {
	if h, ok := providers.Embedder.(interface{ Healthy(context.Context) error }); ok {
		if err := h.Healthy(ctx); err != nil {
			return fmt.Errorf("%w (run: ollama serve)", err)
		}
	}
	return nil
}

func requireGeneration(ctx context.Context, providers embed.Providers) error {
	if err := requireEmbedder(ctx, providers); err != nil {
		return err
	}
	if h, ok := providers.Generator.(interface{ Healthy(context.Context) error }); ok {
		if err := h.Healthy(ctx); err != nil {
			return fmt.Errorf("generation backend unavailable: %w", err)
		}
	}
	return nil
}

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize archivist config and data directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := repoRoot()
			if err != nil {
				return err
			}
			if err := os.MkdirAll(root, 0o755); err != nil {
				return err
			}
			cfg := config.Default()
			if err := config.Save(root, cfg); err != nil {
				return err
			}
			if err := os.MkdirAll(config.DataDir(root), 0o755); err != nil {
				return err
			}
			gitignore := filepath.Join(root, ".gitignore")
			appendGitignore(gitignore, ".archivist/\n")
			fmt.Println("Created .archivist.json and .archivist/")
			fmt.Println("Embeddings use Ollama:")
			fmt.Printf("  ollama pull %s\n", cfg.Ollama.EmbedModel)
			fmt.Println("Generation provider:", cfg.Provider)
			if cfg.UsesCursorGeneration() {
				fmt.Printf("  export %s=your_cursor_api_key\n", config.CursorAPIKeyEnv)
				fmt.Printf("  cursor model: %s\n", cfg.Cursor.Model)
			} else {
				fmt.Printf("  ollama pull %s\n", cfg.Ollama.GenerateModel)
			}
			return nil
		},
	}
}

func appendGitignore(path, line string) {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	trimmed := strings.TrimSpace(line)
	if err == nil {
		for _, l := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(l) == trimmed {
				return
			}
		}
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.WriteString(line)
}

func newIndexCmd() *cobra.Command {
	var scope string
	cmd := &cobra.Command{
		Use:   "index",
		Short: "Incrementally index code, docs, git history, and ADRs",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			st, err := openStore(root, cfg)
			if err != nil {
				return err
			}
			defer st.Close()

			client := embed.NewOllamaClientFromConfig(cfg.Ollama)
			if err := client.Healthy(cmd.Context()); err != nil {
				return fmt.Errorf("%w (run: ollama serve)", err)
			}

			idx := &index.Indexer{
				RepoRoot: root,
				Cfg:      cfg,
				Store:    st,
				Embedder: client,
			}
			if err := idx.Index(cmd.Context(), scope); err != nil {
				return err
			}
			count, _ := st.ChunkCount()
			fmt.Printf("Indexed successfully (%d chunks)\n", count)
			return nil
		},
	}
	cmd.Flags().StringVar(&scope, "scope", "", "limit indexing to a subdirectory")
	return cmd
}

func newSearchCmd() *cobra.Command {
	var topK int
	var chunkType string
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Semantic search over the index",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			st, err := openStore(root, cfg)
			if err != nil {
				return err
			}
			defer st.Close()

			client := embed.NewOllamaClientFromConfig(cfg.Ollama)
			query := args[0]
			opts := search.Options{TopK: topK, AsJSON: jsonOut}
			if chunkType != "" {
				opts.Type = store.ChunkType(chunkType)
			}
			results, err := search.Search(cmd.Context(), st, client, query, opts)
			if err != nil {
				return err
			}
			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(results)
			}
			fmt.Print(search.FormatResults(results))
			return nil
		},
	}
	cmd.Flags().IntVar(&topK, "top", 10, "number of results")
	cmd.Flags().StringVar(&chunkType, "type", "", "filter by chunk type: code|doc|commit|adr|comment")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output JSON")
	return cmd
}

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show index and provider status",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			st, err := openStore(root, cfg)
			if err != nil {
				return err
			}
			defer st.Close()

			health := embed.CheckHealth(context.Background(), cfg)

			chunks, _ := st.ChunkCount()
			files, _ := st.FileCount()
			lastIdx, hasIdx, _ := st.LastIndexedAt()
			docsSync, hasDocs, _ := st.DocsSyncAt()

			type status struct {
				Provider       string `json:"provider"`
				EmbedderOK     bool   `json:"embedder_ok"`
				EmbedderError  string `json:"embedder_error,omitempty"`
				GeneratorOK    bool   `json:"generator_ok"`
				GeneratorError string `json:"generator_error,omitempty"`
				ChunkCount     int    `json:"chunk_count"`
				FileCount      int    `json:"file_count"`
				LastIndexedAt  string `json:"last_indexed_at,omitempty"`
				DocsSyncAt     string `json:"docs_sync_at,omitempty"`
				Stale          bool   `json:"stale"`
			}
			s := status{
				Provider:      health.Provider,
				EmbedderOK:    health.EmbedderOK,
				EmbedderError: health.EmbedderError,
				GeneratorOK:   health.GeneratorOK,
				GeneratorError: health.GeneratorError,
				ChunkCount:    chunks,
				FileCount:     files,
				Stale:         hasIdx && hasDocs && lastIdx.After(docsSync),
			}
			if hasIdx {
				s.LastIndexedAt = lastIdx.Format("2006-01-02 15:04:05 UTC")
			}
			if hasDocs {
				s.DocsSyncAt = docsSync.Format("2006-01-02 15:04:05 UTC")
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(s)
			}

			fmt.Printf("Provider: %s\n", s.Provider)
			fmt.Printf("Embeddings (Ollama): ")
			if s.EmbedderOK {
				fmt.Println("ok")
			} else {
				fmt.Println("unavailable -", s.EmbedderError)
			}
			genLabel := "Generation"
			if cfg.UsesCursorGeneration() {
				genLabel = "Generation (Cursor)"
			} else {
				genLabel = "Generation (Ollama)"
			}
			fmt.Printf("%s: ", genLabel)
			if s.GeneratorOK {
				fmt.Println("ok")
			} else {
				fmt.Println("unavailable -", s.GeneratorError)
			}
			fmt.Printf("Chunks: %d (%d files)\n", s.ChunkCount, s.FileCount)
			if hasIdx {
				fmt.Printf("Last indexed: %s\n", s.LastIndexedAt)
			}
			if hasDocs {
				fmt.Printf("Docs synced: %s\n", s.DocsSyncAt)
			}
			if s.Stale {
				fmt.Println("Docs may be stale (index newer than last docs update)")
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output JSON")
	return cmd
}

func newDocsCmd() *cobra.Command {
	docsCmd := &cobra.Command{
		Use:   "docs",
		Short: "Documentation maintenance",
	}
	docsCmd.AddCommand(newDocsUpdateCmd())
	return docsCmd
}

func newDocsUpdateCmd() *cobra.Command {
	var dryRun, all bool
	var scope string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Refresh or create documentation pages",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			st, err := openStore(root, cfg)
			if err != nil {
				return err
			}
			defer st.Close()

			providers, err := loadProviders(cfg)
			if err != nil {
				return err
			}
			if err := requireGeneration(cmd.Context(), providers); err != nil {
				return err
			}

			return docs.Update(cmd.Context(), docs.UpdateOptions{
				RepoRoot: root,
				Cfg:      cfg,
				Store:    st,
				Embedder: providers.Embedder,
				Gen:      providers.Generator,
				DryRun:   dryRun,
				All:      all,
				Scope:    scope,
			})
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print diffs without writing files")
	cmd.Flags().BoolVar(&all, "all", false, "update all pages regardless of staleness")
	cmd.Flags().StringVar(&scope, "scope", "", "limit to pages matching scope")
	return cmd
}

func newDecisionsCmd() *cobra.Command {
	decisionsCmd := &cobra.Command{
		Use:   "decisions",
		Short: "Record architecture decisions",
	}
	decisionsCmd.AddCommand(newDecisionsRecordCmd())
	return decisionsCmd
}

func newDecisionsRecordCmd() *cobra.Command {
	var (
		title   string
		summary string
		ctxText string
		status  string
		files   string
		output  string
		stdin   bool
		dryRun  bool
		topK    int
	)

	cmd := &cobra.Command{
		Use:   "record",
		Short: "Record a decision as an ADR with relevant code context",
		Long: `Record an architecture decision for bot or automation use.

Provide flags directly, or pass a JSON object via --stdin:

  {
    "title": "Use SQLite for local index",
    "summary": "Embedded SQLite keeps the index portable and inspectable.",
    "context": "Bot noticed repeated storage tradeoff discussion.",
    "status": "accepted",
    "files": ["internal/store/store.go"]
  }
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			st, err := openStore(root, cfg)
			if err != nil {
				return err
			}
			defer st.Close()

			input, err := decisionInputFromFlags(stdin, title, summary, ctxText, status, files)
			if err != nil {
				return err
			}

			providers, err := loadProviders(cfg)
			if err != nil {
				return err
			}
			if err := requireGeneration(cmd.Context(), providers); err != nil {
				return err
			}

			result, err := docs.RecordDecision(cmd.Context(), docs.RecordDecisionOptions{
				RepoRoot: root,
				Cfg:      cfg,
				Store:    st,
				Embedder: providers.Embedder,
				Gen:      providers.Generator,
				Input:    input,
				DryRun:   dryRun,
				Output:   output,
				TopK:     topK,
			})
			if err != nil {
				return err
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(result)
			}

			if dryRun {
				fmt.Printf("Would write: %s\n\n", result.Path)
				fmt.Println(result.Content)
				return nil
			}

			fmt.Printf("Recorded decision: %s\n", result.Path)
			if len(result.RelatedFiles) > 0 {
				fmt.Printf("Related code: %s\n", strings.Join(result.RelatedFiles, ", "))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&title, "title", "", "decision title")
	cmd.Flags().StringVar(&summary, "summary", "", "decision summary")
	cmd.Flags().StringVar(&ctxText, "context", "", "extra context from the bot about what was observed")
	cmd.Flags().StringVar(&status, "status", "accepted", "decision status: proposed|accepted|deprecated|superseded")
	cmd.Flags().StringVar(&files, "files", "", "comma-separated file paths to include as code context")
	cmd.Flags().StringVar(&output, "output", "", "output ADR path (default: auto-numbered under docs/decisions/)")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "read decision input as JSON from stdin")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview ADR without writing")
	cmd.Flags().IntVar(&topK, "top", 8, "number of related code chunks to retrieve")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "output result as JSON")
	return cmd
}

func decisionInputFromFlags(useStdin bool, title, summary, ctxText, status, files string) (docs.DecisionInput, error) {
	if useStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return docs.DecisionInput{}, err
		}
		input, err := docs.ParseDecisionInputJSON(data)
		if err != nil {
			return docs.DecisionInput{}, err
		}
		if input.Status == "" && status != "" {
			input.Status = status
		}
		return input, nil
	}

	input := docs.DecisionInput{
		Title:   title,
		Summary: summary,
		Context: ctxText,
		Status:  status,
	}
	if files != "" {
		for _, f := range strings.Split(files, ",") {
			f = strings.TrimSpace(f)
			if f != "" {
				input.Files = append(input.Files, f)
			}
		}
	}
	return input, nil
}
