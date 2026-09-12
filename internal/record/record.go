package record

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/bwireman/archivist/internal/config"
	"github.com/bwireman/archivist/internal/glob"
)

type Type string

const (
	TypeDecision Type = "decision"
	TypeRule     Type = "rule"
	TypeGuide    Type = "guide"
	TypeMap      Type = "map"
	TypePitfall  Type = "pitfall"
)

type Scope string

const (
	ScopeDev    Scope = "dev"
	ScopeRepo   Scope = "repo"
	ScopeGlobal Scope = "global"
)

type Status string

const (
	StatusProposed    Status = "proposed"
	StatusAccepted    Status = "accepted"
	StatusDeprecated  Status = "deprecated"
	StatusSuperseded  Status = "superseded"
)

type Severity string

const (
	SeverityMust       Severity = "must"
	SeverityMustNot    Severity = "must-not"
	SeverityShould     Severity = "should"
	SeverityShouldNot  Severity = "should-not"
)

// Record is the unit of knowledge in the archive.
type Record struct {
	ID                string
	Slug              string
	Type              Type
	Scope             Scope
	Title             string
	Status            Status
	Severity          Severity
	Body              string
	SourcePath        string
	Tags              []string
	AppliesTo         []string
	Supersedes        []string
	SupersededBy      string
	ProvenanceCommit  string
	ContentHash       string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ScopePrecedence returns higher for more specific scopes.
func ScopePrecedence(s Scope) int {
	switch s {
	case ScopeRepo:
		return 3
	case ScopeGlobal:
		return 2
	case ScopeDev:
		return 1
	default:
		return 0
	}
}

// InferScopeFromPath derives scope from a record file path relative to repo or home.
func InferScopeFromPath(path string) Scope {
	path = filepath.ToSlash(path)
	if strings.HasPrefix(path, config.UserGlobalPrefix+"/") {
		return ScopeDev
	}
	if glob.MatchAnyPattern(path, []string{config.DefaultGlobalDecisionsDir + "/**"}) {
		return ScopeGlobal
	}
	if glob.MatchAnyPattern(path, []string{config.DefaultDecisionsDir + "/**"}) {
		return ScopeRepo
	}
	return ScopeRepo
}

// SlugFromPath returns the filename without extension as slug.
func SlugFromPath(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	if ext != "" {
		base = strings.TrimSuffix(base, ext)
	}
	return base
}

// NewID generates a stable-looking record id.
func NewID() string {
	var b [10]byte
	_, _ = rand.Read(b[:])
	return "rec_" + hex.EncodeToString(b[:])
}

// ContentHash computes a hash of front matter fields plus body.
func ContentHash(r *Record) string {
	h := sha256.New()
	h.Write([]byte(r.Type))
	h.Write([]byte(r.Scope))
	h.Write([]byte(r.Title))
	h.Write([]byte(r.Status))
	h.Write([]byte(r.Severity))
	h.Write([]byte(r.Body))
	for _, t := range r.Tags {
		h.Write([]byte(t))
	}
	for _, a := range r.AppliesTo {
		h.Write([]byte(a))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (r *Record) Validate() error {
	if r.Type == "" {
		return fmt.Errorf("record type is required")
	}
	switch r.Type {
	case TypeDecision, TypeRule, TypeGuide, TypeMap, TypePitfall:
	default:
		return fmt.Errorf("invalid record type %q", r.Type)
	}
	if r.Scope == "" {
		return fmt.Errorf("record scope is required")
	}
	switch r.Scope {
	case ScopeDev, ScopeRepo, ScopeGlobal:
	default:
		return fmt.Errorf("invalid scope %q", r.Scope)
	}
	if r.Status == "" {
		r.Status = StatusAccepted
	}
	switch r.Status {
	case StatusProposed, StatusAccepted, StatusDeprecated, StatusSuperseded:
	default:
		return fmt.Errorf("invalid status %q", r.Status)
	}
	if r.Type == TypeRule && r.Severity != "" {
		switch r.Severity {
		case SeverityMust, SeverityMustNot, SeverityShould, SeverityShouldNot:
		default:
			return fmt.Errorf("invalid severity %q", r.Severity)
		}
	}
	if r.Title == "" {
		return fmt.Errorf("record title is required")
	}
	if r.Slug == "" {
		r.Slug = SlugFromPath(r.SourcePath)
	}
	if r.Slug == "" || r.Slug == "." {
		return fmt.Errorf("record slug is required")
	}
	return nil
}

// EmbedText returns the text used for embedding and FTS.
func (r *Record) EmbedText() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Title: %s\n", r.Title)
	fmt.Fprintf(&b, "Type: %s\n", r.Type)
	fmt.Fprintf(&b, "Scope: %s\n", r.Scope)
	if r.Severity != "" {
		fmt.Fprintf(&b, "Severity: %s\n", r.Severity)
	}
	if len(r.Tags) > 0 {
		fmt.Fprintf(&b, "Tags: %s\n", strings.Join(r.Tags, ", "))
	}
	if len(r.AppliesTo) > 0 {
		fmt.Fprintf(&b, "AppliesTo: %s\n", strings.Join(r.AppliesTo, ", "))
	}
	b.WriteByte('\n')
	b.WriteString(strings.TrimSpace(r.Body))
	return b.String()
}

// MatchesPaths reports whether any applies_to glob matches any of the paths.
func (r *Record) MatchesPaths(paths []string) bool {
	if len(r.AppliesTo) == 0 {
		return false
	}
	for _, p := range paths {
		if glob.MatchAnyPattern(p, r.AppliesTo) {
			return true
		}
	}
	return false
}

// IsEnforceable returns true for must/must-not rules.
func (r *Record) IsEnforceable() bool {
	return r.Type == TypeRule && (r.Severity == SeverityMust || r.Severity == SeverityMustNot)
}
