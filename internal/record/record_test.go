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

func TestValidateFeatureType(t *testing.T) {
	r := &Record{Type: TypeFeature, Scope: ScopeGlobal, Title: "Embed queue", Slug: "embed-queue"}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	r.Type = "unknown"
	if err := r.Validate(); err == nil {
		t.Fatal("expected invalid type")
	}
}

func TestIndexOrderCoversKnownTypes(t *testing.T) {
	known := []Type{TypeDecision, TypeRule, TypeFeature, TypeGuide, TypeMap, TypePitfall}
	seen := map[Type]bool{}
	for _, typ := range IndexOrder {
		if !ValidType(typ) {
			t.Fatalf("IndexOrder has invalid type %s", typ)
		}
		seen[typ] = true
	}
	for _, typ := range known {
		if !seen[typ] {
			t.Fatalf("IndexOrder missing %s", typ)
		}
	}
}

func TestInferScopeFromPath(t *testing.T) {
	if InferScopeFromPath("user/note.md") != ScopeDev {
		t.Fatal("user")
	}
	if InferScopeFromPath("global/note.md") != ScopeGlobal {
		t.Fatal("home global")
	}
	if InferScopeFromPath("docs/global-decisions/note.md") != ScopeGlobal {
		t.Fatal("in-repo global")
	}
	if InferScopeFromPath("docs/decisions/note.md") != ScopeRepo {
		t.Fatal("repo")
	}
}
