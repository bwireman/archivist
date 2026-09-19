package version

// Version is the CLI / product version. Override at build time with
// -ldflags "-X github.com/bwireman/archivist/internal/version.Version=..."
var Version = "0.1.0"

func String() string {
	return Version
}
