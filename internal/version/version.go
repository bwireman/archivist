package version

import "fmt"

// Version is the CLI / product version. Override at build time with
// -ldflags "-X github.com/bwireman/archivist/internal/version.Version=..."
var Version = "0.1.0"

// Schema is the index database format. Bump it when the store layout or
// chunk contract changes, and add a migration in store.applyMigrations.
const Schema = 2

func String() string {
	return fmt.Sprintf("%s (schema %d)", Version, Schema)
}
