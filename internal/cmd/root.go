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
	"github.com/bwireman/archivist/internal/search"
	"github.com/bwireman/archivist/internal/store"
	"github.com/bwireman/archivist/internal/version"
	"github.com/spf13/cobra"
)

var (
	repoPath string
)

func NewRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "archivist",
		Short:         "Local code indexer",
		Version:       version.String(),
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetVersionTemplate("{{.Name}} version {{.Version}}\n")
	root.PersistentFlags().StringVar(&repoPath, "path", ".", "repository root path")
	root.AddCommand(newInitCmd())
	root.AddCommand(newIndexCmd())
	root.AddCommand(newSearchCmd())
	root.AddCommand(newDumpCmd())
	root.AddCommand(newStatusCmd())
	root.AddCommand(newVersionCmd())
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

func openStore(root string, cfg *config.Config) (*store.Store, error) {
	return store.Open(config.StorePath(root, cfg))
}

func openGlobalStore(cfg *config.Config) (*store.Store, error) {
	path, err := resolveGlobalStorePath(cfg)
	if err != nil {
		return nil, err
	}
	return store.Open(path)
}

func openGlobalStoreExisting(cfg *config.Config) (*store.Store, error) {
	path, err := resolveGlobalStorePath(cfg)
	if err != nil {
		return nil, err
	}
	st, ok, err := store.OpenIfExists(path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("global index not found at %s (run archivist index)", path)
	}
	return st, nil
}

func resolveGlobalStorePath(cfg *config.Config) (string, error) {
	path := config.GlobalStorePath(cfg)
	if path != "" {
		return path, nil
	}
	if cfg != nil {
		if p := strings.TrimSpace(cfg.Store.GlobalPath); p != "" && !filepath.IsAbs(p) {
			return "", fmt.Errorf("store.global_path %q escapes ~/.archivist; use an absolute path", p)
		}
	}
	return "", fmt.Errorf("cannot resolve global index path (set store.global_path or $HOME)")
}

// openStoreForQuery uses the repo index unless --adr-scope global, which
// reads the machine-wide ADR database.
func openStoreForQuery(root string, cfg *config.Config, adrScope string) (*store.Store, error) {
	if adrScope == "global" {
		return openGlobalStoreExisting(cfg)
	}
	return openStore(root, cfg)
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

func newSearchCmd() *cobra.Command {
	var topK int
	var chunkType string
	var adrScope string
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
			parsed, err := parseADRScope(adrScope)
			if err != nil {
				return err
			}
			st, err := openStoreForQuery(root, cfg, parsed)
			if err != nil {
				return err
			}
			defer st.Close()

			client := embed.NewOllamaClientFromConfig(cfg.Ollama)
			query := strings.Join(args, " ")
			opts := search.Options{TopK: topK, ADRScope: parsed}
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
	cmd.Flags().IntVar(&topK, "top", search.DefaultTopK, "number of results")
	cmd.Flags().StringVar(&chunkType, "type", "", "filter by chunk type: code|doc|commit|adr|comment")
	cmd.Flags().StringVar(&adrScope, "adr-scope", "", "repo index (default/repo) or machine-wide global.db (global)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	return cmd
}

func newDumpCmd() *cobra.Command {
	var (
		output    string
		scope     string
		adrScope  string
		chunkType string
		topK      int
	)
	cmd := &cobra.Command{
		Use:   "dump [query]",
		Short: "Dump indexed context for an LLM",
		Long: `Write retrieved index chunks as markdown you can paste into an LLM.

With a query, dump the closest matching chunks. Without a query, dump all
indexed chunks (optionally limited by --scope, --type, and --adr-scope).
Default search and dump use the repo index only. --adr-scope global reads
the machine-wide ADR database (~/.archivist/global.db).

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
  archivist dump --type adr --adr-scope repo
  archivist dump --type adr --adr-scope global
`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, cfg, err := loadEnv()
			if err != nil {
				return err
			}
			parsed, err := parseADRScope(adrScope)
			if err != nil {
				return err
			}
			st, err := openStoreForQuery(root, cfg, parsed)
			if err != nil {
				return err
			}
			defer st.Close()

			query := strings.TrimSpace(strings.Join(args, " "))
			opts := dump.Options{
				Query:    query,
				Scope:    scope,
				ADRScope: parsed,
				TopK:     topK,
				Output:   output,
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
	cmd.Flags().StringVar(&adrScope, "adr-scope", "", "repo index (default/repo) or machine-wide global.db (global)")
	cmd.Flags().StringVar(&chunkType, "type", "", "filter by chunk type: code|doc|commit|adr|comment")
	cmd.Flags().IntVar(&topK, "top", 0, "max chunks (default 40 with a query, all without)")
	return cmd
}

func parseADRScope(s string) (string, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "", "repo", "global":
		return s, nil
	default:
		return "", fmt.Errorf("--adr-scope must be repo or global")
	}
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
			schema, _ := st.SchemaVersion()
			indexedBy, _, _ := st.ArchivistVersion()
			lastSearch, hasSearch, _ := st.LastSearch()
			lastSearchAt, hasSearchAt, _ := st.LastSearchAt()

			var (
				globalPath          string
				globalChunks        int
				globalFiles         int
				globalLast          string
				hasGlobalIdx        bool
				globalSchema        int
				globalIndexedBy     string
				globalLastSearch    string
				globalLastSearchAt  string
				hasGlobalLastSearch bool
			)
			gpath, gpathErr := resolveGlobalStorePath(cfg)
			if gpathErr != nil {
				return gpathErr
			}
			if gst, ok, gerr := store.OpenIfExists(gpath); gerr != nil {
				return gerr
			} else if ok {
				defer gst.Close()
				globalPath = gpath
				globalChunks, _ = gst.ChunkCount()
				globalFiles, _ = gst.FileCount()
				globalSchema, _ = gst.SchemaVersion()
				globalIndexedBy, _, _ = gst.ArchivistVersion()
				glast, ghas, _ := gst.LastIndexedAt()
				hasGlobalIdx = ghas
				if ghas {
					globalLast = glast.Format("2006-01-02 15:04:05 UTC")
				}
				if q, ok, _ := gst.LastSearch(); ok {
					globalLastSearch = q
					hasGlobalLastSearch = true
				}
				if at, ok, _ := gst.LastSearchAt(); ok {
					globalLastSearchAt = at.Format("2006-01-02 15:04:05 UTC")
				}
			}

			type status struct {
				Version            string `json:"version"`
				Schema             int    `json:"schema"`
				IndexSchema        int    `json:"index_schema"`
				IndexedBy          string `json:"indexed_by,omitempty"`
				EmbedderOK         bool   `json:"embedder_ok"`
				EmbedderError      string `json:"embedder_error,omitempty"`
				ChunkCount         int    `json:"chunk_count"`
				FileCount          int    `json:"file_count"`
				LastIndexedAt      string `json:"last_indexed_at,omitempty"`
				LastSearch         string `json:"last_search,omitempty"`
				LastSearchAt       string `json:"last_search_at,omitempty"`
				GlobalPath         string `json:"global_path,omitempty"`
				GlobalSchema       int    `json:"global_schema,omitempty"`
				GlobalIndexedBy    string `json:"global_indexed_by,omitempty"`
				GlobalChunkCount   int    `json:"global_chunk_count"`
				GlobalFileCount    int    `json:"global_file_count"`
				GlobalLastIndexed  string `json:"global_last_indexed_at,omitempty"`
				GlobalLastSearch   string `json:"global_last_search,omitempty"`
				GlobalLastSearchAt string `json:"global_last_search_at,omitempty"`
			}
			s := status{
				Version:            version.Version,
				Schema:             version.Schema,
				IndexSchema:        schema,
				IndexedBy:          indexedBy,
				EmbedderOK:         health.EmbedderOK,
				EmbedderError:      health.EmbedderError,
				ChunkCount:         chunks,
				FileCount:          files,
				GlobalPath:         globalPath,
				GlobalSchema:       globalSchema,
				GlobalIndexedBy:    globalIndexedBy,
				GlobalChunkCount:   globalChunks,
				GlobalFileCount:    globalFiles,
				GlobalLastIndexed:  globalLast,
				GlobalLastSearch:   globalLastSearch,
				GlobalLastSearchAt: globalLastSearchAt,
			}
			if hasIdx {
				s.LastIndexedAt = lastIdx.Format("2006-01-02 15:04:05 UTC")
			}
			if hasSearch {
				s.LastSearch = lastSearch
			}
			if hasSearchAt {
				s.LastSearchAt = lastSearchAt.Format("2006-01-02 15:04:05 UTC")
			}

			if asJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(s)
			}

			fmt.Printf("Version: %s\n", version.String())
			if s.IndexSchema != 0 {
				fmt.Printf("Index schema: %d", s.IndexSchema)
				if s.IndexedBy != "" {
					fmt.Printf(" (written by %s)", s.IndexedBy)
				}
				fmt.Println()
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
			if s.LastSearch != "" {
				fmt.Printf("Last search: %s\n", s.LastSearch)
			}
			if s.LastSearchAt != "" {
				fmt.Printf("Last search at: %s\n", s.LastSearchAt)
			}
			if s.GlobalPath != "" {
				fmt.Printf("Global ADRs: %d chunks (%d files) at %s\n", s.GlobalChunkCount, s.GlobalFileCount, s.GlobalPath)
				if hasGlobalIdx {
					fmt.Printf("Global last indexed: %s\n", s.GlobalLastIndexed)
				}
				if hasGlobalLastSearch {
					fmt.Printf("Global last search: %s\n", s.GlobalLastSearch)
				}
				if s.GlobalLastSearchAt != "" {
					fmt.Printf("Global last search at: %s\n", s.GlobalLastSearchAt)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output JSON")
	return cmd
}
