package cmd

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
		fields := strings.Fields(out)
		if len(fields) < 3 || fields[2] != version.Version {
			t.Fatalf("%v version field = %q in %q", args, fields, out)
		}
		if rev := version.Revision(); rev != "" && !strings.Contains(out, rev) {
			t.Fatalf("%v missing commit %q in %q", args, rev, out)
		}
		if built := version.Built(); built != "" && !strings.Contains(out, built) {
			t.Fatalf("%v missing build time %q in %q", args, built, out)
		}
	}
}

func TestBuiltBinaryPrintsHashAndBuildTime(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "archivist")
	build := exec.Command("go", "build", "-o", bin, "github.com/bwireman/archivist/cmd/archivist")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	cmd := exec.Command(bin, "version")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("version: %v\n%s", err, stderr.String())
	}
	fields := strings.Fields(stdout.String())
	if len(fields) != 5 || fields[0] != "archivist" || fields[1] != "version" || fields[2] != version.Version {
		t.Fatalf("stdout %q", stdout.String())
	}
	if len(fields[3]) != 40 {
		t.Fatalf("commit %q", fields[3])
	}
	if _, err := time.Parse(time.RFC3339, fields[4]); err != nil {
		t.Fatalf("build time %q: %v", fields[4], err)
	}
}
