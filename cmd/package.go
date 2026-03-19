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

var packageCmd = &cobra.Command{
	Use:   "package [platform] [app-directory]",
	Short: "Package built applications for distribution",
	Long:  "Create distribution packages from built applications. Takes apps from .bin/ and creates packages in .dist/",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		platform := args[0]
		appDir := args[1]

		// Validate platform
		validPlatforms := []string{"macos", "android", "ios", "windows"}
		if !utils.Contains(validPlatforms, platform) {
			return fmt.Errorf("invalid platform: %s. Valid platforms: %v", platform, validPlatforms)
		}

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
			return packageMacOS(proj.RootDir, proj.Name)
		case "android":
			return packageAndroid(proj.RootDir, proj.Name)
		case "ios":
			return packageIOS(proj.RootDir, proj.Name)
		case "windows":
			return packageWindows(proj.RootDir, proj.Name)
		}

		return nil
	},
}

func packageMacOS(appDir, appName string) error {
	fmt.Printf("Packaging %s for macOS distribution...\n", appName)

	binDir := filepath.Join(appDir, constants.BinDir)
	distDir := filepath.Join(appDir, constants.DistDir)

	// Ensure dist directory exists
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return fmt.Errorf("failed to create dist directory: %w", err)
	}

	// Check if app exists
	appPath := filepath.Join(binDir, appName+".app")
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		return fmt.Errorf("app not found: %s. Run 'utm-dev build macos %s' first", appPath, appDir)
	}

	// Create tar.gz package using packaging library
	packagePath := filepath.Join(distDir, appName+"-macos.tar.gz")
	if err := packaging.CreateArchive(appPath, packagePath, packaging.TarGz); err != nil {
		return fmt.Errorf("failed to create package: %w", err)
	}

	fmt.Printf("✓ Packaged %s for macOS: %s\n", appName, packagePath)
	return nil
}

func packageAndroid(appDir, appName string) error {
	fmt.Printf("Packaging %s for Android distribution...\n", appName)

	binDir := filepath.Join(appDir, constants.BinDir)
	distDir := filepath.Join(appDir, constants.DistDir)

	// Ensure dist directory exists
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return fmt.Errorf("failed to create dist directory: %w", err)
	}

	// Check if APK exists
	apkPath := filepath.Join(binDir, appName+".apk")
	if _, err := os.Stat(apkPath); os.IsNotExist(err) {
		return fmt.Errorf("APK not found: %s. Run 'utm-dev build android %s' first", apkPath, appDir)
	}

	// Copy APK to dist with versioned name using packaging library
	packagePath := filepath.Join(distDir, appName+"-android.apk")
	if err := packaging.CopyFile(apkPath, packagePath); err != nil {
		return fmt.Errorf("failed to create package: %w", err)
	}

	fmt.Printf("✓ Packaged %s for Android: %s\n", appName, packagePath)
	return nil
}

func packageIOS(appDir, appName string) error {
	fmt.Printf("Packaging %s for iOS distribution...\n", appName)

	binDir := filepath.Join(appDir, constants.BinDir)
	distDir := filepath.Join(appDir, constants.DistDir)

	// Ensure dist directory exists
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return fmt.Errorf("failed to create dist directory: %w", err)
	}

	// Check if app exists
	appPath := filepath.Join(binDir, appName+".app")
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		return fmt.Errorf("app not found: %s. Run 'utm-dev build ios %s' first", appPath, appDir)
	}

	// Create tar.gz package using packaging library
	packagePath := filepath.Join(distDir, appName+"-ios.tar.gz")
	if err := packaging.CreateArchive(appPath, packagePath, packaging.TarGz); err != nil {
		return fmt.Errorf("failed to create package: %w", err)
	}

	fmt.Printf("✓ Packaged %s for iOS: %s\n", appName, packagePath)
	return nil
}

func packageWindows(appDir, appName string) error {
	fmt.Printf("Packaging %s for Windows distribution...\n", appName)

	binDir := filepath.Join(appDir, constants.BinDir)
	distDir := filepath.Join(appDir, constants.DistDir)

	// Ensure dist directory exists
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return fmt.Errorf("failed to create dist directory: %w", err)
	}

	// Check if exe exists
	exePath := filepath.Join(binDir, appName+".exe")
	if _, err := os.Stat(exePath); os.IsNotExist(err) {
		return fmt.Errorf("executable not found: %s. Run 'utm-dev build windows %s' first", exePath, appDir)
	}

	// Create zip package using packaging library
	packagePath := filepath.Join(distDir, appName+"-windows.zip")
	if err := packaging.CreateArchive(exePath, packagePath, packaging.Zip); err != nil {
		return fmt.Errorf("failed to create package: %w", err)
	}

	fmt.Printf("✓ Packaged %s for Windows: %s\n", appName, packagePath)
	return nil
}

func init() {
	// Group for help organization
	packageCmd.GroupID = "build"

	rootCmd.AddCommand(packageCmd)
}
