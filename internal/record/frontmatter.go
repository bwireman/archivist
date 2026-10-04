package record

import (
	"errors"
	"fmt"
	"strings"
)

// HasFrontMatterID reports whether content has typed-record front matter with an id field.
func HasFrontMatterID(content string) bool {
	fm, _, err := splitFrontMatter(content)
	if err != nil || fm == "" {
		return false
	}
	return strings.TrimSpace(parseYAMLMap(fm)["id"]) != ""
}

// ParseFile parses a markdown file with YAML front matter into a Record.
// fallbackScope applies when neither the path prefix nor the front matter
// names a scope; an empty fallback means repo.
func ParseFile(sourcePath, content string, fallbackScope Scope) (*Record, error) {
	fm, body, err := splitFrontMatter(content)
	if err != nil {
		return nil, err
	}
	fields := parseYAMLMap(fm)

	scope := InferScopeFromPath(sourcePath)
	if scope == "" {
		scope = fallbackScope
	}
	if scope == "" {
		scope = ScopeRepo
	}
	r := &Record{
		SourcePath: sourcePath,
		Slug:       SlugFromPath(sourcePath),
		Body:       strings.TrimSpace(body),
		Scope:      scope,
	}

	if v, ok := fields["id"]; ok {
		r.ID = v
	}
	if v, ok := fields["type"]; ok {
		r.Type = Type(v)
	}
	if v, ok := fields["scope"]; ok {
		r.Scope = Scope(v)
	}
	if v, ok := fields["status"]; ok {
		r.Status = Status(v)
	}
	if v, ok := fields["title"]; ok {
		r.Title = v
	}
	if v, ok := fields["severity"]; ok {
		r.Severity = Severity(v)
	}
	if v, ok := fields["superseded_by"]; ok {
		r.SupersededBy = v
	}
	if v, ok := fields["provenance_commit"]; ok {
		r.ProvenanceCommit = v
	}
	if v, ok := fields["tags"]; ok {
		r.Tags = parseInlineList(v)
	}
	if v, ok := fields["applies_to"]; ok {
		r.AppliesTo = parseInlineList(v)
	}
	if v, ok := fields["supersedes"]; ok {
		r.Supersedes = parseInlineList(v)
	}

	if r.ID == "" {
		r.ID = NewID()
	}
	if r.Type == "" {
		r.Type = TypeDecision
	}
	if r.Title == "" {
		r.Title = TitleFromBody(r.Body)
	}
	r.ContentHash = ContentHash(r)
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return r, nil
}

// Serialize writes a record back to markdown with front matter.
func Serialize(r *Record) string {
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "id: %s\n", r.ID)
	fmt.Fprintf(&b, "type: %s\n", r.Type)
	fmt.Fprintf(&b, "scope: %s\n", r.Scope)
	fmt.Fprintf(&b, "status: %s\n", r.Status)
	fmt.Fprintf(&b, "title: %s\n", yamlQuote(r.Title))
	if r.Severity != "" {
		fmt.Fprintf(&b, "severity: %s\n", r.Severity)
	}
	if len(r.AppliesTo) > 0 {
		fmt.Fprintf(&b, "applies_to: %s\n", formatInlineList(r.AppliesTo))
	}
	if len(r.Tags) > 0 {
		fmt.Fprintf(&b, "tags: %s\n", formatInlineList(r.Tags))
	}
	if len(r.Supersedes) > 0 {
		fmt.Fprintf(&b, "supersedes: %s\n", formatInlineList(r.Supersedes))
	}
	if r.SupersededBy != "" {
		fmt.Fprintf(&b, "superseded_by: %s\n", r.SupersededBy)
	}
	if r.ProvenanceCommit != "" {
		fmt.Fprintf(&b, "provenance_commit: %s\n", r.ProvenanceCommit)
	}
	b.WriteString("---\n\n")
	b.WriteString(strings.TrimSpace(r.Body))
	if !strings.HasSuffix(r.Body, "\n") {
		b.WriteByte('\n')
	}
	return b.String()
}

func splitFrontMatter(content string) (string, string, error) {
	content = strings.TrimPrefix(content, "\ufeff")
	if !strings.HasPrefix(content, "---") {
		return "", content, nil
	}
	rest := content[3:]
	if strings.HasPrefix(rest, "\n") {
		rest = rest[1:]
	} else if strings.HasPrefix(rest, "\r\n") {
		rest = rest[2:]
	} else {
		return "", content, nil
	}
	fm, body, ok := strings.Cut(rest, "\n---")
	if !ok {
		return "", content, errors.New("unclosed front matter")
	}
	body = strings.TrimPrefix(body, "\n")
	return fm, body, nil
}

func parseYAMLMap(fm string) map[string]string {
	out := map[string]string{}
	for line := range strings.SplitSeq(fm, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := splitYAMLLine(line)
		if !ok {
			continue
		}
		out[key] = unquoteYAML(val)
	}
	return out
}

func splitYAMLLine(line string) (string, string, bool) {
	key, val, ok := strings.Cut(line, ":")
	if !ok {
		return "", "", false
	}
	key = strings.TrimSpace(key)
	val = strings.TrimSpace(val)
	return key, val, key != ""
}

func unquoteYAML(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

func yamlQuote(s string) string {
	if strings.ContainsAny(s, ":#[]{}\"'\n") {
		return fmt.Sprintf("%q", s)
	}
	return s
}

func parseInlineList(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" || s == "[]" {
		return nil
	}
	if !strings.HasPrefix(s, "[") {
		return []string{s}
	}
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(strings.TrimSpace(s), "]")
	if s == "" {
		return nil
	}
	var out []string
	for part := range strings.SplitSeq(s, ",") {
		part = strings.TrimSpace(part)
		part = unquoteYAML(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func formatInlineList(items []string) string {
	if len(items) == 0 {
		return "[]"
	}
	quoted := make([]string, len(items))
	for i, item := range items {
		quoted[i] = yamlQuote(item)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func TitleFromBody(body string) string {
	for line := range strings.SplitSeq(body, "\n") {
		line = strings.TrimSpace(line)
		if title, ok := strings.CutPrefix(line, "# "); ok {
			return strings.TrimSpace(title)
		}
	}
	first := strings.TrimSpace(body)
	if len(first) > 80 {
		return first[:80] + "..."
	}
	if first != "" {
		return first
	}
	return "Untitled"
}
