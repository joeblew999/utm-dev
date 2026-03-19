// Package utm provides UTM virtual machine management for utm-dev.
//
// Architecture:
//   - This package handles VM configuration, paths, and gallery management
//   - cmd/utm.go provides the CLI interface using this package
//
// Paths:
//   Global (shared across projects - ~/utm-dev-sdks/utm/):
//   - ~/utm-dev-sdks/utm/UTM.app  - UTM application
//   - ~/utm-dev-sdks/utm/iso/     - ISO images for VM creation
//   - ~/utm-dev-sdks/utm/vms/     - Virtual machine files (.utm)
//   - ~/utm-dev-sdks/utm/share/   - Shared folder for host<->VM file transfer
package utm

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/joeblew999/utm-dev/pkg/config"
)

// Paths holds all UTM-related paths
type Paths struct {
	// Root is the base directory for all UTM data (default: .data/utm)
	Root string

	// App is where UTM.app is installed (default: .bin/UTM.app)
	App string

	// VMs is where virtual machines are stored
	VMs string

	// ISO is where ISO images are stored
	ISO string

	// Share is the shared folder for host<->guest file transfer
	Share string
}

// DefaultPaths returns the default UTM paths
// All paths are global (shared across projects)
func DefaultPaths() Paths {
	sdkDir := config.GetSDKDir() // ~/utm-dev-sdks

	return Paths{
		// Global paths (shared across projects)
		Root:  filepath.Join(sdkDir, "utm"),
		App:   filepath.Join(sdkDir, "utm", "UTM.app"),
		ISO:   filepath.Join(sdkDir, "utm", "iso"),
		VMs:   filepath.Join(sdkDir, "utm", "vms"),
		Share: filepath.Join(sdkDir, "utm", "share"),
	}
}

// LegacyPaths returns the old local paths (for migration)
func LegacyPaths() Paths {
	return Paths{
		Root:  ".data/utm",
		App:   ".bin/UTM.app",
		VMs:   ".data/utm/vms",
		ISO:   ".data/utm/iso",
		Share: ".data/utm/share",
	}
}

// GetPaths returns UTM paths, using defaults if not configured
func GetPaths() Paths {
	// TODO: Load from config file if present
	return DefaultPaths()
}

// GetUTMCtlPath returns the path to the utmctl binary
func GetUTMCtlPath() string {
	if runtime.GOOS != "darwin" {
		return "" // UTM is macOS only
	}

	paths := GetPaths()
	legacy := LegacyPaths()

	// Check common locations in order of preference
	locations := []string{
		// Global install (preferred - new location)
		filepath.Join(paths.App, "Contents/MacOS/utmctl"),
		// Legacy local install (for migration)
		filepath.Join(legacy.App, "Contents/MacOS/utmctl"),
		// Homebrew
		"/opt/homebrew/bin/utmctl",
		"/usr/local/bin/utmctl",
		// System install
		"/Applications/UTM.app/Contents/MacOS/utmctl",
	}

	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}

	// Fallback to PATH lookup
	return "utmctl"
}

// IsUTMInstalled checks if UTM is available
func IsUTMInstalled() bool {
	path := GetUTMCtlPath()
	if path == "utmctl" {
		// Check if utmctl is in PATH
		_, err := os.Stat(path)
		return err == nil
	}
	_, err := os.Stat(path)
	return err == nil
}

// GetVMPath returns the full path for a VM by name
func GetVMPath(vmName string) string {
	paths := GetPaths()
	return filepath.Join(paths.VMs, vmName+".utm")
}

// GetISOPath returns the full path for an ISO by name
func GetISOPath(isoName string) string {
	paths := GetPaths()
	return filepath.Join(paths.ISO, isoName)
}

// EnsureDirectories creates all required UTM directories (all global)
func EnsureDirectories() error {
	paths := GetPaths()

	dirs := []string{
		paths.Root,  // ~/utm-dev-sdks/utm
		paths.ISO,   // ~/utm-dev-sdks/utm/iso
		paths.VMs,   // ~/utm-dev-sdks/utm/vms
		paths.Share, // ~/utm-dev-sdks/utm/share
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

// EnsureGlobalDirectories creates all UTM directories (all are global now)
func EnsureGlobalDirectories() error {
	paths := GetPaths()

	dirs := []string{
		paths.Root,  // ~/utm-dev-sdks/utm
		paths.ISO,   // ~/utm-dev-sdks/utm/iso
		paths.VMs,   // ~/utm-dev-sdks/utm/vms
		paths.Share, // ~/utm-dev-sdks/utm/share
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

