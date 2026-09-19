package cmdlog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/bwireman/archivist/internal/config"
)

const maxFieldBytes = 64 * 1024

type Logger struct {
	enabled bool
	path    string
	mu      sync.Mutex
}

type Entry struct {
	TS         string `json:"ts"`
	Dir        string `json:"dir"`
	Source     string `json:"source"`
	Command    string `json:"command"`
	Args       any    `json:"args,omitempty"`
	Result     any    `json:"result,omitempty"`
	Error      string `json:"error,omitempty"`
	DurationMS int64  `json:"duration_ms,omitempty"`
}

func Disabled() *Logger {
	return &Logger{}
}

func New(path string) *Logger {
	if path == "" {
		return Disabled()
	}
	return &Logger{enabled: true, path: path}
}

func FromConfig(repoRoot string, cfg *config.Config) *Logger {
	if cfg == nil || !cfg.LogCommands {
		return Disabled()
	}
	return New(config.CommandsLogPath(repoRoot))
}

func (l *Logger) In(source, command string, args any) {
	l.append(Entry{
		Dir:     "in",
		Source:  source,
		Command: command,
		Args:    clip(args),
	})
}

func (l *Logger) Out(source, command string, result any, err error, started time.Time) {
	e := Entry{
		Dir:        "out",
		Source:     source,
		Command:    command,
		Result:     clip(result),
		DurationMS: time.Since(started).Milliseconds(),
	}
	if err != nil {
		e.Error = err.Error()
	}
	l.append(e)
}

func (l *Logger) append(e Entry) {
	if l == nil || !l.enabled {
		return
	}
	if e.TS == "" {
		e.TS = time.Now().UTC().Format(time.RFC3339Nano)
	}
	data, err := json.Marshal(e)
	if err != nil {
		return
	}
	data = append(data, '\n')
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(data)
}

func clip(v any) any {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil || len(b) <= maxFieldBytes {
		return v
	}
	return map[string]any{
		"truncated": true,
		"bytes":     len(b),
		"preview":   string(b[:maxFieldBytes]),
	}
}
