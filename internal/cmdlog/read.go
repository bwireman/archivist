package cmdlog

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Read loads a commands.log snapshot. A torn final line (no trailing newline
// and invalid JSON) is dropped. Any other invalid line is an error.
func Read(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	trailingNL := data[len(data)-1] == '\n'
	lines := strings.Split(string(data), "\n")
	var out []Entry
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			last := i == len(lines)-1 || (i == len(lines)-2 && lines[len(lines)-1] == "")
			if last && !trailingNL {
				continue
			}
			return nil, fmt.Errorf("commands.log line %d: %w", i+1, err)
		}
		out = append(out, e)
	}
	return out, nil
}
