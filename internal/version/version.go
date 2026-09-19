package version

import "runtime/debug"

// Version is the CLI / product version. Override at build time with
// -ldflags "-X github.com/bwireman/archivist/internal/version.Version=..."
var Version = "0.1.0"

// Commit is the source revision. Override at build time with
// -ldflags "-X github.com/bwireman/archivist/internal/version.Commit=..."
// When empty, Revision falls back to the Go VCS build stamp.
var Commit = ""

func String() string {
	return Version
}

// Revision is the git commit of this binary: ldflags Commit, else vcs.revision.
func Revision() string {
	if Commit != "" {
		return Commit
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			return s.Value
		}
	}
	return ""
}
