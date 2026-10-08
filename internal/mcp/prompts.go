package mcp

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	mcpserver "github.com/mark3labs/mcp-go/server"

	"github.com/bwireman/archivist/internal/record"
)

// addPrompt is a prompts/get template for one record type. The prompt does not
// write; it tells the client to search, then call remember.
type addPrompt struct {
	name, title, description string
	kind                     record.Type
	lead, shape, after       string
	extra                    []string
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
		return nil, errors.New("topic is required")
	}
	scope := strings.TrimSpace(args["scope"])
	return mcp.NewGetPromptResult(p.title, []mcp.PromptMessage{
		mcp.NewPromptMessage(mcp.RoleUser, mcp.NewTextContent(p.text(topic, scope))),
	}), nil
}

func (p addPrompt) text(topic, scope string) string {
	procedure := "Before remember, search with no scope argument. Search covers repo, global, and dev. If a current record covers the topic, update it and keep its scope. If search shows a gap, pick one scope. dev: about the person or this machine, including a short-gap answer. repo: true only in this checkout. global: true for the product in every checkout."
	scopeLine := procedure
	if scope != "" {
		scopeLine = procedure + " Suggested scope: " + scope + ". Use it only when it matches that procedure."
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Add an Archivist %s about %q.\n\n%s\n\n", p.kind, topic, p.lead)
	fmt.Fprintf(&b, "1. Call search with type=%s and query=%q. Do not pass scope to search. If a current record covers this topic, call update with its id. Do not remember a second accepted record.\n", p.kind, topic)
	b.WriteString("2. Distill a short body. Leave out the chat transcript.\n")
	b.WriteString(p.shape)
	b.WriteString("\n3. ")
	b.WriteString(scopeLine)
	b.WriteString("\n")
	fmt.Fprintf(&b, "4. Call remember with type=%s, that scope, a title, and the body. remember writes SQLite only.\n", p.kind)
	if len(p.extra) > 0 {
		fmt.Fprintf(&b, " Also pass %s.\n", strings.Join(p.extra, ", "))
	}
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
		kind:        record.TypeDecision,
		lead:        "A decision is a choice among real alternatives. It is not a feature (how it works) and not a rule (must or must-not).",
		shape:       "Body sections: Context, Decision, Consequences. Omit options that were never in play and detail that lives only in code.\n",
	},
	{
		name:        "add-rule",
		title:       "Add a rule",
		description: "Walk through adding a rule: search first, then remember a must or should with applies_to globs.",
		kind:        record.TypeRule,
		lead:        "A rule is a must, must-not, should, or should-not that later work should follow.",
		shape: fmt.Sprintf("State the constraint. Set severity to %s, %s, %s, or %s. Set applies_to to path globs so check can match touched files.\n",
			record.SeverityMust, record.SeverityMustNot, record.SeverityShould, record.SeverityShouldNot),
		extra: []string{"severity", "applies_to"},
		after: "Call check with the paths the rule covers.",
	},
	{
		name:        "add-feature",
		title:       "Add a feature",
		description: "Walk through adding a feature: search first, then remember how a capability works.",
		kind:        record.TypeFeature,
		lead:        "A feature is how a capability works today. It is not why it was chosen and not a constraint.",
		shape:       "Body sections: Purpose, Behavior (including failure and empty cases), Connects to, Entry points. Set applies_to to the implementing packages.\n",
		extra:       []string{"applies_to"},
	},
	{
		name:        "add-guide",
		title:       "Add a guide",
		description: "Walk through adding a guide: search first, then remember a how-to procedure.",
		kind:        record.TypeGuide,
		lead:        "A guide is a how-to procedure.",
		shape:       "Body: when to use it, the steps, and what to skip.\n",
	},
	{
		name:        "add-map",
		title:       "Add a map",
		description: "Walk through adding a map record: search first, then remember how packages connect.",
		kind:        record.TypeMap,
		lead:        "A map record is a structural note about packages and how they connect. It is not the generated code map.",
		shape:       "Call the map tool if you need symbols, imports, or commits. Then write the packages, how they connect, and the entry points. Set applies_to to those packages.\n",
		extra:       []string{"applies_to"},
	},
	{
		name:        "add-pitfall",
		title:       "Add a pitfall",
		description: "Walk through adding a pitfall: search first, then remember a confirmed gotcha.",
		kind:        record.TypePitfall,
		lead:        "A pitfall is a confirmed gotcha, not a guess and not a one-off session error.",
		shape:       "Body: what goes wrong, when it happens, and what to do instead.\n",
	},
}
