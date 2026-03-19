package icons

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"github.com/joeblew999/utm-dev/pkg/constants"
)

// Config holds configuration for icon generation
type Config struct {
	InputPath  string // Path to source icon (e.g., "icon-source.png")
	OutputPath string // Directory to output icons
	Platform   string // Target platform: android, ios, macos, windows-msix, windows-ico
}

// ProjectConfig holds configuration for project-aware icon generation
type ProjectConfig struct {
	ProjectPath string // Path to the project directory
	Platform    string // Target platform
}

// GenerateTestIcon creates a simple blue test icon
func GenerateTestIcon(outputPath string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, 1024, 1024))
	// Simple blue color
	blue := color.RGBA{0, 0, 255, 255}
	for x := 0; x < 1024; x++ {
		for y := 0; y < 1024; y++ {
			img.Set(x, y, blue)
		}
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

// GenerateForProject creates platform-specific icons for a project
func GenerateForProject(cfg ProjectConfig) error {
	// This function provides a temporary bridge - in a service architecture,
	// you'd want to have proper dependency injection here

	// For now, we'll call the existing EnsureSourceIcon function
	sourceIconPath, err := EnsureSourceIcon(cfg.ProjectPath)
	if err != nil {
		return fmt.Errorf("failed to ensure source icon: %w", err)
	}

	// Determine output path based on platform and ensure directories exist
	var outputPath string
	switch cfg.Platform {
	case "android":
		outputPath = filepath.Join(cfg.ProjectPath, constants.BuildDir)
	case "ios", "macos":
		outputPath = filepath.Join(cfg.ProjectPath, constants.BuildDir, "Assets.xcassets")
		// Ensure Assets.xcassets directory exists in build artifacts
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			return fmt.Errorf("failed to create Assets.xcassets directory: %w", err)
		}
	case "windows", "windows-msix", "windows-ico":
		outputPath = filepath.Join(cfg.ProjectPath, constants.BuildDir)
		// Ensure build directory exists
		if err := os.MkdirAll(outputPath, 0755); err != nil {
			return fmt.Errorf("failed to create build directory: %w", err)
		}
	default:
		outputPath = cfg.ProjectPath
	}

	// Generate icons for the platform
	return Generate(Config{
		InputPath:  sourceIconPath,
		OutputPath: outputPath,
		Platform:   cfg.Platform,
	})
}

// Generate creates platform-specific icons from a source image
func Generate(cfg Config) error {
	switch cfg.Platform {
	case "android":
		return generateAndroidIcons(cfg.InputPath, cfg.OutputPath)
	case "ios":
		return generateIOSIcons(cfg.InputPath, cfg.OutputPath)
	case "macos":
		return generateICNS(cfg.InputPath, cfg.OutputPath)
	case "windows-msix":
		return generateWindowsIcons(cfg.InputPath, cfg.OutputPath)
	case "windows-ico":
		return generateICO(cfg.InputPath, cfg.OutputPath)
	default:
		return fmt.Errorf("unsupported platform: %s", cfg.Platform)
	}
}

// EnsureSourceIcon generates a test icon if the source doesn't exist
// Deprecated: Use GenerateForProject instead for better project management
func EnsureSourceIcon(appDir string) (string, error) {
	sourceIconPath := filepath.Join(appDir, "icon-source.png")
	if _, err := os.Stat(sourceIconPath); os.IsNotExist(err) {
		if err := GenerateTestIcon(sourceIconPath); err != nil {
			return "", fmt.Errorf("failed to generate source icon: %w", err)
		}
	}
	return sourceIconPath, nil
}
