package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/bwireman/archivist/internal/archive"
	"github.com/bwireman/archivist/internal/check"
	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/embed"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/retrieve"
	"github.com/bwireman/archivist/internal/store"
	"github.com/bwireman/archivist/internal/version"
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
		Archive:  &archive.Service{RepoRoot: repoRoot, RepoDB: repoDB, HomeDB: homeDB},
		Engine:   &retrieve.Engine{Repo: repoDB, Home: homeDB},
		Embedder: embedder,
	}
}

func (s *Server) MCPServer() *mcpserver.MCPServer {
	srv := mcpserver.NewMCPServer("archivist", version.Version)
	srv.AddTool(mcp.NewTool("search",
		mcp.WithDescription("Search the knowledge archive"),
		mcp.WithString("query", mcp.Required()),
		mcp.WithString("type"),
		mcp.WithString("scope"),
		mcp.WithNumber("top_k"),
	), s.toolSearch)
	srv.AddTool(mcp.NewTool("get",
		mcp.WithDescription("Get a record by id or slug"),
		mcp.WithString("id", mcp.Required()),
	), s.toolGet)
	srv.AddTool(mcp.NewTool("check",
		mcp.WithDescription("Check rules against a change"),
		mcp.WithString("description"),
		mcp.WithString("paths"),
		mcp.WithString("diff"),
	), s.toolCheck)
	srv.AddTool(mcp.NewTool("map",
		mcp.WithDescription("Find where code lives"),
		mcp.WithString("query", mcp.Required()),
	), s.toolMap)
	srv.AddTool(mcp.NewTool("remember",
		mcp.WithDescription("Create a new archive record"),
		mcp.WithString("type", mcp.Required()),
		mcp.WithString("scope", mcp.Required()),
		mcp.WithString("title", mcp.Required()),
		mcp.WithString("body", mcp.Required()),
		mcp.WithString("severity"),
		mcp.WithString("applies_to"),
		mcp.WithString("tags"),
	), s.toolRemember)
	srv.AddTool(mcp.NewTool("update",
		mcp.WithDescription("Update an existing record"),
		mcp.WithString("id", mcp.Required()),
		mcp.WithString("title"),
		mcp.WithString("body"),
		mcp.WithString("status"),
	), s.toolUpdate)
	srv.AddTool(mcp.NewTool("retire",
		mcp.WithDescription("Mark a record superseded"),
		mcp.WithString("id", mcp.Required()),
		mcp.WithString("superseded_by"),
	), s.toolRetire)
	srv.AddTool(mcp.NewTool("status",
		mcp.WithDescription("Archive and embedder status"),
	), s.toolStatus)
	return srv
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
	results, err := s.Engine.Search(ctx, s.Embedder, opts)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return jsonResult(results)
}

func (s *Server) toolGet(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	rec, err := s.Archive.Get(req.GetString("id", ""))
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return jsonResult(rec)
}

func (s *Server) toolCheck(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var paths []string
	if p := req.GetString("paths", ""); p != "" {
		paths = []string{p}
	}
	res, err := check.Run(ctx, s.Engine, s.Embedder, check.Options{
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
	query := req.GetString("query", "")
	syms, err := s.RepoDB.SearchSymbols(query, 30)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return jsonResult(syms)
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
		rec.AppliesTo = []string{a}
	}
	id, err := s.Archive.Remember(rec)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return jsonResult(map[string]string{"id": id})
}

func (s *Server) toolUpdate(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	err := s.Archive.Update(id, func(r *record.Record) error {
		if t := req.GetString("title", ""); t != "" {
			r.Title = t
		}
		if b := req.GetString("body", ""); b != "" {
			r.Body = b
		}
		if st := req.GetString("status", ""); st != "" {
			r.Status = record.Status(st)
		}
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

func (s *Server) toolStatus(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	health := embed.CheckHealth(context.Background(), s.Cfg)
	count, _ := s.RepoDB.RecordCount()
	queue, _ := s.RepoDB.QueueDepth()
	status := map[string]any{
		"version":      version.String(),
		"record_count": count,
		"queue_depth":  queue,
		"embedder_ok":  health.EmbedderOK,
	}
	if health.EmbedderError != "" {
		status["embedder_error"] = health.EmbedderError
	}
	return jsonResult(status)
}

func ServeStdio(s *Server) error {
	return mcpserver.ServeStdio(s.MCPServer())
}

func ServeHTTP(s *Server, addr string) error {
	return fmt.Errorf("http transport: use stdio for now; addr was %s", addr)
}
