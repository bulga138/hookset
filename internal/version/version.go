// Package version holds build-time version metadata injected via -ldflags.
package version

// These variables are set at build time via:
//
//	-X github.com/bulga138/hookset/internal/version.Version=<tag>
//	-X github.com/bulga138/hookset/internal/version.Commit=<sha>
//	-X github.com/bulga138/hookset/internal/version.BuildTime=<date>
var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)
