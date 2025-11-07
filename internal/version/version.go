package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the semantic version (set by ldflags during build)
	Version = "dev"
	// Commit is the git commit hash (set by ldflags during build)
	Commit = "unknown"
	// Date is the build date (set by ldflags during build)
	Date = "unknown"
)

// Info contains complete version information
type Info struct {
	Version   string
	Commit    string
	Date      string
	GoVersion string
	Platform  string
}

// GetInfo returns complete version information
func GetInfo() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		Date:      Date,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a formatted version string
func (i Info) String() string {
	commitShort := i.Commit
	if len(commitShort) > 8 {
		commitShort = commitShort[:8]
	}
	return fmt.Sprintf("Specular %s (%s) built %s with %s for %s",
		i.Version, commitShort, i.Date, i.GoVersion, i.Platform)
}

// Short returns just the version number
func (i Info) Short() string {
	return i.Version
}
