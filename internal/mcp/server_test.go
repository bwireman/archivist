package mcp

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/bwireman/archivist/internal/cmdlog"
	"github.com/bwireman/archivist/internal/config"
	ruletmpl "github.com/bwireman/archivist/rules"
)

func TestAgentInstructionsMatchRuleTemplates(t *testing.T) {
	consult, err := ruletmpl.FS.ReadFile("consult.md")
	if err != nil {
		t.Fatal(err)
	}
	record, err := ruletmpl.FS.ReadFile("record.md")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(string(consult)) + "\n\n" + strings.TrimSpace(string(record))
	got := agentInstructions()
	if got != want {
		t.Fatalf("MCP instructions drifted from rules/consult.md + rules/record.md")
	}
	for _, needle := range []string{
		"Consult the archive",
		"Scan this conversation",
		"Do not wait for \"remember this.\"",
		"Search first",
	} {
		if !strings.Contains(got, needle) {
			t.Fatalf("instructions missing %q", needle)
		}
	}
}

func TestToolDescriptionsPreferDistillOverGlut(t *testing.T) {
	srv := (&Server{}).MCPServer()
	remember := srv.GetTool("remember")
	if remember == nil {
		t.Fatal("missing remember tool")
	}
	desc := remember.Tool.Description
	for _, needle := range []string{"search shows a gap", "not a chat transcript"} {
		if !strings.Contains(desc, needle) {
			t.Fatalf("remember description missing %q: %s", needle, desc)
		}
	}
	update := srv.GetTool("update")
	if update == nil || !strings.Contains(update.Tool.Description, "Prefer this over remember") {
		t.Fatalf("update should prefer in-place edits: %+v", update)
	}
	search := srv.GetTool("search")
	if search == nil || !strings.Contains(search.Tool.Description, "before implementing or writing") {
		t.Fatalf("search should run before remember: %+v", search)
	}
}

func TestCommandLogMiddlewareWritesJSONL(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	cfg.LogCommands = true
	s := &Server{RepoRoot: dir, Cfg: cfg}
	h := s.commandLogMiddleware()(func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		return mcp.NewToolResultText(`{"ok":true}`), nil
	})
	req := mcp.CallToolRequest{}
	req.Params.Name = "status"
	req.Params.Arguments = map[string]any{"ping": true}
	if _, err := h(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(config.CommandsLogPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d\n%s", len(lines), data)
	}
	var in, out cmdlog.Entry
	if err := json.Unmarshal([]byte(lines[0]), &in); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &out); err != nil {
		t.Fatal(err)
	}
	if in.Dir != "in" || in.Source != "mcp" || in.Command != "status" {
		t.Fatalf("in: %+v", in)
	}
	if out.Dir != "out" || out.Error != "" {
		t.Fatalf("out: %+v", out)
	}
}

func TestMcpLogResultParsesJSON(t *testing.T) {
	got := mcpLogResult(mcp.NewToolResultText(`{"id":"rec_1"}`))
	m, ok := got.(map[string]any)
	if !ok || m["id"] != "rec_1" {
		t.Fatalf("got %#v", got)
	}
	errRes := mcp.NewToolResultError("nope")
	got = mcpLogResult(errRes)
	m, ok = got.(map[string]any)
	if !ok || m["is_error"] != true {
		t.Fatalf("error result: %#v", got)
	}
}
