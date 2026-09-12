package codemap

import (
	"path/filepath"
	"testing"
)

const jsSample = `import { x } from "lib";

export function parseAdv(content) {
  return content;
}

class Widget {}
`

const tsSample = `import { x } from "lib";

export function parseAdv(content: string): string {
  return content;
}

class Widget {}
`

func TestExtractEveryRegisteredLanguage(t *testing.T) {
	cases := []struct {
		path     string
		src      string
		contains []string
		pkg      string
	}{
		{
			path: "demo.go",
			src: `package demo

import "fmt"

func Hello() {}

type Box struct{}
`,
			contains: []string{"Hello", "Box"},
			pkg:      "demo",
		},
		{
			path: "demo.py",
			src: `import os

def greet(name):
    pass

class Thing:
    def method(self):
        pass
`,
			contains: []string{"greet", "Thing"},
		},
		{path: "demo.js", src: jsSample, contains: []string{"parseAdv", "Widget"}},
		{path: "demo.jsx", src: jsSample, contains: []string{"parseAdv", "Widget"}},
		{path: "demo.mjs", src: jsSample, contains: []string{"parseAdv", "Widget"}},
		{path: "demo.cjs", src: jsSample, contains: []string{"parseAdv", "Widget"}},
		{path: "demo.ts", src: tsSample, contains: []string{"parseAdv", "Widget"}},
		{path: "demo.tsx", src: tsSample, contains: []string{"parseAdv", "Widget"}},
		{path: "demo.mts", src: tsSample, contains: []string{"parseAdv", "Widget"}},
		{path: "demo.cts", src: tsSample, contains: []string{"parseAdv", "Widget"}},
		{
			path: "demo.rs",
			src: `use std::io;

pub fn hello() {}
pub struct Foo {}
pub enum Bar { A }
pub trait T {}
impl Foo {}
`,
			contains: []string{"hello", "Foo", "Bar", "T"},
		},
		{
			path: "Demo.java",
			src: `package demo;

import java.util.List;

public class Foo {
  public void bar() {}
}

public interface Baz {}
`,
			contains: []string{"Foo", "bar", "Baz"},
		},
		{
			path: "demo.gleam",
			src: `import gleam/io

pub fn main() {
  Nil
}

pub type Box {
  Box
}
`,
			contains: []string{"main", "Box"},
		},
	}

	seen := map[string]bool{}
	for _, tc := range cases {
		ext := filepath.Ext(tc.path)
		seen[ext] = true
		t.Run(ext, func(t *testing.T) {
			res, err := Extract(tc.path, tc.src)
			if err != nil {
				t.Fatal(err)
			}
			got := map[string]bool{}
			for _, s := range res.Symbols {
				got[s.Name] = true
			}
			if len(res.Symbols) == 0 {
				t.Fatalf("no symbols")
			}
			for _, name := range tc.contains {
				if !got[name] {
					t.Errorf("missing %s in %v", name, symbolNames(res))
				}
			}
			if tc.pkg != "" && res.PackageName != tc.pkg {
				t.Errorf("package %q want %q", res.PackageName, tc.pkg)
			}
		})
	}

	required := map[string]bool{".gleam": true}
	for ext := range languageSpecs {
		required[ext] = true
	}
	for ext := range required {
		if !seen[ext] {
			t.Errorf("no extract fixture for registered language %s", ext)
		}
	}
	for ext := range seen {
		if !required[ext] {
			t.Errorf("fixture for unregistered language %s", ext)
		}
	}
}

func symbolNames(res Result) []string {
	out := make([]string, 0, len(res.Symbols))
	for _, s := range res.Symbols {
		out = append(out, s.Name)
	}
	return out
}
