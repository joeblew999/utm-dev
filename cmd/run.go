package cmd

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/joeblew999/utm-dev/pkg/adb"
	"github.com/joeblew999/utm-dev/pkg/project"
	"github.com/joeblew999/utm-dev/pkg/simctl"
	"github.com/joeblew999/utm-dev/pkg/utils"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [platform] [app-directory]",
	Short: "Build and run a Gio application",
	Long: `Build and run a Gio application for the specified platform.

This command builds the app (if needed) and launches it. The app is automatically
opened using the platform-specific path, so you don't need to know where it's built.

Platforms: macos, android, ios-simulator

For Windows, use: utm-dev utm run "Windows 11" <app-dir>

Examples:
  utm-dev run macos ./myapp
  utm-dev run android examples/hybrid-dashboard
  utm-dev run ios-simulator examples/hybrid-dashboard`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		platform := args[0]
		appDir := args[1]

		// Validate platform - support platforms we can run locally
		validPlatforms := []string{"macos", "android"}
		if runtime.GOOS == "darwin" {
			validPlatforms = append(validPlatforms, "ios-simulator")
		}

		if !utils.Contains(validPlatforms, platform) {
			return fmt.Errorf("cannot run %s apps on %s. Valid platforms: %v", platform, runtime.GOOS, validPlatforms)
		}

		// Create and validate project
		proj, err := project.NewGioProject(appDir)
		if err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}

		if err := proj.Validate(); err != nil {
			return fmt.Errorf("invalid project: %w", err)
		}

		// Get build flags
		force, _ := cmd.Flags().GetBool("force")
		skipIcons, _ := cmd.Flags().GetBool("skip-icons")
		schemes, _ := cmd.Flags().GetString("schemes")

		// Build the app
		opts := BuildOptions{
			Force:     force,
			SkipIcons: skipIcons,
			Schemes:   schemes,
		}

		switch platform {
		case "macos":
			if err := buildMacOS(proj, platform, opts); err != nil {
				return fmt.Errorf("build failed: %w", err)
			}
		case "android":
			if err := buildAndroid(proj, platform, opts); err != nil {
				return fmt.Errorf("build failed: %w", err)
			}
		case "ios-simulator":
			if err := buildIOS(proj, platform, opts, true); err != nil {
				return fmt.Errorf("build failed: %w", err)
			}
		}

		// Launch the app
		appPath := proj.GetOutputPath(platform)
		fmt.Printf("Launching %s...\n", appPath)

		switch platform {
		case "macos":
			return launchMacOSApp(appPath)
		case "android":
			return launchAndroidApp(appPath, proj.Name)
		case "ios-simulator":
			return launchIOSSimulator(appPath, proj.Name)
		}

		return nil
	},
}

func launchAndroidApp(apkPath, appName string) error {
	// Ensure adb is installed (idempotent)
	if err := ensureAndroidSDK(); err != nil {
		return err
	}
	client := adb.New()

	// Ensure a device is connected
	if !client.HasDevice() {
		return fmt.Errorf("no Android device connected. Start an emulator with: utm-dev android emulator start <avd-name>")
	}

	// Install the APK
	fmt.Printf("Installing %s...\n", apkPath)
	if err := client.Install(apkPath); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	// Launch the app — gogio uses "localhost.<appname>" as package by default
	pkg := "localhost." + appName
	fmt.Printf("Launching %s...\n", pkg)
	if err := client.Launch(pkg); err != nil {
		return fmt.Errorf("launch failed: %w", err)
	}

	fmt.Printf("✓ App running on device\n")
	return nil
}

func launchMacOSApp(appPath string) error {
	cmd := exec.Command("open", appPath)
	return cmd.Run()
}

func launchIOSSimulator(appPath, appName string) error {
	client := simctl.New()
	if !client.Available() {
		return fmt.Errorf("xcrun simctl not available\nInstall Xcode command line tools: xcode-select --install")
	}

	// Ensure a simulator is booted
	if !client.HasBooted() {
		// Try to open Simulator.app which boots the default device
		fmt.Println("No simulator booted, opening Simulator.app...")
		if err := client.OpenSimulatorApp(); err != nil {
			return fmt.Errorf("could not open Simulator app: %w\nBoot a simulator with: utm-dev ios boot \"iPhone 16\"", err)
		}
		fmt.Println("Waiting for simulator to boot...")
	}

	// Install the app
	fmt.Printf("Installing %s...\n", appPath)
	if err := client.Install(appPath); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}

	// Launch the app — gogio uses "localhost.<appname>" as bundle ID by default
	bundleID := "localhost." + appName
	fmt.Printf("Launching %s...\n", bundleID)
	if err := client.Launch(bundleID); err != nil {
		return fmt.Errorf("launch failed: %w", err)
	}

	fmt.Printf("✓ App running on simulator\n")
	return nil
}

func init() {
	runCmd.Flags().Bool("force", false, "Force rebuild even if up-to-date")
	runCmd.Flags().Bool("skip-icons", false, "Skip icon generation")
	runCmd.Flags().String("schemes", "", "Deep linking URI schemes")

	// Group for help organization
	runCmd.GroupID = "build"

	rootCmd.AddCommand(runCmd)
}
