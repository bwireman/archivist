package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/dump"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/index"
	"github.com/bwireman/archivist/internal/search"
	"github.com/bwireman/archivist/internal/store"
	"github.com/spf13/cobra"
)

var (
	repoPath string
)

func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "archivist",
		Short:         "Local code indexer",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&repoPath, "path", ".", "repository root path")
	root.AddCommand(newInitCmd())
	root.AddCommand(newIndexCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newDumpCmd())
	root.AddCommand(newStatusCmd())
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
			if err := appendGitignore(gitignore, ".archivist/\n"); err != nil {
				return err
			}
			fmt.Println("Created .archivist.json and .archivist/")
			fmt.Println("Embeddings use Ollama:")
			fmt.Printf("  ollama pull %s\n", cfg.Ollama.EmbedModel)
			return nil
		},
	}
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
	_, err = f.WriteString(line)
	return err
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
	var asJSON bool
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
			query := strings.Join(args, " ")
			opts := search.Options{TopK: topK}
			if chunkType != "" {
				opts.Type = store.ChunkType(chunkType)
			}
			results, err := search.Search(cmd.Context(), st, client, query, opts)
			if err != nil {
				return err
			}
			if asJSON {
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
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	return cmd
}

func newDumpCmd() *cobra.Command {
	var (
		output    string
		scope     string
		chunkType string
		topK      int
	)
	cmd := &cobra.Command{
		Use:   "dump [query]",
		Short: "Dump indexed context for an LLM",
		Long: `Write retrieved index chunks as markdown you can paste into an LLM.

With a query, dump the closest matching chunks. Without a query, dump all
indexed chunks (optionally limited by --scope and --type).

Output:
  (default)     stdout, so you can pipe the dump into another command
  -o file.md    a single markdown file
  -o dumps/     one markdown file per source path, plus index.md
                (trailing slash, or an existing directory)

Examples:
  archivist dump "how does indexing work"
  archivist dump "auth middleware" -o context.md
  archivist dump --scope internal -o dumps/
  archivist dump --type adr
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

			query := strings.TrimSpace(strings.Join(args, " "))
			opts := dump.Options{
				Query:  query,
				Scope:  scope,
				TopK:   topK,
				Output: output,
			}
			if chunkType != "" {
				opts.Type = store.ChunkType(chunkType)
			}

			var embedder embed.Embedder
			if query != "" {
				client := embed.NewOllamaClientFromConfig(cfg.Ollama)
				if err := client.Healthy(cmd.Context()); err != nil {
					return fmt.Errorf("%w (run: ollama serve)", err)
				}
				embedder = client
			}

			result, err := dump.Collect(cmd.Context(), st, embedder, opts)
			if err != nil {
				return err
			}
			written, err := dump.Write(result, output, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			if written.Stdout {
				return nil
			}
			if len(written.Files) == 1 {
				fmt.Fprintf(cmd.ErrOrStderr(), "Wrote %s (%d chunks)\n", written.Files[0], len(result.Items))
				return nil
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Wrote %d files (%d chunks)\n", len(written.Files), len(result.Items))
			return nil
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "", "write to a file, a directory (trailing /), or stdout (default)")
	cmd.Flags().StringVar(&scope, "scope", "", "limit to a path prefix or glob")
	cmd.Flags().StringVar(&chunkType, "type", "", "filter by chunk type: code|doc|commit|adr|comment")
	cmd.Flags().IntVar(&topK, "top", 0, "max chunks (default 20 with a query, all without)")
	return cmd
}

func newStatusCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show index and embedder status",
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

			health := embed.CheckHealth(cmd.Context(), cfg)

			chunks, _ := st.ChunkCount()
			files, _ := st.FileCount()
			lastIdx, hasIdx, _ := st.LastIndexedAt()

			type status struct {
				EmbedderOK    bool   `json:"embedder_ok"`
				EmbedderError string `json:"embedder_error,omitempty"`
				ChunkCount    int    `json:"chunk_count"`
				FileCount     int    `json:"file_count"`
				LastIndexedAt string `json:"last_indexed_at,omitempty"`
			}
			s := status{
				EmbedderOK:    health.EmbedderOK,
				EmbedderError: health.EmbedderError,
				ChunkCount:    chunks,
				FileCount:     files,
			}
			if hasIdx {
				s.LastIndexedAt = lastIdx.Format("2006-01-02 15:04:05 UTC")
			}

			if asJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(s)
			}

			fmt.Printf("Embeddings (Ollama): ")
			if s.EmbedderOK {
				fmt.Println("ok")
			} else {
				fmt.Println("unavailable -", s.EmbedderError)
			}
			fmt.Printf("Chunks: %d (%d files)\n", s.ChunkCount, s.FileCount)
			if hasIdx {
				fmt.Printf("Last indexed: %s\n", s.LastIndexedAt)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	return cmd
}
