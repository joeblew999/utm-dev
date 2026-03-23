package packaging

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"text/template"

	"github.com/joeblew999/utm-dev/pkg/cli"
)

//go:embed templates/windows-appxmanifest.xml.tmpl
var windowsManifestTemplate string

// WindowsBundleConfig contains configuration for creating a Windows MSIX bundle
type WindowsBundleConfig struct {
	// App identity
	Name                 string // App name (e.g., "MyApp")
	Publisher            string // Publisher (e.g., "CN=MyCompany")
	PublisherDisplayName string // Display name for publisher
	DisplayName          string // Display name shown to users
	Description          string // App description
	Version              string // Version in format "1.0.0.0" (4 parts required)

	// Paths
	BinaryPath string // Path to the compiled .exe
	OutputDir  string // Where to create the MSIX bundle
	AssetsDir  string // Path to logo assets (optional)

	// Packaging options
	CreateMSIX bool // Whether to create the actual MSIX (Windows-only)

	// Code signing (future)
	SigningCertificate  string // Path to .pfx file
	CertificatePassword string // Password for certificate
}

// CreateWindowsBundle creates a properly structured Windows MSIX bundle
func CreateWindowsBundle(config WindowsBundleConfig) error {
	// Validate config
	if config.Name == "" {
		return fmt.Errorf("app name is required")
	}
	if config.BinaryPath == "" {
		return fmt.Errorf("binary path is required")
	}
	if config.OutputDir == "" {
		return fmt.Errorf("output directory is required")
	}

	// Set defaults
	if config.DisplayName == "" {
		config.DisplayName = config.Name
	}
	if config.Publisher == "" {
		config.Publisher = fmt.Sprintf("CN=%s", config.Name)
	}
	if config.PublisherDisplayName == "" {
		config.PublisherDisplayName = config.Name
	}
	if config.Description == "" {
		config.Description = config.DisplayName
	}
	if config.Version == "" {
		config.Version = "1.0.0.0"
	}

	// Ensure version has 4 parts (required by MSIX)
	config.Version = normalizeVersion(config.Version)

	// Check binary exists
	if _, err := os.Stat(config.BinaryPath); os.IsNotExist(err) {
		return fmt.Errorf("binary not found: %s", config.BinaryPath)
	}

	cli.Info("Creating Windows MSIX bundle: %s", config.Name)

	// Create bundle staging directory
	stagingDir := filepath.Join(config.OutputDir, ".staging")
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}
	// Only clean up staging after successful MSIX creation on Windows
	cleanupStaging := false
	defer func() {
		if cleanupStaging {
			os.RemoveAll(stagingDir)
		}
	}()

	// Copy binary to staging
	executableName := config.Name + ".exe"
	targetBinary := filepath.Join(stagingDir, executableName)
	if err := CopyFile(config.BinaryPath, targetBinary); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}
	cli.Success("Binary copied: %s", executableName)

	// Create assets directory
	assetsDir := filepath.Join(stagingDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		return fmt.Errorf("failed to create assets directory: %w", err)
	}

	// Copy or generate assets
	if config.AssetsDir != "" {
		// Copy user-provided assets
		if err := copyAssets(config.AssetsDir, assetsDir); err != nil {
			return fmt.Errorf("failed to copy assets: %w", err)
		}
		cli.Success("Assets copied from %s", config.AssetsDir)
	} else {
		// Generate placeholder assets (required by MSIX)
		if err := generatePlaceholderAssets(assetsDir); err != nil {
			return fmt.Errorf("failed to generate placeholder assets: %w", err)
		}
		cli.Success("Generated placeholder assets")
	}

	// Generate AppxManifest.xml
	manifestPath := filepath.Join(stagingDir, "AppxManifest.xml")
	manifestConfig := config
	manifestConfig.Name = executableName // Executable field in manifest

	if err := generateWindowsManifest(manifestPath, manifestConfig); err != nil {
		return fmt.Errorf("failed to generate AppxManifest.xml: %w", err)
	}
	cli.Success("AppxManifest.xml created")

	// Create MSIX package (Windows-only)
	if config.CreateMSIX {
		if runtime.GOOS != "windows" {
			cli.Warn("Skipping MSIX creation: requires Windows")
			cli.Info("   Bundle structure created in .staging/, ready to package on Windows")
		} else {
			msixPath := filepath.Join(config.OutputDir, config.Name+".msix")
			if err := createMSIXPackage(stagingDir, msixPath); err != nil {
				return fmt.Errorf("failed to create MSIX package: %w", err)
			}
			cli.Success("MSIX package created: %s", msixPath)

			// Sign the MSIX if certificate provided
			if config.SigningCertificate != "" {
				if err := signMSIX(msixPath, config.SigningCertificate, config.CertificatePassword); err != nil {
					return fmt.Errorf("failed to sign MSIX: %w", err)
				}
				cli.Success("MSIX signed with certificate")
			}

			// Clean up staging after successful MSIX creation
			cleanupStaging = true
		}
	}

	cli.Success("Windows bundle created successfully")
	if runtime.GOOS != "windows" {
		cli.Info("Bundle structure: %s", stagingDir)
		cli.Info("   Copy to Windows to complete packaging")
	} else {
		cli.Info("MSIX location: %s", filepath.Join(config.OutputDir, config.Name+".msix"))
	}

	return nil
}

// generateWindowsManifest creates the AppxManifest.xml from template
func generateWindowsManifest(path string, config WindowsBundleConfig) error {
	tmpl, err := template.New("manifest").Parse(windowsManifestTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Template expects lowercase field names
	data := map[string]interface{}{
		"name":                 config.Name,
		"publisher":            config.Publisher,
		"version":              config.Version,
		"displayName":          config.DisplayName,
		"publisherDisplayName": config.PublisherDisplayName,
		"executable":           config.Name, // Just the name, .exe added by template
		"description":          config.Description,
	}

	return tmpl.Execute(file, data)
}

// createMSIXPackage creates the MSIX file using the msix toolkit
func createMSIXPackage(sourceDir, outputPath string) error {
	// Check if msix command is available
	msixPath, err := exec.LookPath("msix")
	if err != nil {
		return fmt.Errorf("msix command not found. Install via: winget install Microsoft.MsixPackagingTool")
	}

	// Run msix pack command
	cmd := exec.Command(msixPath, "pack", "-d", sourceDir, "-p", outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("msix pack failed: %w\nOutput: %s", err, output)
	}

	return nil
}

// signMSIX signs an MSIX package using signtool
func signMSIX(msixPath, certPath, password string) error {
	// Check if signtool is available
	signtool, err := exec.LookPath("signtool")
	if err != nil {
		return fmt.Errorf("signtool not found. Install Windows SDK or use Visual Studio Developer Command Prompt")
	}

	// Build signtool command
	args := []string{
		"sign",
		"/fd", "SHA256",
		"/f", certPath,
	}

	if password != "" {
		args = append(args, "/p", password)
	}

	args = append(args, msixPath)

	cmd := exec.Command(signtool, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("signtool failed: %w\nOutput: %s", err, output)
	}

	return nil
}

// copyAssets copies asset files from source to destination
func copyAssets(sourceDir, destDir string) error {
	// Required MSIX assets:
	// - logo.png (at least 50x50)
	// - Square150x150Logo.png
	// - Square44x44Logo.png

	assets := []string{
		"logo.png",
		"Square150x150Logo.png",
		"Square44x44Logo.png",
	}

	for _, asset := range assets {
		src := filepath.Join(sourceDir, asset)
		dst := filepath.Join(destDir, asset)

		if _, err := os.Stat(src); err == nil {
			if err := CopyFile(src, dst); err != nil {
				return fmt.Errorf("failed to copy %s: %w", asset, err)
			}
		}
	}

	return nil
}

// generatePlaceholderAssets creates minimal placeholder assets for MSIX
func generatePlaceholderAssets(destDir string) error {
	// Create minimal 1x1 PNG placeholders
	// In a real implementation, you'd generate proper PNG files
	// For now, we'll create empty files with a warning

	assets := []string{
		"logo.png",
		"Square150x150Logo.png",
		"Square44x44Logo.png",
	}

	for _, asset := range assets {
		path := filepath.Join(destDir, asset)
		if err := os.WriteFile(path, []byte{}, 0644); err != nil {
			return fmt.Errorf("failed to create %s: %w", asset, err)
		}
	}

	cli.Warn("Using placeholder assets - provide real icons for production")

	return nil
}

// normalizeVersion ensures version has 4 parts (required by MSIX)
func normalizeVersion(version string) string {
	// Split by dots
	parts := []string{"1", "0", "0", "0"}

	// Parse input
	var parsed []string
	current := ""
	for _, ch := range version {
		if ch == '.' {
			if current != "" {
				parsed = append(parsed, current)
				current = ""
			}
		} else if ch >= '0' && ch <= '9' {
			current += string(ch)
		}
	}
	if current != "" {
		parsed = append(parsed, current)
	}

	// Copy up to 4 parts
	for i := 0; i < len(parsed) && i < 4; i++ {
		parts[i] = parsed[i]
	}

	return fmt.Sprintf("%s.%s.%s.%s", parts[0], parts[1], parts[2], parts[3])
}
