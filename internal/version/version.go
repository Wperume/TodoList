package version

import (
	"fmt"
	"runtime"
)

// Build information. These are set via -ldflags at build time.
var (
	// Version is the semantic version (e.g., "1.0.0" or "1.0.0-dev")
	Version = "dev"

	// Commit is the git commit SHA
	Commit = "unknown"

	// BuildDate is the date the binary was built
	BuildDate = "unknown"

	// GoVersion is the Go version used to build the binary
	GoVersion = runtime.Version()
)

// Info returns a formatted version string with all build information
func Info() string {
	return fmt.Sprintf("Version: %s, Commit: %s, Built: %s, Go: %s",
		Version, Commit, BuildDate, GoVersion)
}

// Short returns a short version string
func Short() string {
	if Commit != "unknown" && len(Commit) > 7 {
		return fmt.Sprintf("%s (%s)", Version, Commit[:7])
	}
	return Version
}
