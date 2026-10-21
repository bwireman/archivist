package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/bwireman/archivist/internal/archive"
	"github.com/bwireman/archivist/internal/cmdlog"
	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/record"
	"github.com/bwireman/archivist/internal/retrieve"
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
		"index":    toolAnnotation("Index code map", false, true, true),
		"embed":    toolAnnotation("Embed records", false, true, true),
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
	indexTool := srv.GetTool("index")
	if indexTool == nil || !strings.Contains(indexTool.Tool.Description, "Does not ingest or delete records") {
		t.Fatalf("index description: %+v", indexTool)
	}
	if _, ok := indexTool.Tool.InputSchema.Properties["scope"]; !ok {
		t.Fatal("index schema missing scope")
	}
	if slices.Contains(indexTool.Tool.InputSchema.Required, "scope") {
		t.Fatal("index scope should be optional")
	}
	embedTool := srv.GetTool("embed")
	if embedTool == nil || !strings.Contains(embedTool.Tool.Description, "once=true") {
		t.Fatalf("embed description: %+v", embedTool)
	}
	if _, ok := embedTool.Tool.InputSchema.Properties["once"]; !ok {
		t.Fatal("embed schema missing once")
	}
	if _, ok := embedTool.Tool.InputSchema.Properties["concurrency"]; !ok {
		t.Fatal("embed schema missing concurrency")
	}
	if slices.Contains(embedTool.Tool.InputSchema.Required, "once") || slices.Contains(embedTool.Tool.InputSchema.Required, "concurrency") {
		t.Fatal("embed once and concurrency should be optional")
	}
	status := srv.GetTool("status")
	if status == nil || !strings.Contains(status.Tool.Description, "missing_vectors") || !strings.Contains(status.Tool.Description, "call index") {
		t.Fatalf("status description: %+v", status)
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
	if in.ID == "" || in.ID != out.ID {
		t.Fatalf("call id in=%q out=%q", in.ID, out.ID)
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
	if !strings.Contains(searchDesc, "before implementing or writing") || !strings.Contains(searchDesc, "snippet") {
		t.Fatalf("search description: %s", searchDesc)
	}
	if _, ok := searchTool.Tool.InputSchema.Properties["scope"]; ok {
		t.Fatal("search schema has scope")
	}
	remember := srv.GetTool("remember")
	if remember == nil || !slices.Contains(remember.Tool.InputSchema.Required, "scope") {
		t.Fatal("remember must require scope")
	}
	getTool := srv.GetTool("get")
	if getTool == nil || !strings.Contains(getTool.Tool.Description, "full") {
		t.Fatalf("get description: %+v", getTool)
	}
	checkTool := srv.GetTool("check")
	if checkTool == nil || !strings.Contains(checkTool.Tool.Description, "Related") {
		t.Fatalf("check description: %+v", checkTool)
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
	created, _ := callTool(t, s, "remember", map[string]any{
		"type":  "feature",
		"scope": "repo",
		"title": "Zqxcard fixture",
		"body":  "BODYTOKEN lives here",
	}).(map[string]any)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("remember: %#v", created)
	}

	hits, _ := callTool(t, s, "search", map[string]any{"query": "Zqxcard"}).([]any)
	if len(hits) != 1 {
		t.Fatalf("compact search: %#v", hits)
	}
	hit, _ := hits[0].(map[string]any)
	if hit["title"] != "Zqxcard fixture" {
		t.Fatalf("compact search: %#v", hit)
	}
	assertJSONKeys(t, hit, "id", "slug", "type", "scope", "title", "status", "score", "source", "cosine", "applies_to", "superseded_by", "snippet")

	got, _ := callTool(t, s, "get", map[string]any{"id": id}).(map[string]any)
	if !strings.Contains(fmt.Sprint(got["body"]), "BODYTOKEN") {
		t.Fatalf("compact get: %#v", got)
	}
	assertJSONKeys(t, got, "id", "slug", "type", "scope", "title", "status", "severity", "body", "tags", "applies_to", "superseded_by")

	fullHits, _ := callTool(t, s, "search", map[string]any{"query": "Zqxcard", "full": true}).([]any)
	if len(fullHits) != 1 {
		t.Fatalf("full search: %#v", fullHits)
	}
	fullHit, _ := fullHits[0].(map[string]any)
	rec, _ := fullHit["Record"].(map[string]any)
	if rec == nil || rec["ContentHash"] == "" || !strings.Contains(fmt.Sprint(rec["Body"]), "BODYTOKEN") {
		t.Fatalf("full search record: %#v", fullHit)
	}

	fullRec, _ := callTool(t, s, "get", map[string]any{"id": id, "full": true}).(map[string]any)
	if fullRec["ContentHash"] == "" || !strings.Contains(fmt.Sprint(fullRec["Body"]), "BODYTOKEN") {
		t.Fatalf("full get: %#v", fullRec)
	}
}

func callTool(t *testing.T, s *Server, name string, args map[string]any) any {
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
	case "check":
		res, err = s.toolCheck(context.Background(), req)
	case "cite":
		res, err = s.toolCite(context.Background(), req)
	case "status":
		res, err = s.toolStatus(context.Background(), req)
	case "map":
		res, err = s.toolMap(context.Background(), req)
	case "index":
		res, err = s.toolIndex(context.Background(), req)
	case "embed":
		res, err = s.toolEmbed(context.Background(), req)
	default:
		t.Fatalf("unknown tool %s", name)
	}
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || res.IsError {
		t.Fatalf("%s error: %+v", name, res)
	}
	return mcpLogResult(res)
}

func TestCheckIgnoresExtraAndCiteRememberStatusUseIt(t *testing.T) {
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
	extraDir := t.TempDir()
	extraPath := config.StorePath(extraDir)
	extraDB, err := store.Open(extraPath)
	if err != nil {
		t.Fatal(err)
	}
	rule := &record.Record{
		ID: "rec_extra_rule", Slug: "extra-only-rule", Type: record.TypeRule, Scope: record.ScopeRepo,
		Title: "Zedextra rule only", Body: "Zedextra rule only applies in the other checkout",
		Status: record.StatusAccepted, Severity: record.SeverityMust,
		SourcePath: "docs/decisions/extra-only-rule.md",
	}
	if err := extraDB.UpsertRecord(rule); err != nil {
		t.Fatal(err)
	}
	if err := extraDB.Close(); err != nil {
		t.Fatal(err)
	}
	ro, err := store.OpenReadOnly(extraPath)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	canon, err := config.Canonical(extraDir)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Default()
	cfg.LogCommands = true
	s := New(repo, cfg, repoDB, homeDB, nil)
	s.Engine.Extras = []retrieve.Extra{{Root: canon, DB: ro}}
	s.Archive.Extras = []archive.Extra{{Root: canon, DB: ro}}
	s.Archive.ExtraRoots = []string{canon}

	checked := callTool(t, s, "check", map[string]any{"description": "Zedextra rule only"})
	raw, err := json.Marshal(checked)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "rec_extra_rule") {
		t.Fatalf("check saw extra rule: %s", raw)
	}

	before, err := ro.RecordCount()
	if err != nil {
		t.Fatal(err)
	}
	cited := callTool(t, s, "cite", map[string]any{"id": "rec_extra_rule", "effect": "kept the extra rule"}).(map[string]any)
	if cited["id"] != "rec_extra_rule" {
		t.Fatalf("cite %#v", cited)
	}
	after, err := ro.RecordCount()
	if err != nil || after != before {
		t.Fatalf("cite changed extra count %d -> %d err=%v", before, after, err)
	}

	created := callTool(t, s, "remember", map[string]any{
		"type": "decision", "scope": "repo", "title": "Written extra", "body": "only there",
		"archive": canon,
	}).(map[string]any)
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatalf("remember %#v", created)
	}
	if _, ok, err := repoDB.GetRecordByID(id); err != nil || ok {
		t.Fatalf("primary has remembered id ok=%v err=%v", ok, err)
	}

	status := callTool(t, s, "status", map[string]any{}).(map[string]any)
	if status["record_count"] != float64(0) && status["record_count"] != 0 {
		t.Fatalf("record_count %#v", status["record_count"])
	}
	extras, _ := status["extras"].([]any)
	if len(extras) != 1 {
		t.Fatalf("status %#v", status)
	}
	row, _ := extras[0].(map[string]any)
	if row["root"] != canon {
		t.Fatalf("extra status %#v", row)
	}
	count, _ := row["record_count"].(float64)
	if count < 1 {
		t.Fatalf("extra record_count %#v", row)
	}
}

func TestMapReadsExtraCodeMapWithoutWriting(t *testing.T) {
	repo := t.TempDir()
	repoDB, err := store.Open(config.StorePath(repo))
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := repoDB.ReplaceFileMap(
		store.FileRecord{Path: "primary.go", ContentHash: "p", IndexedAt: now},
		[]store.Symbol{{Name: "PrimaryWidget", Kind: "function_declaration", Line: 1, Exported: true}},
		nil,
	); err != nil {
		t.Fatal(err)
	}
	if err := repoDB.Close(); err != nil {
		t.Fatal(err)
	}
	extra := t.TempDir()
	extraPath := config.StorePath(extra)
	extraDB, err := store.Open(extraPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := extraDB.ReplaceFileMap(
		store.FileRecord{Path: "extra.go", ContentHash: "e", IndexedAt: now},
		[]store.Symbol{{Name: "ExtraWidget", Kind: "function_declaration", Line: 2, Exported: true}},
		nil,
	); err != nil {
		t.Fatal(err)
	}
	if err := extraDB.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(extraPath)
	if err != nil {
		t.Fatal(err)
	}
	ro, err := store.OpenReadOnly(extraPath)
	if err != nil {
		t.Fatal(err)
	}
	defer ro.Close()
	canon, err := config.Canonical(extra)
	if err != nil {
		t.Fatal(err)
	}
	primary, err := store.Open(config.StorePath(repo))
	if err != nil {
		t.Fatal(err)
	}
	defer primary.Close()
	s := New(repo, config.Default(), primary, nil, nil)
	s.Archive.Extras = []archive.Extra{{Root: canon, DB: ro}}

	got := callTool(t, s, "map", map[string]any{"query": "Widget"}).(map[string]any)
	names := symbolNames(t, got["symbols"])
	if !slices.Contains(names, "PrimaryWidget") || slices.Contains(names, "ExtraWidget") {
		t.Fatalf("top-level symbols %v", names)
	}
	archives, _ := got["archives"].([]any)
	if len(archives) != 1 {
		t.Fatalf("archives %#v", got["archives"])
	}
	row, _ := archives[0].(map[string]any)
	if row["root"] != canon {
		t.Fatalf("root %#v", row["root"])
	}
	extraNames := symbolNames(t, row["symbols"])
	if !slices.Contains(extraNames, "ExtraWidget") || slices.Contains(extraNames, "PrimaryWidget") {
		t.Fatalf("extra symbols %v", extraNames)
	}
	after, err := os.ReadFile(extraPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("map changed the extra database")
	}

	plain := New(repo, config.Default(), primary, nil, nil)
	noExtra := callTool(t, plain, "map", map[string]any{"query": "Widget"}).(map[string]any)
	if _, ok := noExtra["archives"]; ok {
		t.Fatalf("archives field present: %#v", noExtra["archives"])
	}
}

func symbolNames(t *testing.T, raw any) []string {
	t.Helper()
	rows, _ := raw.([]any)
	var names []string
	for _, row := range rows {
		sym, _ := row.(map[string]any)
		name, _ := sym["name"].(string)
		names = append(names, name)
	}
	return names
}

func TestMCPQueryReadsFrozenStore(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	if err := config.Save(root, config.Default()); err != nil {
		t.Fatal(err)
	}
	repo, err := store.Open(config.StorePath(root))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.UpsertRecord(&record.Record{
		ID: "rec_frozen", Slug: "frozen-query", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "Frozen query", Body: "readable", Status: record.StatusAccepted,
		SourcePath: "docs/features/frozen-query.md",
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Close(); err != nil {
		t.Fatal(err)
	}
	homeDB, err := store.Open(config.HomeStorePath())
	if err != nil {
		t.Fatal(err)
	}
	if err := homeDB.Close(); err != nil {
		t.Fatal(err)
	}

	repoRO, err := store.OpenReadOnly(config.StorePath(root))
	if err != nil {
		t.Fatal(err)
	}
	homeRO, err := store.OpenReadOnly(config.HomeStorePath())
	if err != nil {
		t.Fatal(err)
	}
	s := New(root, config.Default(), repoRO, homeRO, nil)
	hits, _ := callTool(t, s, "search", map[string]any{"query": "Frozen query"}).([]any)
	if len(hits) != 1 {
		t.Fatalf("search: %#v", hits)
	}
	created, _ := callTool(t, s, "remember", map[string]any{
		"type": "decision", "scope": "repo", "title": "Writable again", "body": "wrote",
	}).(map[string]any)
	if created["id"] == "" {
		t.Fatalf("remember: %#v", created)
	}
	if err := repoRO.Close(); err != nil {
		t.Fatal(err)
	}
	if err := homeRO.Close(); err != nil {
		t.Fatal(err)
	}

	dataDir := config.DataDir(root)
	t.Cleanup(func() {
		_ = os.Chmod(dataDir, 0o755)
		entries, _ := os.ReadDir(dataDir)
		for _, e := range entries {
			_ = os.Chmod(filepath.Join(dataDir, e.Name()), 0o644)
		}
	})
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if err := os.Chmod(filepath.Join(dataDir, e.Name()), 0o444); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Chmod(dataDir, 0o555); err != nil {
		t.Fatal(err)
	}
	repoRO, err = store.OpenReadOnly(config.StorePath(root))
	if err != nil {
		t.Fatal(err)
	}
	defer repoRO.Close()
	homeRO, err = store.OpenReadOnly(config.HomeStorePath())
	if err != nil {
		t.Fatal(err)
	}
	defer homeRO.Close()
	s = New(root, config.Default(), repoRO, homeRO, nil)
	hits, _ = callTool(t, s, "search", map[string]any{"query": "Frozen query"}).([]any)
	if len(hits) != 1 {
		t.Fatalf("frozen search: %#v", hits)
	}
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{
		"type": "decision", "scope": "repo", "title": "Should fail", "body": "no",
	}
	res, err := s.toolRemember(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("remember on a read-only checkout: %+v", res)
	}
}

type textEmbedder struct {
	got string
}

func (e *textEmbedder) Embed(_ context.Context, text string) ([]float32, error) {
	e.got = text
	return []float32{1, 0}, nil
}

func TestMCPNewCopiesQueryModel(t *testing.T) {
	cfg := config.Default()
	cfg.Ollama.EmbedModel = "qwen3-embedding:8b"
	repoDB, err := store.Open(filepath.Join(t.TempDir(), "repo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repoDB.Close() })
	s := New(t.TempDir(), cfg, repoDB, nil, nil)
	if s.Engine.QueryModel != "qwen3-embedding:8b" {
		t.Fatalf("QueryModel %q", s.Engine.QueryModel)
	}
}

func TestMCPCheckEmbedsQwenPrefix(t *testing.T) {
	cfg := config.Default()
	cfg.Ollama.EmbedModel = "qwen3-embedding:8b"
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
	emb := &textEmbedder{}
	s := New(t.TempDir(), cfg, repoDB, homeDB, emb)
	callTool(t, s, "check", map[string]any{"description": "hybrid rules"})
	if emb.got != retrieve.QueryText("qwen3-embedding:8b", "hybrid rules") {
		t.Fatalf("check embed %q", emb.got)
	}
}

func TestToolIndexWritesCheckoutMap(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	root := t.TempDir()
	writeGoFile(t, root, "widget.go", "package widget\n\nfunc UniqueWidget() {}\n")
	repo, err := store.Open(config.StorePath(root))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Close(); err != nil {
		t.Fatal(err)
	}
	homeDB, err := store.Open(config.HomeStorePath())
	if err != nil {
		t.Fatal(err)
	}
	if err := homeDB.Close(); err != nil {
		t.Fatal(err)
	}
	repoRO, err := store.OpenReadOnly(config.StorePath(root))
	if err != nil {
		t.Fatal(err)
	}
	defer repoRO.Close()
	homeRO, err := store.OpenReadOnly(config.HomeStorePath())
	if err != nil {
		t.Fatal(err)
	}
	defer homeRO.Close()

	extra := t.TempDir()
	extraPath := config.StorePath(extra)
	extraDB, err := store.Open(extraPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := extraDB.ReplaceFileMap(
		store.FileRecord{Path: "extra.go", ContentHash: "e", IndexedAt: time.Now().UTC()},
		[]store.Symbol{{Name: "ExtraWidget", Kind: "function_declaration", Line: 1, Exported: true}},
		nil,
	); err != nil {
		t.Fatal(err)
	}
	if err := extraDB.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(extraPath)
	if err != nil {
		t.Fatal(err)
	}
	extraRO, err := store.OpenReadOnly(extraPath)
	if err != nil {
		t.Fatal(err)
	}
	defer extraRO.Close()

	s := New(root, config.Default(), repoRO, homeRO, nil)
	s.Archive.Extras = []archive.Extra{{Root: extra, DB: extraRO}}

	got := callTool(t, s, "index", map[string]any{}).(map[string]any)
	assertJSONKeys(t, got, "files_indexed", "files_removed", "commits_new", "records")
	if got["files_indexed"] != float64(1) {
		t.Fatalf("index %#v", got)
	}

	fresh, err := store.OpenReadOnly(config.StorePath(root))
	if err != nil {
		t.Fatal(err)
	}
	defer fresh.Close()
	if !symbolNamed(t, fresh, "widget.go", "UniqueWidget") {
		t.Fatal("missing UniqueWidget")
	}
	after, err := os.ReadFile(extraPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("index changed the extra database")
	}

	writeGoFile(t, root, "sub/scoped.go", "package scoped\n\nfunc ScopedOnly() {}\n")
	scoped := callTool(t, s, "index", map[string]any{"scope": "sub"}).(map[string]any)
	if scoped["files_indexed"] != float64(1) {
		t.Fatalf("scoped index %#v", scoped)
	}
	if err := fresh.Close(); err != nil {
		t.Fatal(err)
	}
	fresh, err = store.OpenReadOnly(config.StorePath(root))
	if err != nil {
		t.Fatal(err)
	}
	defer fresh.Close()
	if !symbolNamed(t, fresh, "sub/scoped.go", "ScopedOnly") {
		t.Fatal("missing ScopedOnly")
	}
	if !symbolNamed(t, fresh, "widget.go", "UniqueWidget") {
		t.Fatal("scope index removed UniqueWidget")
	}
}

func TestToolEmbedDrainsQueue(t *testing.T) {
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
	if err := repoDB.UpsertRecord(&record.Record{
		ID: "rec_embedme", Slug: "embed-me", Type: record.TypeFeature, Scope: record.ScopeRepo,
		Title: "Embed me", Body: "queue this", Status: record.StatusAccepted,
		SourcePath: "docs/features/embed-me.md",
	}); err != nil {
		t.Fatal(err)
	}
	depth, err := repoDB.QueueDepth()
	if err != nil || depth != 1 {
		t.Fatalf("queue %d err=%v", depth, err)
	}
	emb := &textEmbedder{}
	s := New(t.TempDir(), config.Default(), repoDB, homeDB, emb)
	got := callTool(t, s, "embed", map[string]any{"once": true, "concurrency": 1}).(map[string]any)
	if got["embedded"] != float64(1) {
		t.Fatalf("embed %#v", got)
	}
	if emb.got == "" || !strings.Contains(emb.got, "Embed me") {
		t.Fatalf("embedded text %q", emb.got)
	}
	depth, err = repoDB.QueueDepth()
	if err != nil || depth != 0 {
		t.Fatalf("queue after %d err=%v", depth, err)
	}
}

func TestToolEmbedReportsOllamaDown(t *testing.T) {
	cfg := config.Default()
	cfg.Ollama.BaseURL = "http://127.0.0.1:1"
	cfg.Ollama.EmbedTimeout = "1s"
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
	s := New(t.TempDir(), cfg, repoDB, homeDB, nil)
	req := mcp.CallToolRequest{}
	req.Params.Arguments = map[string]any{"once": true}
	res, err := s.toolEmbed(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if res == nil || !res.IsError || !strings.Contains(toolErrorText(res), "ollama serve") {
		t.Fatalf("embed result %+v", res)
	}
}

func writeGoFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func symbolNamed(t *testing.T, st *store.Store, path, name string) bool {
	t.Helper()
	syms, err := st.SymbolsForFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, sym := range syms {
		if sym.Name == name {
			return true
		}
	}
	return false
}

func toolErrorText(res *mcp.CallToolResult) string {
	logged := mcpLogResult(res)
	row, _ := logged.(map[string]any)
	text, _ := row["text"].(string)
	return text
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
