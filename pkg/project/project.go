package project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joeblew999/utm-dev/pkg/constants"
)

// GioProject represents a Gio application project
type GioProject struct {
	// Root directory of the Gio app
	RootDir string

	// App name (derived from directory name or go.mod)
	Name string

	// Build output directory
	OutputDir string
}

// NewGioProject creates a new GioProject instance
func NewGioProject(rootDir string) (*GioProject, error) {
	// Validate that the directory exists
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("project directory does not exist: %s", rootDir)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Derive app name from directory
	appName := filepath.Base(absPath)

	project := &GioProject{
		RootDir:   absPath,
		Name:      appName,
		OutputDir: filepath.Join(absPath, constants.BinDir),
	}

	return project, nil
}

// NewGioProjectWithOutput creates a new GioProject with a custom output directory
func NewGioProjectWithOutput(rootDir, outputBaseDir string) (*GioProject, error) {
	// Validate that the directory exists
	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("project directory does not exist: %s", rootDir)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Derive app name from directory
	appName := filepath.Base(absPath)

	// Set output directory - use custom base if provided
	var outputDir string
	if outputBaseDir != "" {
		// Convert output base to absolute path
		absOutputBase, err := filepath.Abs(outputBaseDir)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute output path: %w", err)
		}
		outputDir = filepath.Join(absOutputBase, constants.BinDir)
	} else {
		outputDir = filepath.Join(absPath, constants.BinDir)
	}

	project := &GioProject{
		RootDir:   absPath,
		Name:      appName,
		OutputDir: outputDir,
	}

	return project, nil
}

// Paths returns commonly used paths for the project
// Returns PathsInterface for compatibility with icons package
func (p *GioProject) Paths() *ProjectPaths {
	return &ProjectPaths{
		project:      p,
		Root:         p.RootDir,
		Output:       p.OutputDir,
		SourceIcon:   filepath.Join(p.RootDir, "icon-source.png"),
		AndroidIcons: filepath.Join(p.RootDir, constants.BuildDir),
		IOSIcons:     filepath.Join(p.RootDir, constants.BuildDir, "Assets.xcassets"),
		WindowsIcons: filepath.Join(p.RootDir, constants.BuildDir),
		GoMod:        filepath.Join(p.RootDir, "go.mod"),
		MainGo:       filepath.Join(p.RootDir, "main.go"),
		MSIXData:     filepath.Join(p.RootDir, "msix-data.yml"),
		AppConfig:    filepath.Join(p.RootDir, "app.json"),
	}
}

// ProjectPaths contains all commonly used paths for a Gio project
type ProjectPaths struct {
	project      *GioProject // Reference to parent project
	Root         string      // Root directory
	Output       string      // Build output directory (constants.BinDir)
	SourceIcon   string      // Source icon file (icon-source.png)
	AndroidIcons string      // Android icons output directory
	IOSIcons     string      // iOS icons output directory (build/Assets.xcassets)
	WindowsIcons string      // Windows icons output directory
	GoMod        string      // go.mod file
	MainGo       string      // main.go file
	MSIXData     string      // msix-data.yml file
	AppConfig    string      // app.json file
}

// GetSourceIcon returns the path to the source icon file
func (pp *ProjectPaths) GetSourceIcon() string {
	return pp.SourceIcon
}

// GetIconOutputPath returns the appropriate output path for icons based on platform
func (pp *ProjectPaths) GetIconOutputPath(platform string) string {
	switch platform {
	case "android":
		return pp.AndroidIcons
	case "ios", "macos":
		return pp.IOSIcons
	case "windows", "windows-msix", "windows-ico":
		return pp.WindowsIcons
	default:
		return pp.Output
	}
}

// EnsureDirectories creates necessary directories for the project
func (p *GioProject) EnsureDirectories() error {
	paths := p.Paths()

	// Create output directory
	if err := os.MkdirAll(paths.Output, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create Assets.xcassets for iOS in build directory if it doesn't exist
	if err := os.MkdirAll(paths.IOSIcons, 0755); err != nil {
		return fmt.Errorf("failed to create iOS assets directory: %w", err)
	}

	return nil
}

// Validate checks if the project is a valid Gio project
func (p *GioProject) Validate() error {
	paths := p.Paths()

	// Check if go.mod exists
	if _, err := os.Stat(paths.GoMod); os.IsNotExist(err) {
		return fmt.Errorf("go.mod not found - not a valid Go project: %s", paths.GoMod)
	}

	// Check if main.go exists
	if _, err := os.Stat(paths.MainGo); os.IsNotExist(err) {
		return fmt.Errorf("main.go not found: %s", paths.MainGo)
	}

	return nil
}

// HasSourceIcon checks if the project has a source icon
func (p *GioProject) HasSourceIcon() bool {
	paths := p.Paths()
	_, err := os.Stat(paths.SourceIcon)
	return err == nil
}

// GetOutputPath returns the path for a specific platform build
// Uses platform-specific subdirectories to avoid overwrites (e.g., .bin/macos/, .bin/ios/)
func (p *GioProject) GetOutputPath(platform string) string {
	paths := p.Paths()
	platformDir := filepath.Join(paths.Output, platform)
	switch platform {
	case "macos":
		return filepath.Join(platformDir, p.Name+".app")
	case "android":
		return filepath.Join(platformDir, p.Name+".apk")
	case "ios", "ios-simulator":
		return filepath.Join(platformDir, p.Name+".app")
	case "windows":
		return filepath.Join(platformDir, p.Name+".exe")
	case "linux":
		return filepath.Join(platformDir, p.Name)
	default:
		return filepath.Join(platformDir, p.Name)
	}
}

// GetPlatformDir returns the platform-specific output directory (e.g., .bin/macos/)
func (p *GioProject) GetPlatformDir(platform string) string {
	paths := p.Paths()
	return filepath.Join(paths.Output, platform)
}

// GenerateSourceIcon creates a source icon for the project if it doesn't exist
func (p *GioProject) GenerateSourceIcon() error {
	if p.HasSourceIcon() {
		return nil // Already exists
	}

	// This is a placeholder - in a real implementation, you might want to import
	// the icons package here, but that would create a circular dependency.
	// Instead, this should be handled by the calling code.
	return fmt.Errorf("source icon does not exist and cannot be generated from project package")
}
