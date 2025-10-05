package version

import (
	"fmt"
	"runtime"
	"time"
)

// Version information - set during build time
var (
	// Version is the semantic version of the application
	Version = "1.2.0"

	// GitCommit is the git commit hash
	GitCommit = "unknown"

	// GitBranch is the git branch
	GitBranch = "unknown"

	// BuildDate is when the binary was built
	BuildDate = "unknown"

	// BuildUser is who built the binary
	BuildUser = "unknown"

	// GoVersion is the Go version used to compile
	GoVersion = runtime.Version()

	// GitTag is the git tag (if any)
	GitTag = "unknown"

	// GitDirty indicates if the working directory was dirty
	GitDirty = "unknown"
)

// BuildInfo contains all version and build information
type BuildInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	GitBranch string `json:"git_branch"`
	GitTag    string `json:"git_tag"`
	GitDirty  string `json:"git_dirty"`
	BuildDate string `json:"build_date"`
	BuildUser string `json:"build_user"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
	Service   string `json:"service"`
}

// GetBuildInfo returns complete build information
func GetBuildInfo(serviceName string) *BuildInfo {
	return &BuildInfo{
		Version:   Version,
		GitCommit: GitCommit,
		GitBranch: GitBranch,
		GitTag:    GitTag,
		GitDirty:  GitDirty,
		BuildDate: BuildDate,
		BuildUser: BuildUser,
		GoVersion: GoVersion,
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
		Service:   serviceName,
	}
}

// GetVersion returns the semantic version
func GetVersion() string {
	return Version
}

// GetShortVersion returns version with commit hash
func GetShortVersion() string {
	if GitCommit != "unknown" && len(GitCommit) > 7 {
		return fmt.Sprintf("%s-%s", Version, GitCommit[:7])
	}
	return Version
}

// GetFullVersion returns complete version information
func GetFullVersion(serviceName string) string {
	return fmt.Sprintf(`%s version %s
Git Commit: %s
Git Branch: %s
Git Tag: %s
Build Date: %s
Build User: %s
Go Version: %s
Platform: %s/%s`,
		serviceName,
		Version,
		GitCommit,
		GitBranch,
		GitTag,
		BuildDate,
		BuildUser,
		GoVersion,
		runtime.GOOS,
		runtime.GOARCH,
	)
}

// IsDevBuild returns true if this is a development build
func IsDevBuild() bool {
	return GitCommit == "unknown" || GitDirty == "true"
}

// GetUserAgent returns a user agent string for HTTP requests
func GetUserAgent(serviceName string) string {
	return fmt.Sprintf("OpenUSP-%s/%s (%s/%s; %s)",
		serviceName,
		Version,
		runtime.GOOS,
		runtime.GOARCH,
		GoVersion)
}

// PrintVersionInfo prints version information to console
func PrintVersionInfo(serviceName string) {
	fmt.Printf("🚀 %s v%s\n", serviceName, Version)
	if GitCommit != "unknown" {
		fmt.Printf("   📝 Commit: %s", GitCommit)
		if len(GitCommit) > 7 {
			fmt.Printf(" (%s)", GitCommit[:7])
		}
		fmt.Println()
	}
	if GitBranch != "unknown" {
		fmt.Printf("   🌿 Branch: %s\n", GitBranch)
	}
	if GitTag != "unknown" && GitTag != "" {
		fmt.Printf("   🏷️  Tag: %s\n", GitTag)
	}
	if BuildDate != "unknown" {
		fmt.Printf("   🔨 Built: %s\n", BuildDate)
	}
	if IsDevBuild() {
		fmt.Printf("   ⚠️  Development Build\n")
	}
}

// GetCompatibleVersions returns USP protocol versions supported
func GetCompatibleVersions() []string {
	return []string{"1.3", "1.4"}
}

// GetSupportedProtocols returns supported protocols
func GetSupportedProtocols() []string {
	return []string{"USP", "CWMP", "HTTP", "WebSocket", "MQTT", "STOMP"}
}

// GetBuildTime returns build time as time.Time
func GetBuildTime() time.Time {
	if BuildDate == "unknown" {
		return time.Time{}
	}

	// Try parsing different time formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, BuildDate); err == nil {
			return t
		}
	}

	return time.Time{}
}
