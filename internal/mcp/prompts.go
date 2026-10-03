package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"
)

// addPrompt is a prompts/get template for one record type. The prompt does not
// write; it tells the client to search, then call remember.
type addPrompt struct {
	name, title, description, kind string
	lead, shape, fields, after     string
}

func registerAddPrompts(srv *mcpserver.MCPServer) {
	for _, spec := range addPrompts {
		srv.AddPrompt(mcp.NewPrompt(spec.name,
			mcp.WithPromptTitle(spec.title),
			mcp.WithPromptDescription(spec.description),
			mcp.WithArgument("topic",
				mcp.RequiredArgument(),
				mcp.ArgumentDescription("What to record. Search this before writing."),
			),
			mcp.WithArgument("scope",
				mcp.ArgumentDescription("repo, global, or dev. Omit to choose from the prompt."),
			),
		), spec.handle)
	}
}

func (p addPrompt) handle(_ context.Context, req mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	args := req.Params.Arguments
	topic := strings.TrimSpace(args["topic"])
	if topic == "" {
		return nil, fmt.Errorf("topic is required")
	}
	scope := strings.TrimSpace(args["scope"])
	return mcp.NewGetPromptResult(p.title, []mcp.PromptMessage{
		mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(p.text(topic, scope))),
	}), nil
}

func (p addPrompt) text(topic, scope string) string {
	scopeLine := "Choose scope: repo for this checkout, global for the product, dev for a personal note."
	if scope != "" {
		scopeLine = "Use scope " + scope + "."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Add an Archivist %s about %q.\n\n%s\n\n", p.kind, topic, p.lead)
	fmt.Fprintf(&b, "1. Call search with type=%s and query=%q. If a current record covers this topic, call update with its id. Do not remember a second accepted record.\n", p.kind, topic)
	b.WriteString("2. Distill a short body. Leave out the chat transcript.\n")
	b.WriteString(p.shape)
	b.WriteString("\n3. ")
	b.WriteString(scopeLine)
	b.WriteString("\n")
	fmt.Fprintf(&b, "4. Call remember with type=%s, that scope, a title, and the body%s. remember writes SQLite only.\n", p.kind, p.fields)
	step := 5
	if p.after != "" {
		fmt.Fprintf(&b, "%d. %s\n", step, p.after)
		step++
	}
	fmt.Fprintf(&b, "%d. Run archivist embed --once if the body changed. Skip embed when Ollama is down. archivist export writes only when records.write_docs is true.\n", step)
	return b.String()
}

var addPrompts = []addPrompt{
	{
		name:        "add-decision",
		title:       "Add a decision",
		description: "Walk through adding a decision: search first, then remember a choice among alternatives.",
		kind:        "decision",
		lead:        "A decision is a choice among real alternatives. It is not a feature (how it works) and not a rule (must or must-not).",
		shape:       "Body sections: Context, Decision, Consequences. Omit options that were never in play and detail that lives only in code.\n",
	},
	{
		name:        "add-rule",
		title:       "Add a rule",
		description: "Walk through adding a rule: search first, then remember a must or should with applies_to globs.",
		kind:        "rule",
		lead:        "A rule is a must, must-not, should, or should-not that later work should follow.",
		shape:       "State the constraint. Set severity to must, must-not, should, or should-not. Set applies_to to path globs so check can match touched files.\n",
		fields:      ", severity, and applies_to",
		after:       "Call check with the paths the rule covers.",
	},
	{
		name:        "add-feature",
		title:       "Add a feature",
		description: "Walk through adding a feature: search first, then remember how a capability works.",
		kind:        "feature",
		lead:        "A feature is how a capability works today. It is not why it was chosen and not a constraint.",
		shape:       "Body sections: Purpose, Behavior (including failure and empty cases), Connects to, Entry points. Set applies_to to the implementing packages.\n",
		fields:      " and applies_to",
	},
	{
		name:        "add-guide",
		title:       "Add a guide",
		description: "Walk through adding a guide: search first, then remember a how-to procedure.",
		kind:        "guide",
		lead:        "A guide is a how-to procedure.",
		shape:       "Body: when to use it, the steps, and what to skip.\n",
	},
	{
		name:        "add-map",
		title:       "Add a map",
		description: "Walk through adding a map record: search first, then remember how packages connect.",
		kind:        "map",
		lead:        "A map record is a structural note about packages and how they connect. It is not the generated code map.",
		shape:       "Call the map tool if you need symbols, imports, or commits. Then write the packages, how they connect, and the entry points. Set applies_to to those packages.\n",
		fields:      " and applies_to",
	},
	{
		name:        "add-pitfall",
		title:       "Add a pitfall",
		description: "Walk through adding a pitfall: search first, then remember a confirmed gotcha.",
		kind:        "pitfall",
		lead:        "A pitfall is a confirmed gotcha, not a guess and not a one-off session error.",
		shape:       "Body: what goes wrong, when it happens, and what to do instead.\n",
	},
}
