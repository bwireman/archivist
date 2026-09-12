package record

import (
	"strings"
	"testing"
)

func TestParseAndSerialize(t *testing.T) {
	raw := `---
id: rec_test123
type: rule
scope: repo
status: accepted
title: Never call billing from handlers
severity: must-not
applies_to: ["internal/http/**"]
tags: [billing, http]
---

## Context
Why.

## Decision
Don't.
`
	r, err := ParseFile("docs/decisions/001-billing.md", raw)
	if err != nil {
		t.Fatal(err)
	}
	if r.Type != TypeRule {
		t.Fatalf("type: %s", r.Type)
	}
	if r.Severity != SeverityMustNot {
		t.Fatalf("severity: %s", r.Severity)
	}
	if len(r.AppliesTo) != 1 {
		t.Fatalf("applies_to: %v", r.AppliesTo)
	}
	out := Serialize(r)
	if !strings.Contains(out, "severity: must-not") {
		t.Fatalf("serialize missing severity: %s", out)
	}
}

func TestMatchesPaths(t *testing.T) {
	r := &Record{
		Type:      TypeRule,
		AppliesTo: []string{"internal/http/**"},
	}
	if !r.MatchesPaths([]string{"internal/http/server.go"}) {
		t.Fatal("expected match")
	}
	if r.MatchesPaths([]string{"pkg/foo.go"}) {
		t.Fatal("expected no match")
	}
}
