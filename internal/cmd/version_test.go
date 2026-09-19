package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bwireman/archivist/internal/version"
)

func TestVersionFlagAndCommand(t *testing.T) {
	for _, args := range [][]string{{"--version"}, {"version"}} {
		root := NewRoot()
		var buf bytes.Buffer
		root.SetOut(&buf)
		root.SetErr(&buf)
		root.SetArgs(args)
		if err := root.Execute(); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
		out := buf.String()
		if !strings.Contains(out, version.Version) {
			t.Fatalf("%v missing version in %q", args, out)
		}
	}
}
