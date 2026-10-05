package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/bwireman/archivist/internal/cmdlog"
	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/store"
	ruletmpl "github.com/bwireman/archivist/rules"
)

func TestAgentInstructionsMatchRuleTemplates(t *testing.T) {
	consult, err := ruletmpl.FS.ReadFile("consult.md")
	if err != nil {
		t.Fatal(err)
	}
	cite, err := ruletmpl.FS.ReadFile("cite.md")
	if err != nil {
		t.Fatal(err)
	}
	ask, err := ruletmpl.FS.ReadFile("ask.md")
	if err != nil {
		t.Fatal(err)
	}
	record, err := ruletmpl.FS.ReadFile("record.md")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.TrimSpace(string(consult)) + "\n\n" + strings.TrimSpace(string(cite)) + "\n\n" + strings.TrimSpace(string(ask)) + "\n\n" + strings.TrimSpace(string(record))
	got := agentInstructions()
	if got != want {
		t.Fatalf("MCP instructions drifted from rules/consult.md + rules/cite.md + rules/ask.md + rules/record.md")
	}
	for _, needle := range []string{
		"Consult the archive",
		"Ask when unsure",
		"Scan this conversation",
		"Do not wait for \"remember this.\"",
		"Search first",
		"call cite",
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

func TestToolAnnotations(t *testing.T) {
	srv := (&Server{}).MCPServer()
	want := map[string]mcp.ToolAnnotation{
		"search":   toolAnnotation("Search archive", true, false, true),
		"get":      toolAnnotation("Get record", true, false, true),
		"check":    toolAnnotation("Check rules", true, false, true),
		"map":      toolAnnotation("Explore code map", true, false, true),
		"status":   toolAnnotation("Archive status", true, false, true),
		"remember": toolAnnotation("Remember record", false, true, true),
		"update":   toolAnnotation("Update record", false, true, true),
		"retire":   toolAnnotation("Retire record", false, true, true),
		"import":   toolAnnotation("Import markdown", false, true, true),
		"cite":     toolAnnotation("Cite record", false, false, false),
	}
	listed := srv.ListTools()
	if len(listed) != len(want) {
		t.Fatalf("tool count %d, want %d", len(listed), len(want))
	}
	for name, hint := range want {
		tool := srv.GetTool(name)
		if tool == nil {
			t.Fatalf("missing %s", name)
		}
		got := tool.Tool.Annotations
		if got.Title != hint.Title || !sameBool(got.ReadOnlyHint, hint.ReadOnlyHint) ||
			!sameBool(got.DestructiveHint, hint.DestructiveHint) ||
			!sameBool(got.IdempotentHint, hint.IdempotentHint) ||
			!sameBool(got.OpenWorldHint, hint.OpenWorldHint) {
			t.Fatalf("%s annotations: %+v", name, got)
		}
	}
}

func sameBool(a, b *bool) bool {
	return a != nil && b != nil && *a == *b
}

func TestCommandLogMiddlewareWritesJSONL(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	cfg.LogCommands = true
	s := &Server{RepoRoot: dir, Cfg: cfg}
	h := s.commandLogMiddleware()(func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func TestCiteRequiresLogCommands(t *testing.T) {
	s := &Server{Cfg: config.Default()}
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"id": "rec_1", "effect": "kept the store"}
	res, err := s.toolCite(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("expected cite error, got %+v", res)
	}
	got := mcpLogResult(res)
	m, ok := got.(map[string]any)
	text, _ := m["text"].(string)
	if !ok || !strings.Contains(text, "log_commands is false") {
		t.Fatalf("cite error: %#v", got)
	}
}

func TestToolRememberSplitsAppliesToAndTags(t *testing.T) {
	repo := t.TempDir()
	repoDB, err := store.Open(filepath.Join(t.TempDir(), "repo.db"))
	if err != nil {
		t.Fatal(err)
	}
	homeDB, err := store.Open(filepath.Join(t.TempDir(), "home.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = repoDB.Close()
		_ = homeDB.Close()
	})
	s := New(repo, config.Default(), repoDB, homeDB, nil)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"type":       "rule",
		"scope":      "repo",
		"title":      "Split fields",
		"body":       "Keep lists as lists.",
		"severity":   "must",
		"applies_to": "internal/check/**, internal/cmd/**",
		"tags":       "check,strict",
	}
	res, err := s.toolRemember(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if res.IsError {
		t.Fatalf("remember error: %+v", res)
	}
	rec, err := s.Archive.Get("split-fields")
	if err != nil {
		t.Fatal(err)
	}
	if len(rec.AppliesTo) != 2 || rec.AppliesTo[0] != "internal/check/**" || rec.AppliesTo[1] != "internal/cmd/**" {
		t.Fatalf("applies_to %v", rec.AppliesTo)
	}
	if len(rec.Tags) != 2 || rec.Tags[0] != "check" || rec.Tags[1] != "strict" {
		t.Fatalf("tags %v", rec.Tags)
	}
}

func TestSearchAndGetDefaultCompactFullRestoresRecord(t *testing.T) {
	srv := (&Server{}).MCPServer()
	searchTool := srv.GetTool("search")
	if searchTool == nil {
		t.Fatal("missing search")
	}
	searchDesc := searchTool.Tool.Description
	if !strings.Contains(searchDesc, "before implementing or writing") || !strings.Contains(searchDesc, "omit the body") {
		t.Fatalf("search description: %s", searchDesc)
	}
	getTool := srv.GetTool("get")
	if getTool == nil || !strings.Contains(getTool.Tool.Description, "full") {
		t.Fatalf("get description: %+v", getTool)
	}

	repo := t.TempDir()
	repoDB, err := store.Open(filepath.Join(t.TempDir(), "repo.db"))
	if err != nil {
		t.Fatal(err)
	}
	homeDB, err := store.Open(filepath.Join(t.TempDir(), "home.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = repoDB.Close()
		_ = homeDB.Close()
	})
	s := New(repo, config.Default(), repoDB, homeDB, nil)
	remembered := callTool(t, s, "remember", map[string]any{
		"type":  "feature",
		"scope": "repo",
		"title": "Zqxcard fixture",
		"body":  "BODYTOKEN lives here",
	})
	var created map[string]string
	if err := json.Unmarshal([]byte(remembered), &created); err != nil {
		t.Fatal(err)
	}
	id := created["id"]

	compactSearch := callTool(t, s, "search", map[string]any{"query": "Zqxcard"})
	if strings.Contains(compactSearch, "BODYTOKEN") || strings.Contains(compactSearch, "ContentHash") {
		t.Fatalf("compact search leaked fields: %s", compactSearch)
	}
	var hits []map[string]any
	if err := json.Unmarshal([]byte(compactSearch), &hits); err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0]["title"] != "Zqxcard fixture" {
		t.Fatalf("compact search: %s", compactSearch)
	}
	assertJSONKeys(t, hits[0], "id", "slug", "type", "scope", "title", "status", "score", "source")

	compactGet := callTool(t, s, "get", map[string]any{"id": id})
	if !strings.Contains(compactGet, "BODYTOKEN") || strings.Contains(compactGet, "ContentHash") {
		t.Fatalf("compact get: %s", compactGet)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(compactGet), &got); err != nil {
		t.Fatal(err)
	}
	assertJSONKeys(t, got, "id", "slug", "type", "scope", "title", "status", "severity", "body", "tags", "applies_to", "superseded_by")

	fullSearch := callTool(t, s, "search", map[string]any{"query": "Zqxcard", "full": true})
	var fullHits []map[string]any
	if err := json.Unmarshal([]byte(fullSearch), &fullHits); err != nil {
		t.Fatal(err)
	}
	if len(fullHits) != 1 {
		t.Fatalf("full search: %s", fullSearch)
	}
	rec, _ := fullHits[0]["Record"].(map[string]any)
	if rec == nil || rec["ContentHash"] == "" || !strings.Contains(fmtBody(rec["Body"]), "BODYTOKEN") {
		t.Fatalf("full search record: %#v", fullHits[0])
	}

	fullGet := callTool(t, s, "get", map[string]any{"id": id, "full": true})
	var fullRec map[string]any
	if err := json.Unmarshal([]byte(fullGet), &fullRec); err != nil {
		t.Fatal(err)
	}
	if fullRec["ContentHash"] == "" || !strings.Contains(fmtBody(fullRec["Body"]), "BODYTOKEN") {
		t.Fatalf("full get: %s", fullGet)
	}
}

func callTool(t *testing.T, s *Server, name string, args map[string]any) string {
	t.Helper()
	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args
	var (
		res *mcp.CallToolResult
		err error
	)
	switch name {
	case "remember":
		res, err = s.toolRemember(context.Background(), req)
	case "search":
		res, err = s.toolSearch(context.Background(), req)
	case "get":
		res, err = s.toolGet(context.Background(), req)
	default:
		t.Fatalf("unknown tool %s", name)
	}
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.IsError {
		t.Fatalf("%s error: %+v", name, res)
	}
	var b strings.Builder
	for _, c := range res.Content {
		switch tc := c.(type) {
		case mcp.TextContent:
			b.WriteString(tc.Text)
		case *mcp.TextContent:
			if tc != nil {
				b.WriteString(tc.Text)
			}
		default:
			t.Fatalf("%s content %T", name, c)
		}
	}
	return b.String()
}

func assertJSONKeys(t *testing.T, m map[string]any, want ...string) {
	t.Helper()
	if len(m) != len(want) {
		t.Fatalf("keys %#v, want %v", m, want)
	}
	for _, k := range want {
		if _, ok := m[k]; !ok {
			t.Fatalf("missing %s in %#v", k, m)
		}
	}
}

func fmtBody(v any) string {
	s, _ := v.(string)
	return s
}
