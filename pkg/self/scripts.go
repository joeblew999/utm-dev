package self

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/joeblew999/utm-dev/pkg/cli"
)

//go:embed templates/macos-bootstrap.sh.tmpl
var macosTemplate string

//go:embed templates/windows-bootstrap.ps1.tmpl
var windowsTemplate string

const (
	// SetupCommand is the command that bootstrap scripts must call.
	// This is the single source of truth for the bootstrap command.
	SetupCommand = "self setup"
)

// Config holds the configuration for generating bootstrap scripts.
// All string fields are required except LocalBinDir (only used in LOCAL mode).
type Config struct {
	Repo            string   // GitHub repository (e.g., FullRepoName)
	SupportedArchs  string   // Human-readable list (e.g., "arm64, amd64")
	MacOSArchs      []string // macOS architectures (e.g., ["arm64", "amd64"])
	WindowsArchs    []string // Windows architectures (e.g., ["amd64", "arm64"])
	LocalBinDir     string   // Optional: local directory with binaries (for testing)
	UseLocal        bool     // If true, use local binaries instead of GitHub releases
	SetupCommand    string   // Command to run for setup (auto-populated with SetupCommand const)
}

// Validate checks if the Config has all required fields.
func (c Config) Validate() error {
	if c.Repo == "" {
		return fmt.Errorf("Repo is required")
	}
	if c.SupportedArchs == "" {
		return fmt.Errorf("SupportedArchs is required")
	}
	if len(c.MacOSArchs) == 0 && len(c.WindowsArchs) == 0 {
		return fmt.Errorf("at least one of MacOSArchs or WindowsArchs is required")
	}
	if c.UseLocal && c.LocalBinDir == "" {
		return fmt.Errorf("LocalBinDir is required when UseLocal is true")
	}
	if c.SetupCommand != SetupCommand {
		return fmt.Errorf("SetupCommand must be %q, got %q", SetupCommand, c.SetupCommand)
	}
	return nil
}

// Generate creates bootstrap scripts from templates.
// The scripts will be created in outputDir with the following names:
//   - macos-bootstrap.sh (if MacOSArchs is set)
//   - windows-bootstrap.ps1 (if WindowsArchs is set)
//
// In RELEASE mode (UseLocal=false), scripts download binaries from GitHub.
// In LOCAL mode (UseLocal=true), scripts copy binaries from LocalBinDir.
func Generate(outputDir string, config Config) error {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %q: %w", outputDir, err)
	}

	// Generate macOS bootstrap if architectures are specified
	if len(config.MacOSArchs) > 0 {
		macosPath := filepath.Join(outputDir, "macos-bootstrap.sh")
		if err := generateScript(macosPath, "macos-bootstrap.sh.tmpl", macosTemplate, config, 0755); err != nil {
			return fmt.Errorf("failed to generate macOS bootstrap: %w", err)
		}
		cli.Success("Generated %s", macosPath)
	}

	// Generate Windows bootstrap if architectures are specified
	if len(config.WindowsArchs) > 0 {
		windowsPath := filepath.Join(outputDir, "windows-bootstrap.ps1")
		if err := generateScript(windowsPath, "windows-bootstrap.ps1.tmpl", windowsTemplate, config, 0644); err != nil {
			return fmt.Errorf("failed to generate Windows bootstrap: %w", err)
		}
		cli.Success("Generated %s", windowsPath)
	}

	return nil
}

// generateScript creates a script file from a template.
func generateScript(outputPath, templateName, tmplContent string, config Config, perm os.FileMode) error {
	// Parse template
	tmpl, err := template.New(templateName).Parse(tmplContent)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create output file
	f, err := os.OpenFile(outputPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w", outputPath, err)
	}
	defer f.Close()

	// Execute template
	if err := tmpl.Execute(f, config); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// ArchsToString converts a slice of architectures to a comma-separated string.
// Returns empty string if archs is nil or empty.
func ArchsToString(archs []string) string {
	if len(archs) == 0 {
		return ""
	}
	return strings.Join(archs, ", ")
}


// init validates that templates are valid at startup.
// Panics if templates are malformed (this is intentional - fail fast at startup).
func init() {
	// Validate macOS template
	if _, err := template.New("macos").Parse(macosTemplate); err != nil {
		panic(fmt.Sprintf("invalid macOS bootstrap template: %v", err))
	}

	// Validate Windows template
	if _, err := template.New("windows").Parse(windowsTemplate); err != nil {
		panic(fmt.Sprintf("invalid Windows bootstrap template: %v", err))
	}
}

// generateScriptToString is the DRY implementation for generating scripts to strings.
func generateScriptToString(name, tmplContent string, config Config) (string, error) {
	if err := config.Validate(); err != nil {
		return "", fmt.Errorf("invalid config: %w", err)
	}

	tmpl, err := template.New(name).Parse(tmplContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf strings.Builder
	if err := tmpl.Execute(&buf, config); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// GenerateMacOSScript generates a macOS bootstrap script and returns it as a string.
// Useful for testing without writing files.
func GenerateMacOSScript(config Config) (string, error) {
	return generateScriptToString("macos", macosTemplate, config)
}

// GenerateWindowsScript generates a Windows bootstrap script and returns it as a string.
// Useful for testing without writing files.
func GenerateWindowsScript(config Config) (string, error) {
	return generateScriptToString("windows", windowsTemplate, config)
}

// init validates that templates are valid at startup.
// Panics if templates are malformed (this is intentional - fail fast at startup).
func init() {
	// Validate macOS template
	if _, err := template.New("macos").Parse(macosTemplate); err != nil {
		panic(fmt.Sprintf("invalid macOS bootstrap template: %v", err))
	}

	// Validate Windows template
	if _, err := template.New("windows").Parse(windowsTemplate); err != nil {
		panic(fmt.Sprintf("invalid Windows bootstrap template: %v", err))
	}
}
