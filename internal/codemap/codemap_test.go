package codemap

import (
	"testing"

	"github.com/bwireman/archivist/internal/store"
)

func TestExtractGleamTopLevel(t *testing.T) {
	src := `import gleam/option.{None, Some}
import gxyz/function as gfunction

/// Audit outcome
pub type AuditResult {
  AuditResult(
    project_root: String,
  )
}

type Hidden = String

/// parse yaml
@external(javascript, "./go_over_ffi.mjs", "parse_adv")
pub fn parse_adv(content: String) -> List(String)

fn prefix_label(label: String) -> String {
  let nested = fn(x) { x }
  fn helper(x: String) -> String {
    x
  }
  label
}

pub const version = "1.0.0"

const hidden = 1
`
	res, err := Extract("src/go_over.gleam", src)
	if err != nil {
		t.Fatal(err)
	}

	wantEdges := []string{"gleam/option", "gxyz/function"}
	if len(res.Edges) != len(wantEdges) {
		t.Fatalf("imports: got %v", res.Edges)
	}
	for i, e := range wantEdges {
		if res.Edges[i].ToPath != e {
			t.Errorf("import %d: got %q want %q", i, res.Edges[i].ToPath, e)
		}
	}

	byKey := map[string]store.Symbol{}
	for _, s := range res.Symbols {
		byKey[s.Kind+":"+s.Name] = s
	}

	want := []string{
		"type:AuditResult",
		"constructor:AuditResult",
		"function:parse_adv",
		"function:prefix_label",
		"constant:version",
		"constant:hidden",
		"type:Hidden",
	}
	for _, key := range want {
		if _, ok := byKey[key]; !ok {
			t.Errorf("missing %s in %v", key, keys(byKey))
		}
	}
	if _, ok := byKey["function:helper"]; ok {
		t.Fatal("nested fn helper should not be mapped")
	}
	if !byKey["type:AuditResult"].Exported || !byKey["function:parse_adv"].Exported || !byKey["constant:version"].Exported {
		t.Fatalf("expected pub symbols exported")
	}
	if byKey["function:prefix_label"].Exported || byKey["constant:hidden"].Exported || byKey["type:Hidden"].Exported {
		t.Fatalf("expected private symbols unexported")
	}
	if byKey["function:parse_adv"].DocLine != "parse yaml" {
		t.Errorf("parse_adv doc %q", byKey["function:parse_adv"].DocLine)
	}
	if byKey["type:AuditResult"].DocLine != "Audit outcome" {
		t.Errorf("AuditResult doc %q", byKey["type:AuditResult"].DocLine)
	}
	if !byKey["constructor:AuditResult"].Exported {
		t.Error("constructor of pub type should be exported")
	}
}

func keys(m map[string]store.Symbol) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func TestExtractGleamConstructors(t *testing.T) {
	src := `pub type Severity {
  SeverityHigh
  SeverityUnknown(info: String)
}

fn uses(s: Severity) -> String {
  case s {
    SeverityHigh -> "high"
    SeverityUnknown(_) -> "unknown"
  }
}
`
	res, err := Extract("src/warning.gleam", src)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, s := range res.Symbols {
		if s.Kind == "constructor" {
			names = append(names, s.Name)
		}
	}
	if len(names) != 2 || names[0] != "SeverityHigh" || names[1] != "SeverityUnknown" {
		t.Fatalf("constructors %v", names)
	}
	for _, s := range res.Symbols {
		if s.Name == "uses" && s.Kind != "function" {
			t.Fatalf("uses: %+v", s)
		}
	}
}

func TestExtractGoNamesMethodsNotReturnTypes(t *testing.T) {
	src := `package store

import "database/sql"

type Store struct {
	db *sql.DB
}

// UpsertRecord writes a record.
func (s *Store) UpsertRecord(id string) error {
	return nil
}

func Open(path string) (*Store, error) {
	return nil, nil
}
`
	res, err := Extract("internal/store/store.go", src)
	if err != nil {
		t.Fatal(err)
	}
	if res.PackageName != "store" {
		t.Errorf("package %q", res.PackageName)
	}
	byName := map[string]string{}
	for _, sym := range res.Symbols {
		byName[sym.Name] = sym.Kind
	}
	if byName["UpsertRecord"] != "method_declaration" {
		t.Errorf("method should be named for itself, not its return type: %+v", res.Symbols)
	}
	if byName["Open"] != "function_declaration" {
		t.Errorf("missing Open: %+v", res.Symbols)
	}
	if _, ok := byName["error"]; ok {
		t.Errorf("return type leaked in as a symbol: %+v", res.Symbols)
	}
	if len(res.Edges) != 1 || res.Edges[0].ToPath != "database/sql" {
		t.Errorf("edges %+v", res.Edges)
	}
}

func TestExtractMjsUsesJavaScript(t *testing.T) {
	src := `export function parse_adv(content) {
  return content
}
`
	res, err := Extract("src/go_over_ffi.mjs", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Symbols) != 1 {
		t.Fatalf("symbols %+v", res.Symbols)
	}
	if res.Symbols[0].Name != "parse_adv" {
		t.Errorf("name %q", res.Symbols[0].Name)
	}
	if res.Symbols[0].Kind != "function_declaration" {
		t.Errorf("kind %q", res.Symbols[0].Kind)
	}
}

func TestExtractGenericUnknownLanguage(t *testing.T) {
	src := "export function leftover() {\n}\n"
	res, err := Extract("vendor/script.unknown", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Symbols) != 1 {
		t.Fatalf("generic fallback %+v", res.Symbols)
	}
	if res.Symbols[0].Name != "leftover" || res.Symbols[0].Kind != "function" {
		t.Fatalf("generic fallback %+v", res.Symbols)
	}
	if !res.Symbols[0].Exported {
		t.Fatal("export should be marked exported")
	}
}

func TestExtractGenericUnsupportedLanguages(t *testing.T) {
	cases := []struct {
		path     string
		src      string
		contains []string
	}{
		{
			path:     "lib.ex",
			src:      "defmodule Foo do\n  def bar, do: :ok\n  defp hidden, do: :ok\nend\n",
			contains: []string{"Foo", "bar", "hidden"},
		},
		{
			path:     "lib.rb",
			src:      "class Foo\n  def bar\n  end\nend\n",
			contains: []string{"Foo", "bar"},
		},
		{
			path:     "lib.php",
			src:      "<?php\nfunction foo() {}\nclass Bar {}\n",
			contains: []string{"foo", "Bar"},
		},
		{
			path:     "lib.kt",
			src:      "fun foo() {}\nclass Bar {}\n",
			contains: []string{"foo", "Bar"},
		},
		{
			path:     "lib.swift",
			src:      "func foo() {}\nstruct Bar {}\nprotocol Baz {}\n",
			contains: []string{"foo", "Bar", "Baz"},
		},
		{
			path:     "lib.zig",
			src:      "pub fn foo() void {}\n",
			contains: []string{"foo"},
		},
		{
			path:     "lib.lua",
			src:      "function foo()\nend\n",
			contains: []string{"foo"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			res, err := Extract(tc.path, tc.src)
			if err != nil {
				t.Fatal(err)
			}
			got := map[string]bool{}
			for _, s := range res.Symbols {
				got[s.Name] = true
			}
			for _, name := range tc.contains {
				if !got[name] {
					t.Errorf("missing %s in %v", name, symbolNames(res))
				}
			}
		})
	}
}

func TestExtractGenericSkipsDocsAndData(t *testing.T) {
	src := "# Title\n\nThis class of problem is a type of issue.\n\n```\nfunc Hello() {}\n```\n"
	for _, path := range []string{"README.md", "notes.txt", "data.json", "config.yaml", "go.mod"} {
		res, err := Extract(path, src)
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Symbols) != 0 || len(res.Edges) != 0 {
			t.Errorf("%s: want empty map, got %+v", path, res)
		}
	}
}

func TestExtractGenericBackupWhenTreeSitterFindsNothing(t *testing.T) {
	src := "func Hello() {}\n"
	res, err := Extract("weird.py", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Symbols) != 1 || res.Symbols[0].Name != "Hello" {
		t.Fatalf("backup %+v", res.Symbols)
	}
}

func TestExtractGenericSkipsEnglishStopwords(t *testing.T) {
	src := "type of thing\nclass of problem\n"
	res, err := Extract("notes.ex", src)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Symbols) != 0 {
		t.Fatalf("stopwords %+v", res.Symbols)
	}
}
