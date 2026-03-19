package self

import (
	"fmt"
	"runtime"
)

// Architecture defines a build target with OS, CPU architecture, and output file suffix.
// The Suffix includes the file extension for the target platform (e.g., ".exe" for Windows).
type Architecture struct {
	GOOS   string // Target operating system (darwin, linux, windows)
	GOARCH string // Target CPU architecture (amd64, arm64)
	Suffix string // Binary filename suffix including extension
}

// SupportedArchitectures returns all architectures we build for.
// This is the single source of truth for supported platforms.
//
// To add a new architecture:
// 1. Add entry here
// 2. Run: go run . self build
// 3. Verify bootstrap scripts are updated automatically
func SupportedArchitectures() []Architecture {
	return []Architecture{
		{GOOS: "darwin", GOARCH: "arm64", Suffix: "darwin-arm64"},
		{GOOS: "darwin", GOARCH: "amd64", Suffix: "darwin-amd64"},
		{GOOS: "linux", GOARCH: "amd64", Suffix: "linux-amd64"},
		{GOOS: "linux", GOARCH: "arm64", Suffix: "linux-arm64"},
		{GOOS: "windows", GOARCH: "amd64", Suffix: "windows-amd64.exe"},
		{GOOS: "windows", GOARCH: "arm64", Suffix: "windows-arm64.exe"},
	}
}

// FilterByOS returns architectures for a specific OS.
// Returns empty slice if targetOS has no matches.
func FilterByOS(archs []Architecture, targetOS string) []Architecture {
	if len(archs) == 0 {
		return nil
	}

	// Pre-allocate with estimated capacity
	filtered := make([]Architecture, 0, 2) // Most OSes have 2 architectures
	
	for _, arch := range archs {
		if arch.GOOS == targetOS {
			filtered = append(filtered, arch)
		}
	}
	
	return filtered
}

// ArchsToGoArchList extracts GOARCH values from architectures.
// Preserves order from input slice.
func ArchsToGoArchList(archs []Architecture) []string {
	if len(archs) == 0 {
		return nil
	}

	// Pre-allocate exact size
	list := make([]string, 0, len(archs))
	
	for _, arch := range archs {
		list = append(list, arch.GOARCH)
	}
	
	return list
}

// BinaryName returns the full binary filename for an architecture.
// Format: utm-dev-{suffix}
func (a Architecture) BinaryName() string {
	return "utm-dev-" + a.Suffix
}

// Validate checks if an Architecture has all required fields.
func (a Architecture) Validate() error {
	if a.GOOS == "" {
		return fmt.Errorf("GOOS is required")
	}
	if a.GOARCH == "" {
		return fmt.Errorf("GOARCH is required")
	}
	if a.Suffix == "" {
		return fmt.Errorf("Suffix is required")
	}
	return nil
}

// CurrentArchitecture returns the Architecture for the current runtime platform.
// Returns nil if the current platform is not in SupportedArchitectures.
func CurrentArchitecture() *Architecture {
	for _, arch := range SupportedArchitectures() {
		if arch.GOOS == runtime.GOOS && arch.GOARCH == runtime.GOARCH {
			return &arch
		}
	}
	return nil
}
