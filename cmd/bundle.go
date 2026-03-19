package cmd

import (
	"github.com/joeblew999/utm-dev/pkg/utils"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joeblew999/utm-dev/pkg/constants"
	"github.com/joeblew999/utm-dev/pkg/packaging"
	"github.com/joeblew999/utm-dev/pkg/project"
	"github.com/spf13/cobra"
)

var bundleCmd = &cobra.Command{
	Use:   "bundle [platform] [app-directory]",
	Short: "Create signed app bundles for distribution",
	Long: `Create properly signed and structured app bundles for distribution.
This includes:
- macOS: .app bundle with Info.plist, code signing, and entitlements
- Android: Signed APK (future)
- iOS: Signed IPA (future)
- Windows: Installer (future)

This is different from 'package' which just creates archives of built apps.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		platform := args[0]
		appDir := args[1]

		// Validate platform
		validPlatforms := []string{"macos", "android", "ios", "windows"}
		if !utils.Contains(validPlatforms, platform) {
			return fmt.Errorf("invalid platform: %s. Valid platforms: %v", platform, validPlatforms)
		}

		// Get flags
		bundleID, _ := cmd.Flags().GetString("bundle-id")
		version, _ := cmd.Flags().GetString("version")
		signingIdentity, _ := cmd.Flags().GetString("sign")
		outputDir, _ := cmd.Flags().GetString("output")
		entitlements, _ := cmd.Flags().GetBool("entitlements")

		// Create and validate project
		proj, err := project.NewGioProject(appDir)
		if err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}

		if err := proj.Validate(); err != nil {
			return fmt.Errorf("invalid project: %w", err)
		}

		switch platform {
		case "macos":
			return bundleMacOS(proj, bundleID, version, signingIdentity, outputDir, entitlements)
		case "android":
			return fmt.Errorf("android bundling not yet implemented")
		case "ios":
			return fmt.Errorf("ios bundling not yet implemented")
		case "windows":
			publisher, _ := cmd.Flags().GetString("publisher")
			createMSIX, _ := cmd.Flags().GetBool("create-msix")
			return bundleWindows(proj, bundleID, version, publisher, outputDir, createMSIX)
		}

		return nil
	},
}

func bundleMacOS(proj *project.GioProject, bundleID, version, signingIdentity, outputDir string, useEntitlements bool) error {
	fmt.Printf("Creating macOS bundle for %s...\n", proj.Name)

	// Set defaults
	if bundleID == "" {
		bundleID = fmt.Sprintf("com.example.%s", proj.Name)
	}
	if version == "" {
		version = "1.0.0"
	}
	if outputDir == "" {
		outputDir = filepath.Join(proj.RootDir, constants.DistDir)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Find the built binary - check multiple locations
	binDir := filepath.Join(proj.RootDir, constants.BinDir)
	var binaryPath string

	// Check locations in order of preference:
	// 1. Platform-specific directory: .bin/macos/<name>.app
	// 2. Legacy location: .bin/<name>.app
	// 3. Standalone binary: .bin/<name>

	macosBinDir := filepath.Join(binDir, "macos")
	platformAppBundle := filepath.Join(macosBinDir, proj.Name+".app")
	platformBinaryInApp := filepath.Join(platformAppBundle, "Contents", "MacOS", proj.Name)

	legacyAppBundle := filepath.Join(binDir, proj.Name+".app")
	legacyBinaryInApp := filepath.Join(legacyAppBundle, "Contents", "MacOS", proj.Name)

	standaloneBinary := filepath.Join(binDir, proj.Name)

	if _, err := os.Stat(platformBinaryInApp); err == nil {
		binaryPath = platformBinaryInApp
		fmt.Println("ℹ️  Found binary in .bin/macos/ bundle, will create new signed bundle")
	} else if _, err := os.Stat(legacyBinaryInApp); err == nil {
		binaryPath = legacyBinaryInApp
		fmt.Println("ℹ️  Found binary in existing .app bundle, will create new signed bundle")
	} else if _, err := os.Stat(standaloneBinary); err == nil {
		binaryPath = standaloneBinary
	} else {
		return fmt.Errorf("binary not found in:\n  %s\n  %s\n  %s\nRun 'utm-dev build macos %s' first",
			platformBinaryInApp, legacyBinaryInApp, standaloneBinary, proj.RootDir)
	}

	// Find icon file (check multiple locations)
	var iconPath string
	iconLocations := []string{
		filepath.Join(proj.RootDir, "assets", "icon.icns"),
		filepath.Join(proj.RootDir, "assets", "AppIcon.icns"),
		filepath.Join(platformAppBundle, "Contents", "Resources", "icon.icns"),
		filepath.Join(legacyAppBundle, "Contents", "Resources", "icon.icns"),
	}
	for _, loc := range iconLocations {
		if _, err := os.Stat(loc); err == nil {
			iconPath = loc
			break
		}
	}

	// Create bundle config
	config := packaging.MacOSBundleConfig{
		Name:            proj.Name,
		DisplayName:     toDisplayName(proj.Name),
		BundleID:        bundleID,
		Version:         version,
		BuildNumber:     "1",
		BinaryPath:      binaryPath,
		OutputDir:       outputDir,
		IconPath:        iconPath,
		SigningIdentity: signingIdentity,
		Entitlements:    useEntitlements,
	}

	// Create the bundle
	if err := packaging.CreateMacOSBundle(config); err != nil {
		return fmt.Errorf("failed to create bundle: %w", err)
	}

	fmt.Println()
	fmt.Println("🎯 Next steps:")
	fmt.Println("   1. Test the app: open", filepath.Join(outputDir, proj.Name+".app"))
	fmt.Println("   2. Grant permissions if needed (System Settings → Privacy & Security)")
	fmt.Println("   3. Package for distribution:")
	fmt.Printf("      task package:macos:dmg OR\n")
	fmt.Printf("      utm-dev package macos %s\n", proj.RootDir)

	return nil
}

func bundleWindows(proj *project.GioProject, bundleID, version, publisher, outputDir string, createMSIX bool) error {
	fmt.Printf("Creating Windows bundle for %s...\n", proj.Name)

	// Set defaults
	if bundleID == "" {
		bundleID = proj.Name
	}
	if version == "" {
		version = "1.0.0.0"
	}
	if publisher == "" {
		publisher = fmt.Sprintf("CN=%s", proj.Name)
	}
	if outputDir == "" {
		outputDir = filepath.Join(proj.RootDir, constants.DistDir)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Find the built binary - check multiple locations
	binDir := filepath.Join(proj.RootDir, constants.BinDir)
	var binaryPath string

	// Check locations in order of preference:
	// 1. Platform-specific directory: .bin/windows/<name>.exe
	// 2. Legacy location: .bin/<name>.exe
	windowsBinDir := filepath.Join(binDir, "windows")
	platformBinary := filepath.Join(windowsBinDir, proj.Name+".exe")
	legacyBinary := filepath.Join(binDir, proj.Name+".exe")

	if _, err := os.Stat(platformBinary); err == nil {
		binaryPath = platformBinary
		fmt.Println("ℹ️  Found binary in .bin/windows/")
	} else if _, err := os.Stat(legacyBinary); err == nil {
		binaryPath = legacyBinary
		fmt.Println("ℹ️  Found binary in .bin/")
	} else {
		return fmt.Errorf("binary not found in:\n  %s\n  %s\nRun 'utm-dev build windows %s' first",
			platformBinary, legacyBinary, proj.RootDir)
	}

	// Check for assets directory
	assetsDir := filepath.Join(proj.RootDir, "assets")
	if _, err := os.Stat(assetsDir); os.IsNotExist(err) {
		assetsDir = "" // Will generate placeholders
	}

	// Create bundle config
	config := packaging.WindowsBundleConfig{
		Name:                 bundleID,
		Publisher:            publisher,
		PublisherDisplayName: toDisplayName(proj.Name),
		DisplayName:          toDisplayName(proj.Name),
		Description:          fmt.Sprintf("%s application", toDisplayName(proj.Name)),
		Version:              version,
		BinaryPath:           binaryPath,
		OutputDir:            outputDir,
		AssetsDir:            assetsDir,
		CreateMSIX:           createMSIX,
	}

	// Create the bundle
	if err := packaging.CreateWindowsBundle(config); err != nil {
		return fmt.Errorf("failed to create bundle: %w", err)
	}

	fmt.Println()
	fmt.Println("🎯 Next steps:")
	if createMSIX {
		fmt.Println("   1. Test the MSIX:", filepath.Join(outputDir, proj.Name+".msix"))
		fmt.Println("   2. Install: Add-AppxPackage", filepath.Join(outputDir, proj.Name+".msix"))
	} else {
		fmt.Println("   1. Copy .staging directory to Windows machine")
		fmt.Println("   2. Run: utm-dev bundle --create-msix windows", proj.RootDir)
	}
	fmt.Println("   3. Package for distribution:")
	fmt.Printf("      utm-dev package windows %s\n", proj.RootDir)

	return nil
}

// toDisplayName converts a name like "utm-dev" to "Goup Util"
func toDisplayName(name string) string {
	// Simple title case - can be improved
	return name
}

func init() {
	bundleCmd.Flags().String("bundle-id", "", "Bundle identifier (e.g., com.example.myapp)")
	bundleCmd.Flags().String("version", "1.0.0", "Version string")
	bundleCmd.Flags().String("sign", "", "Code signing identity (empty for auto-detect)")
	bundleCmd.Flags().String("output", "", "Output directory (default: .dist/)")
	bundleCmd.Flags().Bool("entitlements", true, "Use entitlements for hardened runtime (macOS)")
	bundleCmd.Flags().String("publisher", "", "Publisher for Windows MSIX (e.g., CN=MyCompany)")
	bundleCmd.Flags().Bool("create-msix", false, "Create MSIX package (Windows-only, requires msix toolkit)")

	// Group for help organization
	bundleCmd.GroupID = "build"

	rootCmd.AddCommand(bundleCmd)
}
