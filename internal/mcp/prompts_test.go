package mcp

import (
	"context"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

func TestAddPrompts(t *testing.T) {
	srv := (&Server{}).MCPServer()
	listed := srv.ListPrompts()
	if len(listed) != len(addPrompts) {
		t.Fatalf("prompt count %d, want %d", len(listed), len(addPrompts))
	}
	for _, spec := range addPrompts {
		got := listed[spec.name]
		if got == nil {
			t.Fatalf("missing prompt %s", spec.name)
		}
		if got.Prompt.Title != spec.title || got.Prompt.Description != spec.description {
			t.Fatalf("%s metadata: %+v", spec.name, got.Prompt)
		}
		var topicRequired, scopeOptional bool
		for _, arg := range got.Prompt.Arguments {
			switch arg.Name {
			case "topic":
				topicRequired = arg.Required
			case "scope":
				scopeOptional = !arg.Required
			}
		}
		if !topicRequired || !scopeOptional {
			t.Fatalf("%s arguments: %+v", spec.name, got.Prompt.Arguments)
		}

		req := mcp.GetPromptRequest{}
		req.Params.Arguments = map[string]string{"topic": "billing calls", "scope": "repo"}
		res, err := got.Handler(context.Background(), req)
		if err != nil {
			t.Fatal(err)
		}
		text := promptText(t, res)
		for _, needle := range []string{
			`about "billing calls"`,
			"type=" + string(spec.kind),
			"search",
			"update",
			"remember",
			"Suggested scope: repo",
			"archive names a listed extra root",
			"Do not pass scope to search.",
			"embed --once",
		} {
			if !strings.Contains(text, needle) {
				t.Fatalf("%s prompt missing %q:\n%s", spec.name, needle, text)
			}
		}

		req.Params.Arguments = map[string]string{}
		if _, err := got.Handler(context.Background(), req); err == nil {
			t.Fatalf("%s accepted an empty topic", spec.name)
		}
	}
}

func promptText(t *testing.T, res *mcp.GetPromptResult) string {
	t.Helper()
	if res == nil || len(res.Messages) != 1 {
		t.Fatalf("want one message, got %+v", res)
	}
	text, ok := res.Messages[0].Content.(mcp.TextContent)
	if !ok {
		t.Fatalf("content type %T", res.Messages[0].Content)
	}
	return text.Text
}
