package cmdlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bwireman/archivist/internal/config"
)

func TestDisabledWritesNothing(t *testing.T) {
	log := Disabled()
	log.In("cli", "archivist search", map[string]any{"args": []string{"foo"}})
	log.Out("cli", "archivist search", nil, nil, time.Now())
	if log.Enabled() {
		t.Fatal("disabled logger should not be enabled")
	}
}

func TestFromConfigOff(t *testing.T) {
	dir := t.TempDir()
	log := FromConfig(dir, config.Default())
	log.In("mcp", "search", map[string]any{"query": "x"})
	if _, err := os.Stat(config.CommandsLogPath(dir)); !os.IsNotExist(err) {
		t.Fatal("expected no log file when log_commands is false")
	}
}

func TestInOutJSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "commands.log")
	log := New(path)
	start := time.Now()
	log.In("mcp", "search", map[string]any{"query": "sqlite"})
	log.Out("mcp", "search", []any{map[string]any{"id": "rec_1"}}, nil, start)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines: %d\n%s", len(lines), data)
	}
	var in, out Entry
	if err := json.Unmarshal([]byte(lines[0]), &in); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &out); err != nil {
		t.Fatal(err)
	}
	if in.Dir != "in" || in.Source != "mcp" || in.Command != "search" {
		t.Fatalf("in: %+v", in)
	}
	if out.Dir != "out" || out.Error != "" || out.Result == nil {
		t.Fatalf("out: %+v", out)
	}
}

func TestOutRecordsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "commands.log")
	log := New(path)
	log.Out("cli", "archivist remember", nil, os.ErrPermission, time.Now())
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var e Entry
	if err := json.Unmarshal(bytesTrimLine(data), &e); err != nil {
		t.Fatal(err)
	}
	if e.Error == "" {
		t.Fatal("expected error field")
	}
}

func TestClipLargePayload(t *testing.T) {
	big := strings.Repeat("x", maxFieldBytes+10)
	got := clip(big)
	m, ok := got.(map[string]any)
	if !ok || m["truncated"] != true {
		t.Fatalf("got %#v", got)
	}
}

func bytesTrimLine(data []byte) []byte {
	return []byte(strings.TrimSpace(string(data)))
}
