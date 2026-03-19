// Package constants provides centralized constants for directory names and paths
// used throughout the utm-dev project.
package constants

const (
	// BinDir is the directory for development binaries and executables
	BinDir = ".bin"

	// BuildDir is the directory for intermediate build artifacts
	BuildDir = ".build"

	// DistDir is the directory for final distribution packages
	DistDir = ".dist"

	// DataDir is the directory for runtime data storage
	DataDir = ".data"
)

// CommonGitIgnorePatterns returns the standard patterns that should be ignored
// for build artifacts and generated content.
func CommonGitIgnorePatterns() []string {
	return []string{
		BinDir + "/",
		BuildDir + "/",
		DistDir + "/",
		DataDir + "/",
		"Assets.xcassets/",
		"drawable-*/",
	}
}
