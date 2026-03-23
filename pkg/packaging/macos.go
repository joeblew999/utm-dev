package packaging

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/joeblew999/utm-dev/pkg/cli"
)

//go:embed templates/macos-info.plist.tmpl
var macosInfoPlistTemplate string

//go:embed templates/macos-entitlements.plist.tmpl
var macosEntitlementsTemplate string

// MacOSBundleConfig contains configuration for creating a macOS app bundle
type MacOSBundleConfig struct {
	// App identity
	Name        string // App name (e.g., "utm-dev")
	DisplayName string // Display name (e.g., "Goup Util")
	BundleID    string // Bundle identifier (e.g., "com.example.myapp")
	Version     string // Version string (e.g., "1.0.0")
	BuildNumber string // Build number (e.g., "1")
	Year        string // Copyright year (auto-filled if empty)

	// Paths
	BinaryPath string // Path to the compiled binary
	OutputDir  string // Where to create the .app bundle
	IconPath   string // Path to .icns icon file (optional)

	// Code signing
	SigningIdentity string // Code signing identity (empty for ad-hoc)
	Entitlements    bool   // Whether to use entitlements
}

// Executable returns the executable name from the app name
func (c *MacOSBundleConfig) Executable() string {
	return c.Name
}

// CreateMacOSBundle creates a properly structured macOS app bundle with code signing
func CreateMacOSBundle(config MacOSBundleConfig) error {
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
	if config.BundleID == "" {
		config.BundleID = fmt.Sprintf("com.example.%s", strings.ToLower(config.Name))
	}
	if config.Version == "" {
		config.Version = "1.0.0"
	}
	if config.BuildNumber == "" {
		config.BuildNumber = "1"
	}
	if config.Year == "" {
		config.Year = fmt.Sprintf("%d", time.Now().Year())
	}

	// Check binary exists
	if _, err := os.Stat(config.BinaryPath); os.IsNotExist(err) {
		return fmt.Errorf("binary not found: %s", config.BinaryPath)
	}

	// Create bundle structure
	appBundlePath := filepath.Join(config.OutputDir, config.Name+".app")
	contentsDir := filepath.Join(appBundlePath, "Contents")
	macOSDir := filepath.Join(contentsDir, "MacOS")
	resourcesDir := filepath.Join(contentsDir, "Resources")

	cli.Info("Creating macOS app bundle: %s", appBundlePath)

	// Create directories
	if err := os.MkdirAll(macOSDir, 0755); err != nil {
		return fmt.Errorf("failed to create MacOS directory: %w", err)
	}
	if err := os.MkdirAll(resourcesDir, 0755); err != nil {
		return fmt.Errorf("failed to create Resources directory: %w", err)
	}

	// Copy binary
	executablePath := filepath.Join(macOSDir, config.Name)
	if err := copyFile(config.BinaryPath, executablePath); err != nil {
		return fmt.Errorf("failed to copy binary: %w", err)
	}
	if err := os.Chmod(executablePath, 0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	cli.Success("Binary copied to %s", executablePath)

	// Copy icon if available
	if config.IconPath != "" {
		if _, err := os.Stat(config.IconPath); err == nil {
			iconDst := filepath.Join(resourcesDir, "AppIcon.icns")
			if err := copyFile(config.IconPath, iconDst); err != nil {
				cli.Warn("Failed to copy icon: %v", err)
			} else {
				cli.Success("Icon copied to %s", iconDst)
			}
		}
	}

	// Generate Info.plist
	infoPlistPath := filepath.Join(contentsDir, "Info.plist")
	if err := generateInfoPlist(infoPlistPath, config); err != nil {
		return fmt.Errorf("failed to generate Info.plist: %w", err)
	}

	cli.Success("Info.plist created")

	// Generate entitlements if needed
	var entitlementsPath string
	if config.Entitlements {
		entitlementsPath = filepath.Join(contentsDir, "Entitlements.plist")
		if err := generateEntitlements(entitlementsPath); err != nil {
			return fmt.Errorf("failed to generate entitlements: %w", err)
		}
		cli.Success("Entitlements.plist created")
	}

	// Code signing
	if err := signBundle(appBundlePath, config.SigningIdentity, entitlementsPath); err != nil {
		return fmt.Errorf("failed to sign bundle: %w", err)
	}

	cli.Success("macOS app bundle created successfully")
	cli.Info("Location: %s", appBundlePath)

	return nil
}

// generateInfoPlist creates the Info.plist from template
func generateInfoPlist(path string, config MacOSBundleConfig) error {
	tmpl, err := template.New("info.plist").Parse(macosInfoPlistTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	return tmpl.Execute(file, config)
}

// generateEntitlements creates the Entitlements.plist from template
func generateEntitlements(path string) error {
	tmpl, err := template.New("entitlements.plist").Parse(macosEntitlementsTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	return tmpl.Execute(file, nil)
}

// signBundle signs the app bundle with the specified identity
func signBundle(bundlePath, identity, entitlementsPath string) error {
	// Detect signing identity if not provided
	if identity == "" {
		cli.Info("Checking for code signing certificate...")
		detectedIdentity := detectSigningIdentity()

		if detectedIdentity == "" {
			cli.Warn("No code signing certificate found")
			cli.Info("   Using ad-hoc signature (suitable for local testing)")
			identity = "-" // Ad-hoc signature
		} else {
			cli.Success("Found signing identity: %s", detectedIdentity)
			identity = detectedIdentity
		}
	}

	// Build codesign command
	args := []string{
		"--force",
		"--deep",
		"--sign", identity,
	}

	// Add entitlements if provided
	if entitlementsPath != "" && identity != "-" {
		args = append(args, "--entitlements", entitlementsPath)
		args = append(args, "--options", "runtime")
	}

	args = append(args, bundlePath)

	cli.Info("Signing app bundle...")
	cmd := exec.Command("codesign", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("codesign failed: %w\nOutput: %s", err, output)
	}

	// Verify signature
	cli.Info("Verifying signature...")
	verifyCmd := exec.Command("codesign", "--verify", "--deep", "--strict", bundlePath)
	if output, err := verifyCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("signature verification failed: %w\nOutput: %s", err, output)
	}

	cli.Success("App bundle signed successfully")

	return nil
}

// detectSigningIdentity tries to find a valid code signing identity
func detectSigningIdentity() string {
	// Try to find Developer ID Application certificate
	cmd := exec.Command("security", "find-identity", "-v", "-p", "codesigning")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Developer ID Application") {
			// Extract identity between quotes
			if start := strings.Index(line, "\""); start != -1 {
				if end := strings.Index(line[start+1:], "\""); end != -1 {
					return line[start+1 : start+1+end]
				}
			}
		}
	}

	// Fall back to Apple Development
	for _, line := range lines {
		if strings.Contains(line, "Apple Development") {
			if start := strings.Index(line, "\""); start != -1 {
				if end := strings.Index(line[start+1:], "\""); end != -1 {
					return line[start+1 : start+1+end]
				}
			}
		}
	}

	return ""
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
