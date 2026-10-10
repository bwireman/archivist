// Package mcp serves the archive over MCP stdio.
package mcp

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/bwireman/archivist/internal/archive"
	"github.com/bwireman/archivist/internal/check"
	"github.com/bwireman/archivist/internal/cmdlog"
	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/retrieve"
	"github.com/bwireman/archivist/internal/store"
	"github.com/bwireman/archivist/internal/trace"
	"github.com/bwireman/archivist/internal/version"
	ruletmpl "github.com/bwireman/archivist/rules"
)

type Server struct {
	RepoRoot string
	Cfg      *config.Config
	RepoDB   *store.Store
	HomeDB   *store.Store
	Archive  *archive.Service
	Engine   *retrieve.Engine
	Embedder embed.Embedder
}

func New(repoRoot string, cfg *config.Config, repoDB, homeDB *store.Store, embedder embed.Embedder) *Server {
	return &Server{
		RepoRoot: repoRoot,
		Cfg:      cfg,
		RepoDB:   repoDB,
		HomeDB:   homeDB,
		Archive:  archive.New(repoRoot, cfg, repoDB, homeDB),
		Engine:   &retrieve.Engine{Repo: repoDB, Home: homeDB},
		Embedder: embedder,
	}
}

func (s *Server) MCPServer() *mcpserver.MCPServer {
	opts := []mcpserver.ServerOption{mcpserver.WithInstructions(agentInstructions())}
	if s != nil && s.Cfg != nil && s.Cfg.LogCommands {
		opts = append(opts, mcpserver.WithToolHandlerMiddleware(s.commandLogMiddleware()))
	}
	srv := mcpserver.NewMCPServer("archivist", version.Version, opts...)
	srv.AddTool(mcp.NewTool("search",
		annotate("Search archive", true, false, true),
		mcp.WithDescription("Search the knowledge archive before implementing or writing a record. Hybrid FTS + vectors; keyword-only if Ollama is down. Use this to reuse an existing decision, rule, or feature instead of creating a duplicate. Default hits omit the body. Pass full=true for every stored field. Call get to read a body."),
		mcp.WithString("query", mcp.Required()),
		mcp.WithString("type", mcp.Description("optional filter: decision, rule, feature, guide, map, pitfall")),
		mcp.WithString("scope"),
		mcp.WithString("status", mcp.Description("optional record status: proposed, accepted, deprecated, or superseded. Default omits deprecated and superseded.")),
		mcp.WithNumber("top_k"),
		mcp.WithBoolean("full", mcp.Description("When true, return every stored field. Default is a card without the body.")),
	), s.toolSearch)
	srv.AddTool(mcp.NewTool("get",
		annotate("Get record", true, false, true),
		mcp.WithDescription("Get one archive record by id or slug after search. Default returns id, slug, type, scope, title, status, severity, body, tags, applies_to, and superseded_by. Pass full=true for every stored field."),
		mcp.WithString("id", mcp.Required()),
		mcp.WithBoolean("full", mcp.Description("When true, return every stored field.")),
	), s.toolGet)
	srv.AddTool(mcp.NewTool("check",
		annotate("Check rules", true, false, true),
		mcp.WithDescription("Check rule records against a change (description, paths, diff)."),
		mcp.WithString("description"),
		mcp.WithString("paths"),
		mcp.WithString("diff"),
	), s.toolCheck)
	srv.AddTool(mcp.NewTool("map",
		annotate("Explore code map", true, false, true),
		mcp.WithDescription("Explore the code map: symbols and files matching the query, the imports those files declare, the files that import the query, and recent commits mentioning it. The top-level result is the process checkout. Configured archives are returned under archives and are not indexed by this call. Run `archivist index` in a checkout before reading its map."),
		mcp.WithString("query", mcp.Required()),
		mcp.WithNumber("limit", mcp.Description("max rows per section (default 30)")),
	), s.toolMap)
	srv.AddTool(mcp.NewTool("remember",
		annotate("Remember record", false, true, true),
		mcp.WithDescription("Create a record only after search shows a gap. Distill a lasting decision, rule, feature, guide, map, or pitfall from this conversation — not a chat transcript, session error, or restatement of an existing record. Writes SQLite only; does not create a markdown file."),
		mcp.WithString("type", mcp.Required(), mcp.Description("decision, rule, feature, guide, map, or pitfall")),
		mcp.WithString("scope", mcp.Required(), mcp.Description("repo, global, or dev")),
		mcp.WithString("title", mcp.Required()),
		mcp.WithString("body", mcp.Required(), mcp.Description("short distilled markdown; not a chat log")),
		mcp.WithString("severity"),
		mcp.WithString("applies_to"),
		mcp.WithString("tags"),
		mcp.WithString("archive", mcp.Description("listed checkout this repo record is about; omit when that context is the process checkout")),
	), s.toolRemember)
	srv.AddTool(mcp.NewTool("update",
		annotate("Update record", false, true, true),
		mcp.WithDescription("Update an existing record in place when the same topic already has a current document. Prefer this over remember for refinements."),
		mcp.WithString("id", mcp.Required()),
		mcp.WithString("title"),
		mcp.WithString("body"),
		mcp.WithString("status"),
	), s.toolUpdate)
	srv.AddTool(mcp.NewTool("retire",
		annotate("Retire record", false, true, true),
		mcp.WithDescription("Mark a record superseded when a later choice replaces it. Leave a stub plus superseded_by; do not keep two accepted documents on the same topic."),
		mcp.WithString("id", mcp.Required()),
		mcp.WithString("superseded_by"),
	), s.toolRetire)
	srv.AddTool(mcp.NewTool("cite",
		annotate("Cite record", false, false, false),
		mcp.WithDescription("Call before the turn ends when a retrieved record changed the work. id is a record id or slug; effect is one line. Do not cite a hit you did not use. Requires log_commands."),
		mcp.WithString("id", mcp.Required()),
		mcp.WithString("effect", mcp.Required(), mcp.Description("one line: what the record changed")),
	), s.toolCite)
	srv.AddTool(mcp.NewTool("status",
		annotate("Archive status", true, false, true),
		mcp.WithDescription("Archive and embedder status (record count, queue depth, Ollama health)."),
	), s.toolStatus)
	srv.AddTool(mcp.NewTool("import",
		annotate("Import markdown", false, true, true),
		mcp.WithDescription("Import typed markdown from record directories and export copies into SQLite. Does not delete DB-only records."),
	), s.toolImport)
	registerAddPrompts(srv)
	return srv
}

// annotate sets MCP ToolAnnotations. Unset hints default to read-only false,
// destructive true, idempotent false, and open-world true, so every tool sets
// all four. Archivist tools are closed-world: local SQLite and one configured
// embedder. destructive and idempotent still apply when readOnly is false:
// remember, update, retire, and import may replace record content, and a
// repeat whose content hash matches is a no-op. cite only appends a log line.
func annotate(title string, readOnly, destructive, idempotent bool) mcp.ToolOption {
	return mcp.WithToolAnnotation(toolAnnotation(title, readOnly, destructive, idempotent))
}

func toolAnnotation(title string, readOnly, destructive, idempotent bool) mcp.ToolAnnotation {
	return mcp.ToolAnnotation{
		Title:           title,
		ReadOnlyHint:    new(readOnly),
		DestructiveHint: new(destructive),
		IdempotentHint:  new(idempotent),
		OpenWorldHint:   new(false),
	}
}

func (s *Server) commandLogMiddleware() mcpserver.ToolHandlerMiddleware {
	log := cmdlog.FromConfig(s.RepoRoot, s.Cfg)
	return func(next mcpserver.ToolHandlerFunc) mcpserver.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			start := time.Now()
			name := req.Params.Name
			log.In("mcp", name, req.GetArguments())
			res, err := next(ctx, req)
			log.Out("mcp", name, mcpLogResult(res), err, start)
			return res, err
		}
	}
}

func mcpLogResult(res *mcp.CallToolResult) any {
	if res == nil {
		return nil
	}
	var b strings.Builder
	for _, c := range res.Content {
		switch t := c.(type) {
		case mcp.TextContent:
			b.WriteString(t.Text)
		case *mcp.TextContent:
			if t != nil {
				b.WriteString(t.Text)
			}
		}
	}
	text := b.String()
	if res.IsError {
		return map[string]any{"is_error": true, "text": text}
	}
	if text == "" {
		return nil
	}
	var parsed any
	if json.Unmarshal([]byte(text), &parsed) == nil {
		return parsed
	}
	return text
}

// agentInstructions is returned on MCP initialize so hosts without skills
// install still consult the archive, cite a record that changed the work, ask when a gap is unsettled, and distill lasting facts.
func agentInstructions() string {
	return mustRule("consult.md") + "\n\n" + mustRule("cite.md") + "\n\n" + mustRule("ask.md") + "\n\n" + mustRule("record.md")
}

func mustRule(name string) string {
	b, err := ruletmpl.FS.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return strings.TrimSpace(string(b))
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(string(data)), nil
}

func (s *Server) toolSearch(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := req.GetString("query", "")
	topK := int(req.GetFloat("top_k", float64(retrieve.DefaultTopK)))
	opts := retrieve.Options{Query: query, TopK: topK}
	if t := req.GetString("type", ""); t != "" {
		opts.Type = record.Type(t)
	}
	if sc := req.GetString("scope", ""); sc != "" {
		opts.Scope = record.Scope(sc)
	}
	if st := req.GetString("status", ""); st != "" {
		opts.Status = record.Status(st)
	}
	results, err := s.Engine.Search(ctx, s.Embedder, opts)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if req.GetBool("full", false) {
		return jsonResult(results)
	}
	return jsonResult(retrieve.ProjectSearch(results))
}

func (s *Server) toolGet(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rec, err := s.Archive.Get(req.GetString("id", ""))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if req.GetBool("full", false) {
		return jsonResult(rec)
	}
	return jsonResult(retrieve.ProjectRecord(rec))
}

func (s *Server) toolCheck(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var paths []string
	if p := req.GetString("paths", ""); p != "" {
		paths = check.SplitPathList(p)
	}
	res, err := check.Run(ctx, &retrieve.Engine{Repo: s.RepoDB, Home: s.HomeDB}, s.Embedder, check.Options{
		Description: req.GetString("description", ""),
		Paths:       paths,
		Diff:        req.GetString("diff", ""),
	})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return jsonResult(res)
}

func (s *Server) toolMap(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	limit := int(req.GetFloat("limit", float64(store.DefaultExploreLimit)))
	query := req.GetString("query", "")
	primary, err := s.RepoDB.ExploreCode(query, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	res := store.MapResult{CodeSearch: primary}
	if s.Archive != nil {
		for _, ex := range s.Archive.Extras {
			if ex.DB == nil {
				continue
			}
			section, err := ex.DB.ExploreCode(query, limit)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			res.Archives = append(res.Archives, store.RootedCodeSearch{Root: ex.Root, CodeSearch: section})
		}
	}
	return jsonResult(res)
}

func (s *Server) toolRemember(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rec := &record.Record{
		Type:   record.Type(req.GetString("type", "")),
		Scope:  record.Scope(req.GetString("scope", "")),
		Title:  req.GetString("title", ""),
		Body:   req.GetString("body", ""),
		Status: record.StatusAccepted,
	}
	if sev := req.GetString("severity", ""); sev != "" {
		rec.Severity = record.Severity(sev)
	}
	if a := req.GetString("applies_to", ""); a != "" {
		rec.AppliesTo = check.SplitPathList(a)
	}
	if tags := req.GetString("tags", ""); tags != "" {
		rec.Tags = check.SplitPathList(tags)
	}
	id, err := s.Archive.Remember(rec, req.GetString("archive", ""))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return jsonResult(map[string]string{"id": id})
}

func (s *Server) toolUpdate(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	err := s.Archive.Update(id, func(r *record.Record) error {
		archive.PatchText(r, req.GetString("title", ""), req.GetString("body", ""), req.GetString("status", ""))
		return nil
	})
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText("ok"), nil
}

func (s *Server) toolRetire(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	err := s.Archive.Retire(req.GetString("id", ""), req.GetString("superseded_by", ""))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText("ok"), nil
}

func (s *Server) toolImport(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	res, err := s.Archive.Import()
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return jsonResult(map[string]int{
		"imported": res.Imported,
		"updated":  res.Updated,
		"skipped":  res.Skipped,
	})
}

func (s *Server) toolCite(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	effect, err := trace.PrepareCite(s != nil && s.Cfg != nil && s.Cfg.LogCommands, req.GetString("effect", ""))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("cite id is required"), nil
	}
	rec, err := s.Archive.Get(id)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return jsonResult(map[string]string{"id": rec.ID, "effect": effect})
}

func (s *Server) toolStatus(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	health := embed.CheckHealth(context.Background(), s.Cfg)
	count, _ := s.RepoDB.RecordCount()
	queue, _ := s.RepoDB.QueueDepth()
	if s.HomeDB != nil {
		homeCount, _ := s.HomeDB.RecordCount()
		homeQueue, _ := s.HomeDB.QueueDepth()
		count += homeCount
		queue += homeQueue
	}
	status := map[string]any{
		"version":      version.String(),
		"record_count": count,
		"queue_depth":  queue,
		"embedder_ok":  health.EmbedderOK,
	}
	if s.Archive != nil && len(s.Archive.Extras) > 0 {
		extras := make([]map[string]any, 0, len(s.Archive.Extras))
		for _, ex := range s.Archive.Extras {
			if ex.DB == nil {
				continue
			}
			n, _ := ex.DB.RecordCount()
			extras = append(extras, map[string]any{"root": ex.Root, "record_count": n})
		}
		status["extras"] = extras
	}
	if health.EmbedderError != "" {
		status["embedder_error"] = health.EmbedderError
	}
	return jsonResult(status)
}

func ServeStdio(s *Server) error {
	return mcpserver.ServeStdio(s.MCPServer())
}
