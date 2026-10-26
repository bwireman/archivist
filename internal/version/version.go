// Package version reports the archivist build version.
package version

import (
	"runtime/debug"
	"strings"
)

// Version is the CLI / product version. Override at build time with
// -ldflags "-X github.com/bwireman/archivist/internal/version.Version=..."
var Version = "0.2.0"

// Commit is the source revision. Override at build time with
// -ldflags "-X github.com/bwireman/archivist/internal/version.Commit=..."
// When empty, Revision falls back to the Go VCS build stamp.
var Commit = ""

// Date is when this binary was built, as UTC RFC3339. Override at build time with
// -ldflags "-X github.com/bwireman/archivist/internal/version.Date=..."
// When empty, Built falls back to vcs.time (the commit timestamp).
var Date = ""

func String() string {
	return Version
}

// Report is the version command body: version, then commit hash and build
// time when those are known. The version stays the first field.
func Report() string {
	return format(Version, Revision(), Built())
}

func format(version, commit, built string) string {
	parts := []string{version}
	if commit != "" {
		parts = append(parts, commit)
	}
	if built != "" {
		parts = append(parts, built)
	}
	return strings.Join(parts, " ")
}

// Revision is the git commit of this binary: ldflags Commit, else vcs.revision.
func Revision() string {
	if Commit != "" {
		return Commit
	}
	return buildSetting("vcs.revision")
}

// Built is when this binary was built: ldflags Date, else vcs.time.
func Built() string {
	if Date != "" {
		return Date
	}
	return buildSetting("vcs.time")
}

func buildSetting(key string) string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range info.Settings {
		if s.Key == key {
			return s.Value
		}
	}
	return ""
}
