package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/joeblew999/utm-dev/pkg/buildcache"
	"github.com/joeblew999/utm-dev/pkg/cli"
	"github.com/joeblew999/utm-dev/pkg/config"
	"github.com/joeblew999/utm-dev/pkg/constants"
	"github.com/joeblew999/utm-dev/pkg/icons"
	"github.com/joeblew999/utm-dev/pkg/project"
	"github.com/joeblew999/utm-dev/pkg/utils"
	"github.com/spf13/cobra"
)

// BuildOptions contains options for build commands
type BuildOptions struct {
	Force     bool
	CheckOnly bool
	SkipIcons bool
	// New gogio flags (Dec 2025)
	Schemes string // Deep linking URI schemes (e.g., "myapp://,https://example.com")
	Queries string // Android app queries (e.g., "com.google.android.apps.maps")
	SignKey string // Signing key (keystore path for Android, Keychain key name for macOS, or provisioning profile for iOS/macOS)
}

// Global build cache
var globalBuildCache *buildcache.Cache

// getBuildCache returns the global build cache, initializing if needed
func getBuildCache() *buildcache.Cache {
	if globalBuildCache == nil {
		cache, err := buildcache.NewCache(buildcache.GetDefaultCachePath())
		if err != nil {
			// If cache fails, create empty one (won't save)
			cache = &buildcache.Cache{}
		}
		globalBuildCache = cache
	}
	return globalBuildCache
}

var buildCmd = &cobra.Command{
	Use:   "build [platform] [app-directory]",
	Short: "Build Gio applications for different platforms",
	Long: `Build Gio applications for various platforms with deep linking and native features.

Platforms: macos, android, ios, ios-simulator, windows, all

New gogio features (Dec 2025):
  --schemes    Deep linking URI schemes (Android, iOS, macOS, Windows)
  --queries    Android app package queries for intent launching
  --signkey    Signing: keystore (Android), Keychain key (macOS), or provisioning profile (iOS/macOS)

Examples:
  utm-dev build macos ./myapp
  utm-dev build android ./myapp --schemes "myapp://,https://example.com"
  utm-dev build android ./myapp --queries "com.google.android.apps.maps"
  utm-dev build ios ./myapp --signkey /path/to/profile.mobileprovision`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		platform := args[0]
		appDir := args[1]

		// Validate platform
		validPlatforms := []string{"macos", "android", "ios", "ios-simulator", "windows", "linux", "all"}
		if !utils.Contains(validPlatforms, platform) {
			return fmt.Errorf("invalid platform: %s. Valid platforms: %v", platform, validPlatforms)
		}

		// Check for custom output directory flag first
		customOutput, _ := cmd.Flags().GetString("output")

		// Create and validate project with potential custom output
		var proj *project.GioProject
		var err error

		if customOutput != "" {
			// Use custom output directory
			proj, err = project.NewGioProjectWithOutput(appDir, customOutput)
		} else {
			// Use default behavior (artifacts in project directory)
			proj, err = project.NewGioProject(appDir)
		}

		if err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}

		if err := proj.Validate(); err != nil {
			return fmt.Errorf("invalid project: %w", err)
		}

		// Get flags
		skipIcons, _ := cmd.Flags().GetBool("skip-icons")
		force, _ := cmd.Flags().GetBool("force")
		checkOnly, _ := cmd.Flags().GetBool("check")
		schemes, _ := cmd.Flags().GetString("schemes")
		queries, _ := cmd.Flags().GetString("queries")
		signKey, _ := cmd.Flags().GetString("signkey")

		// Create build options
		opts := BuildOptions{
			Force:     force,
			CheckOnly: checkOnly,
			SkipIcons: skipIcons,
			Schemes:   schemes,
			Queries:   queries,
			SignKey:   signKey,
		}

		// Ensure gogio is available (needed for all platforms except linux)
		if platform != "linux" {
			if err := ensureGogio(); err != nil {
				return err
			}
		}

		switch platform {
		case "macos":
			return buildMacOS(proj, platform, opts)
		case "android":
			return buildAndroid(proj, platform, opts)
		case "ios":
			return buildIOS(proj, platform, opts, false)
		case "ios-simulator":
			return buildIOS(proj, "ios-simulator", opts, true)
		case "windows":
			return buildWindows(proj, platform, opts)
		case "linux":
			return buildLinux(proj, platform, opts)
		case "all":
			return buildAll(proj, opts)
		}

		return nil
	},
}

// ensureGogio installs gogio if not available. Idempotent.
// Requires Go to already be installed (we don't install Go for people).
func ensureGogio() error {
	if _, err := exec.LookPath("gogio"); err == nil {
		return nil
	}
	if _, err := exec.LookPath("go"); err != nil {
		return fmt.Errorf("Go not found in PATH — install Go first: https://go.dev/dl/")
	}
	cli.Info("Installing gogio...")
	install := exec.Command("go", "install", "gioui.org/cmd/gogio@latest")
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	if err := install.Run(); err != nil {
		return fmt.Errorf("failed to install gogio: %w", err)
	}
	if _, err := exec.LookPath("gogio"); err != nil {
		return fmt.Errorf("gogio installed but not in PATH — ensure $GOPATH/bin is in your PATH")
	}
	cli.Success("gogio installed")
	return nil
}

func buildMacOS(proj *project.GioProject, platform string, opts BuildOptions) error {
	// Use project's centralized path methods
	platformDir := proj.GetPlatformDir(platform)
	appPath := proj.GetOutputPath(platform)
	cache := getBuildCache()

	// Check if rebuild is needed
	if !opts.Force {
		needsRebuild, reason := cache.NeedsRebuild(proj.Name, platform, proj.RootDir, appPath)

		if opts.CheckOnly {
			if needsRebuild {
				fmt.Printf("Rebuild needed: %s\n", reason)
				os.Exit(1)
			} else {
				fmt.Printf("Up to date: %s\n", appPath)
				os.Exit(0)
			}
		}

		if !needsRebuild {
			fmt.Printf("✓ %s for %s is up-to-date (use --force to rebuild)\n", proj.Name, platform)
			return nil
		}

		fmt.Printf("Rebuilding: %s\n", reason)
	}

	fmt.Printf("Building %s for macOS...\n", proj.Name)

	// Generate icons
	if !opts.SkipIcons {
		if err := generateIcons(proj.RootDir, "macos"); err != nil {
			cache.RecordBuild(proj.Name, platform, proj.RootDir, appPath, false)
			return fmt.Errorf("failed to generate icons: %w", err)
		}
	}

	// Create output directory
	if err := os.MkdirAll(platformDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Remove existing app bundle only if it exists
	if _, err := os.Stat(appPath); err == nil {
		os.RemoveAll(appPath)
	}

	// Build with gogio - run from app directory with GOWORK=off
	// Project paths are already absolute
	iconPath := proj.Paths().SourceIcon

	args := []string{"-target", "macos", "-arch", "arm64", "-icon", iconPath, "-o", appPath}

	// Add deep linking schemes if specified
	if opts.Schemes != "" {
		args = append(args, "-schemes", opts.Schemes)
	}

	// Add signing key / provisioning profile if specified
	if opts.SignKey != "" {
		args = append(args, "-signkey", opts.SignKey)
	}

	args = append(args, ".") // Build current directory
	gogioCmd := exec.Command("gogio", args...)
	gogioCmd.Dir = proj.RootDir // Run from app directory so its go.mod is used
	// Set GOWORK=off to avoid workspace interference with example modules
	gogioCmd.Env = gogioEnv()
	gogioCmd.Stdout = os.Stdout
	gogioCmd.Stderr = os.Stderr

	if err := gogioCmd.Run(); err != nil {
		cache.RecordBuild(proj.Name, platform, proj.RootDir, appPath, false)
		return fmt.Errorf("gogio build failed: %w", err)
	}

	// Record successful build
	cache.RecordBuild(proj.Name, platform, proj.RootDir, appPath, true)

	fmt.Printf("✓ Built %s for macOS: %s\n", proj.Name, appPath)
	return nil
}

func buildAndroid(proj *project.GioProject, platform string, opts BuildOptions) error {
	// Use project's centralized path methods
	platformDir := proj.GetPlatformDir(platform)
	apkPath := proj.GetOutputPath(platform)
	cache := getBuildCache()

	// Check if rebuild is needed
	if !opts.Force {
		needsRebuild, reason := cache.NeedsRebuild(proj.Name, platform, proj.RootDir, apkPath)

		if opts.CheckOnly {
			if needsRebuild {
				fmt.Printf("Rebuild needed: %s\n", reason)
				os.Exit(1)
			} else {
				fmt.Printf("Up to date: %s\n", apkPath)
				os.Exit(0)
			}
		}

		if !needsRebuild {
			fmt.Printf("✓ %s for %s is up-to-date (use --force to rebuild)\n", proj.Name, platform)
			return nil
		}

		fmt.Printf("Rebuilding: %s\n", reason)
	}

	fmt.Printf("Building %s for Android...\n", proj.Name)

	// Generate icons
	if !opts.SkipIcons {
		if err := generateIcons(proj.RootDir, "android"); err != nil {
			cache.RecordBuild(proj.Name, platform, proj.RootDir, apkPath, false)
			return fmt.Errorf("failed to generate icons: %w", err)
		}
	}

	// Create output directory
	if err := os.MkdirAll(platformDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Ensure Android SDK (NDK + platform-tools) — idempotent
	if err := ensureAndroidSDK("gio-android"); err != nil {
		cache.RecordBuild(proj.Name, platform, proj.RootDir, apkPath, false)
		return fmt.Errorf("failed to set up Android SDK: %w", err)
	}
	sdkRoot := config.GetSDKDir()
	ndkPath := filepath.Join(sdkRoot, "ndk-bundle")

	// Set Android environment variables with absolute paths
	env := gogioEnv()
	javaHome := filepath.Join(sdkRoot, "openjdk", "17", "jdk-17.0.11+9", "Contents", "Home")
	env = append(env, "JAVA_HOME="+javaHome)
	env = append(env, "ANDROID_SDK_ROOT="+sdkRoot)
	env = append(env, "ANDROID_HOME="+sdkRoot)
	env = append(env, "ANDROID_NDK_ROOT="+ndkPath)

	// Build with gogio - project paths are already absolute
	// Use minSdk from SDK config (centralized in sdk-android-list.json)
	minSdk := config.GetAndroidMinSdk()
	args := []string{"-target", "android", "-minsdk", minSdk, "-o", apkPath}

	// Add deep linking schemes if specified
	if opts.Schemes != "" {
		args = append(args, "-schemes", opts.Schemes)
	}

	// Add app queries if specified (Android-specific)
	if opts.Queries != "" {
		args = append(args, "-queries", opts.Queries)
	}

	// Add signing key if specified
	if opts.SignKey != "" {
		args = append(args, "-signkey", opts.SignKey)
	}

	args = append(args, ".") // Build current directory
	gogioCmd := exec.Command("gogio", args...)
	gogioCmd.Dir = proj.RootDir // Run from app directory so its go.mod is used
	gogioCmd.Env = env
	gogioCmd.Stdout = os.Stdout
	gogioCmd.Stderr = os.Stderr

	if err := gogioCmd.Run(); err != nil {
		cache.RecordBuild(proj.Name, platform, proj.RootDir, apkPath, false)
		return fmt.Errorf("gogio build failed: %w", err)
	}

	// Record successful build
	cache.RecordBuild(proj.Name, platform, proj.RootDir, apkPath, true)

	fmt.Printf("✓ Built %s for Android: %s\n", proj.Name, apkPath)
	return nil
}

func buildIOS(proj *project.GioProject, platform string, opts BuildOptions, simulator bool) error {
	target := "iOS device"
	if simulator {
		target = "iOS simulator"
	}

	// Use project's centralized path methods
	platformDir := proj.GetPlatformDir(platform)
	appPath := proj.GetOutputPath(platform)
	cache := getBuildCache()

	// Check if rebuild is needed
	if !opts.Force {
		needsRebuild, reason := cache.NeedsRebuild(proj.Name, platform, proj.RootDir, appPath)

		if opts.CheckOnly {
			if needsRebuild {
				fmt.Printf("Rebuild needed: %s\n", reason)
				os.Exit(1)
			} else {
				fmt.Printf("Up to date: %s\n", appPath)
				os.Exit(0)
			}
		}

		if !needsRebuild {
			fmt.Printf("✓ %s for %s is up-to-date (use --force to rebuild)\n", proj.Name, platform)
			return nil
		}

		fmt.Printf("Rebuilding: %s\n", reason)
	}

	fmt.Printf("Building %s for %s...\n", proj.Name, target)

	// Generate icons
	if !opts.SkipIcons {
		if err := generateIcons(proj.RootDir, "ios"); err != nil {
			cache.RecordBuild(proj.Name, platform, proj.RootDir, appPath, false)
			return fmt.Errorf("failed to generate icons: %w", err)
		}
	}

	// Create output directory
	if err := os.MkdirAll(platformDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build with gogio - project paths are already absolute
	// Use minOS from SDK config (centralized in sdk-ios-list.json)
	minOS := config.GetIOSMinOS()

	if simulator && runtime.GOARCH == "arm64" {
		// gogio v0.9.0 hardcodes x86_64 for simulator builds, which causes
		// black screens on Apple Silicon (Metal doesn't work under Rosetta).
		// Build the simulator binary ourselves with the correct arch + SDK.
		if err := buildIOSSimulatorArm64(proj, appPath, minOS, opts); err != nil {
			cache.RecordBuild(proj.Name, platform, proj.RootDir, appPath, false)
			return err
		}
	} else {
		args := []string{"-target", "ios", "-minsdk", minOS, "-o", appPath}

		// Add deep linking schemes if specified
		if opts.Schemes != "" {
			args = append(args, "-schemes", opts.Schemes)
		}

		// Add signing key / provisioning profile if specified
		if opts.SignKey != "" {
			args = append(args, "-signkey", opts.SignKey)
		}

		args = append(args, ".") // Build current directory
		gogioCmd := exec.Command("gogio", args...)
		gogioCmd.Dir = proj.RootDir // Run from app directory so its go.mod is used
		gogioCmd.Env = gogioEnv()
		gogioCmd.Stdout = os.Stdout
		gogioCmd.Stderr = os.Stderr

		if err := gogioCmd.Run(); err != nil {
			cache.RecordBuild(proj.Name, platform, proj.RootDir, appPath, false)
			return fmt.Errorf("gogio build failed: %w", err)
		}
	}

	// Record successful build
	cache.RecordBuild(proj.Name, platform, proj.RootDir, appPath, true)

	fmt.Printf("✓ Built %s for %s: %s\n", proj.Name, target, appPath)
	return nil
}

// buildIOSSimulatorArm64 builds an iOS simulator .app using go build directly,
// bypassing gogio's hardcoded x86_64 arch. gogio v0.9.0 strips arm64 from
// simulator builds, causing black screens on Apple Silicon (Metal needs native arch).
func buildIOSSimulatorArm64(proj *project.GioProject, appPath, minOS string, opts BuildOptions) error {
	// Create .app directory structure
	if err := os.MkdirAll(appPath, 0755); err != nil {
		return fmt.Errorf("failed to create app directory: %w", err)
	}

	// Get the simulator SDK path
	sdkPath, err := exec.Command("xcrun", "--sdk", "iphonesimulator", "--show-sdk-path").Output()
	if err != nil {
		return fmt.Errorf("failed to get simulator SDK path: %w", err)
	}
	sdk := strings.TrimSpace(string(sdkPath))

	// Get clang path from simulator SDK
	clangPath, err := exec.Command("xcrun", "--sdk", "iphonesimulator", "-f", "clang").Output()
	if err != nil {
		return fmt.Errorf("failed to find clang: %w", err)
	}
	clang := strings.TrimSpace(string(clangPath))

	// Use the same binary name convention as gogio: uppercase first letter
	binName := strings.ToUpper(proj.Name[:1]) + proj.Name[1:]
	binPath := filepath.Join(appPath, binName)

	// Match gogio's exact compiler flags for iOS builds:
	// -fembed-bitcode, -fobjc-arc, -arch arm64, simulator SDK, min version
	// Plus -lresolv in ldflags (required by Gio's network code)
	cflags := fmt.Sprintf("-fembed-bitcode -fobjc-arc -arch arm64 -isysroot %s -mios-simulator-version-min=%s", sdk, minOS)
	ldflags := fmt.Sprintf("-lresolv -fembed-bitcode -fobjc-arc -arch arm64 -isysroot %s -mios-simulator-version-min=%s", sdk, minOS)

	env := gogioEnv()
	env = append(env,
		"GOOS=ios",
		"GOARCH=arm64",
		"CGO_ENABLED=1",
		"CC="+clang,
		"CGO_CFLAGS="+cflags,
		"CGO_LDFLAGS="+ldflags,
	)

	// Gio's shader module (gioui.org/shader) selects Metal shader variants at
	// init time based on runtime.GOARCH: "amd64" → simulator shaders, anything
	// else → device shaders. When we build GOARCH=arm64 for the simulator, the
	// wrong (device) shaders are loaded, causing "Compiler failed to build
	// request" at runtime.
	//
	// Fix: vendor dependencies, patch the vendored shaders.go to always select
	// simulator shaders, then build with -mod=vendor.
	if err := patchVendorForSimulatorShaders(proj.RootDir, env); err != nil {
		return fmt.Errorf("failed to patch shaders for simulator: %w", err)
	}
	defer os.RemoveAll(filepath.Join(proj.RootDir, "vendor")) // Clean up vendor dir

	buildCmd := exec.Command("go", "build", "-mod=vendor", "-ldflags=-s -w", "-o", binPath, ".")
	buildCmd.Dir = proj.RootDir
	buildCmd.Env = env
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("go build for iOS simulator failed: %w", err)
	}

	// Generate Info.plist
	bundleID := "localhost." + strings.ReplaceAll(proj.Name, "-", "_")
	plist := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleExecutable</key>
	<string>%s</string>
	<key>CFBundleIdentifier</key>
	<string>%s</string>
	<key>CFBundleName</key>
	<string>%s</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>1.0.0</string>
	<key>CFBundleVersion</key>
	<string>1</string>
	<key>CFBundleSupportedPlatforms</key>
	<array><string>iPhoneSimulator</string></array>
	<key>MinimumOSVersion</key>
	<string>%s</string>
	<key>CFBundleDevelopmentRegion</key>
	<string>en</string>
	<key>UIRequiredDeviceCapabilities</key>
	<array><string>arm64</string></array>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
</dict>
</plist>`, binName, bundleID, proj.Name, minOS)

	plistPath := filepath.Join(appPath, "Info.plist")
	if err := os.WriteFile(plistPath, []byte(plist), 0644); err != nil {
		return fmt.Errorf("failed to write Info.plist: %w", err)
	}

	// Copy icon assets if they exist
	assetsDir := filepath.Join(proj.RootDir, constants.BuildDir, "Assets.xcassets")
	if _, err := os.Stat(assetsDir); err == nil {
		// Compile assets with actool
		actoolCmd := exec.Command("xcrun", "actool", "--compile", appPath,
			"--platform", "iphonesimulator",
			"--minimum-deployment-target", minOS,
			assetsDir)
		actoolCmd.Stdout = os.Stdout
		actoolCmd.Stderr = os.Stderr
		actoolCmd.Run() // Non-fatal if this fails
	}

	return nil
}

// patchVendorForSimulatorShaders vendors the project's dependencies and patches
// gioui.org/shader's shaders.go so that arm64 iOS builds use simulator Metal
// shaders instead of device shaders.
func patchVendorForSimulatorShaders(projectDir string, env []string) error {
	// Vendor dependencies
	vendorCmd := exec.Command("go", "mod", "vendor")
	vendorCmd.Dir = projectDir
	vendorCmd.Env = env
	vendorCmd.Stdout = os.Stdout
	vendorCmd.Stderr = os.Stderr
	if err := vendorCmd.Run(); err != nil {
		return fmt.Errorf("go mod vendor failed: %w", err)
	}

	// Patch the vendored shaders.go
	shadersPath := filepath.Join(projectDir, "vendor", "gioui.org", "shader", "gio", "shaders.go")
	src, err := os.ReadFile(shadersPath)
	if err != nil {
		return fmt.Errorf("failed to read vendored shaders.go: %w", err)
	}

	// Replace the GOARCH check so arm64 simulator builds use simulator shaders.
	// Original: `if runtime.GOARCH == "amd64" {` → selects simulator shaders only for x86_64
	// Patched:  `if true {` → always selects simulator shaders (correct for simulator builds)
	patched := strings.ReplaceAll(string(src),
		`runtime.GOARCH == "amd64"`,
		`true /* patched: arm64 simulator needs simulator shaders */`)

	if err := os.WriteFile(shadersPath, []byte(patched), 0644); err != nil {
		return fmt.Errorf("failed to write patched shaders.go: %w", err)
	}

	return nil
}

func buildWindows(proj *project.GioProject, platform string, opts BuildOptions) error {
	// Use project's centralized path methods
	platformDir := proj.GetPlatformDir(platform)
	exePath := proj.GetOutputPath(platform)
	cache := getBuildCache()

	// Check if rebuild is needed
	if !opts.Force {
		needsRebuild, reason := cache.NeedsRebuild(proj.Name, platform, proj.RootDir, exePath)

		if opts.CheckOnly {
			if needsRebuild {
				fmt.Printf("Rebuild needed: %s\n", reason)
				os.Exit(1)
			} else {
				fmt.Printf("Up to date: %s\n", exePath)
				os.Exit(0)
			}
		}

		if !needsRebuild {
			fmt.Printf("✓ %s for %s is up-to-date (use --force to rebuild)\n", proj.Name, platform)
			return nil
		}

		fmt.Printf("Rebuilding: %s\n", reason)
	}

	fmt.Printf("Building %s for Windows...\n", proj.Name)

	// Generate icons
	if !opts.SkipIcons {
		if err := generateIcons(proj.RootDir, "windows"); err != nil {
			cache.RecordBuild(proj.Name, platform, proj.RootDir, exePath, false)
			return fmt.Errorf("failed to generate icons: %w", err)
		}
	}

	// Create output directory
	if err := os.MkdirAll(platformDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Set Windows environment
	env := gogioEnv()
	env = append(env, "GOOS=windows")
	env = append(env, "GOARCH=amd64") // Use amd64 for broader Windows compatibility

	// Build with gogio - project paths are already absolute
	iconPath := proj.Paths().SourceIcon

	gogioCmd := exec.Command("gogio", "-o", exePath, "-target", "windows", "-icon", iconPath, ".")
	gogioCmd.Dir = proj.RootDir // Run from app directory so its go.mod is used
	gogioCmd.Env = env
	gogioCmd.Stdout = os.Stdout
	gogioCmd.Stderr = os.Stderr

	if err := gogioCmd.Run(); err != nil {
		cache.RecordBuild(proj.Name, platform, proj.RootDir, exePath, false)
		return fmt.Errorf("gogio build failed: %w", err)
	}

	// Record successful build
	cache.RecordBuild(proj.Name, platform, proj.RootDir, exePath, true)

	fmt.Printf("✓ Built %s for Windows: %s\n", proj.Name, exePath)
	return nil
}

func buildLinux(proj *project.GioProject, platform string, opts BuildOptions) error {
	// Use project's centralized path methods
	platformDir := proj.GetPlatformDir(platform)
	binPath := proj.GetOutputPath(platform)
	cache := getBuildCache()

	// Check if rebuild is needed
	if !opts.Force {
		needsRebuild, reason := cache.NeedsRebuild(proj.Name, platform, proj.RootDir, binPath)

		if opts.CheckOnly {
			if needsRebuild {
				fmt.Printf("Rebuild needed: %s\n", reason)
				os.Exit(1)
			} else {
				fmt.Printf("Up to date: %s\n", binPath)
				os.Exit(0)
			}
		}

		if !needsRebuild {
			fmt.Printf("✓ %s for %s is up-to-date (use --force to rebuild)\n", proj.Name, platform)
			return nil
		}

		fmt.Printf("Rebuilding: %s\n", reason)
	}

	fmt.Printf("Building %s for Linux...\n", proj.Name)

	// Create output directory
	if err := os.MkdirAll(platformDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build with go build (Linux doesn't need gogio for basic builds)
	// For webview apps, gogio would be needed but Linux webview support is limited
	env := os.Environ()
	env = append(env, "GOWORK=off") // Avoid workspace interference with example modules
	env = append(env, "GOOS=linux")
	env = append(env, "GOARCH=amd64")
	env = append(env, "CGO_ENABLED=1")

	buildCmd := exec.Command("go", "build", "-o", binPath, ".")
	buildCmd.Env = env
	buildCmd.Dir = proj.RootDir
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		cache.RecordBuild(proj.Name, platform, proj.RootDir, binPath, false)
		return fmt.Errorf("go build failed: %w", err)
	}

	// Record successful build
	cache.RecordBuild(proj.Name, platform, proj.RootDir, binPath, true)

	fmt.Printf("✓ Built %s for Linux: %s\n", proj.Name, binPath)
	return nil
}

func buildAll(proj *project.GioProject, opts BuildOptions) error {
	fmt.Printf("Building %s for all platforms...\n", proj.Name)

	platforms := []string{"macos", "android", "ios-simulator", "windows"}

	for _, platform := range platforms {
		fmt.Printf("\n--- Building for %s ---\n", platform)
		switch platform {
		case "macos":
			if err := buildMacOS(proj, platform, opts); err != nil {
				fmt.Printf("❌ Failed to build for %s: %v\n", platform, err)
			}
		case "android":
			if err := buildAndroid(proj, platform, opts); err != nil {
				fmt.Printf("❌ Failed to build for %s: %v\n", platform, err)
			}
		case "ios-simulator":
			if err := buildIOS(proj, platform, opts, true); err != nil {
				fmt.Printf("❌ Failed to build for %s: %v\n", platform, err)
			}
		case "windows":
			if err := buildWindows(proj, platform, opts); err != nil {
				fmt.Printf("❌ Failed to build for %s: %v\n", platform, err)
			}
		}
	}

	fmt.Printf("\n✓ Build complete for all platforms\n")
	return nil
}

func generateIcons(appDir, platform string) error {
	// Ensure source icon exists
	sourceIconPath, err := icons.EnsureSourceIcon(appDir)
	if err != nil {
		return err
	}

	// Generate platform-specific icons
	var outputPath string
	switch platform {
	case "android":
		outputPath = filepath.Join(appDir, constants.BuildDir)
	case "ios", "macos":
		outputPath = filepath.Join(appDir, constants.BuildDir, "Assets.xcassets")
	case "windows":
		outputPath = filepath.Join(appDir, constants.BuildDir)
		platform = "windows-msix" // Use the correct platform name
	default:
		return nil // Skip icon generation for unknown platforms
	}

	fmt.Printf("Generating %s icons...\n", platform)
	return icons.Generate(icons.Config{
		InputPath:  sourceIconPath,
		OutputPath: outputPath,
		Platform:   platform,
	})
}

// Remove the old generateTestIcon function since it's now in the icons package
// contains() moved to pkg/utils/slice.go

// gogioEnv returns a base environment for gogio subprocesses.
// Disables Go workspace mode and trusts mise config in example dirs
// so gogio builds don't fail with "mise.toml is not trusted" errors.
func gogioEnv() []string {
	env := os.Environ()
	env = append(env, "GOWORK=off")
	// Trust all mise configs under the current project so gogio
	// can run in example directories without manual `mise trust`.
	if cwd, err := os.Getwd(); err == nil {
		env = append(env, "MISE_TRUSTED_CONFIG_PATHS="+cwd)
	}
	return env
}

func init() {
	buildCmd.Flags().BoolVar(&skipIcons, "skip-icons", false, "Skip icon generation")
	buildCmd.Flags().String("output", "", "Custom output directory for build artifacts")
	buildCmd.Flags().Bool("force", false, "Force rebuild even if up-to-date")
	buildCmd.Flags().Bool("check", false, "Check if rebuild needed (exit 0=no, 1=yes)")

	// New gogio flags (Dec 2025)
	buildCmd.Flags().String("schemes", "", "Deep linking URI schemes (comma-separated, e.g., 'myapp://,https://example.com')")
	buildCmd.Flags().String("queries", "", "Android app package queries (comma-separated, e.g., 'com.google.android.apps.maps')")
	buildCmd.Flags().String("signkey", "", "Signing key: keystore path (Android), Keychain key name (macOS), or provisioning profile (iOS/macOS)")

	// Tab completion for platform argument
	buildCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return getPlatformCompletion(cmd, args, toComplete)
		}
		if len(args) == 1 {
			return getExampleCompletion(cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	gioCmd.AddCommand(buildCmd)
}

var skipIcons bool
