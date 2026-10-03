package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/skills"
	"github.com/bwireman/archivist/internal/version"
)

func TestInvocationWarnsWhenRulesAreStale(t *testing.T) {
	root := t.TempDir()
	if err := skills.Install(root, skills.TargetCursor); err != nil {
		t.Fatal(err)
	}
	rule := filepath.Join(root, ".cursor", "rules", "archivist-consult.mdc")
	if err := os.WriteFile(rule, []byte("edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"--path", root, "version"},
		{"--path", root, "--version"},
		{"--path", root, "--help"},
	} {
		c := NewRoot()
		var out, errb bytes.Buffer
		c.SetOut(&out)
		c.SetErr(&errb)
		c.SetArgs(args)
		if err := c.Execute(); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		if args[len(args)-1] != "--help" && !strings.Contains(out.String(), version.Version) {
			t.Fatalf("%v stdout %q", args, out.String())
		}
		if !strings.Contains(errb.String(), "archivist skills install --target cursor") {
			t.Fatalf("%v stderr %q", args, errb.String())
		}
	}
}

func TestNoArgsWarnsOnce(t *testing.T) {
	root := t.TempDir()
	if err := skills.Install(root, skills.TargetCursor); err != nil {
		t.Fatal(err)
	}
	rule := filepath.Join(root, ".cursor", "rules", "archivist-consult.mdc")
	if err := os.WriteFile(rule, []byte("edited\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := NewRoot()
	var out, errb bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&errb)
	c.SetArgs([]string{"--path", root})
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Count(errb.String(), "out of date") != 1 {
		t.Fatalf("stderr %q", errb.String())
	}
}

func TestInvocationQuietWhenRulesMatch(t *testing.T) {
	root := t.TempDir()
	if err := skills.Install(root, skills.TargetCursor); err != nil {
		t.Fatal(err)
	}
	c := NewRoot()
	var out, errb bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&errb)
	c.SetArgs([]string{"--path", root, "version"})
	if err := c.Execute(); err != nil {
		t.Fatal(err)
	}
	if errb.Len() != 0 {
		t.Fatalf("stderr %q", errb.String())
	}
}
