package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/joeblew999/utm-dev/pkg/adb"
	"github.com/joeblew999/utm-dev/pkg/cli"
	"github.com/joeblew999/utm-dev/pkg/config"
	"github.com/joeblew999/utm-dev/pkg/simctl"
	"github.com/joeblew999/utm-dev/pkg/utils"
	"github.com/joeblew999/utm-dev/pkg/utm"
	"github.com/spf13/cobra"
)

// isTauriProject checks for src-tauri/tauri.conf.json in the given directory.
func isTauriProject(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "src-tauri", "tauri.conf.json"))
	return err == nil
}

// ensureRust installs Rust via rustup if not present. Idempotent.
func ensureRust() error {
	if _, err := exec.LookPath("cargo"); err == nil {
		return nil
	}
	cli.Info("Installing Rust via rustup...")
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		// Windows: download and run rustup-init.exe
		cmd = exec.Command("powershell", "-Command",
			`Invoke-WebRequest -Uri "https://win.rustup.rs/aarch64" -OutFile "$env:TEMP\rustup-init.exe"; & "$env:TEMP\rustup-init.exe" -y`)
	} else {
		cmd = exec.Command("sh", "-c", "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install Rust: %w", err)
	}
	cli.Success("Rust installed")
	return nil
}

// ensureCargoTauri ensures Rust + cargo-tauri are installed. Idempotent.
func ensureCargoTauri() error {
	if err := ensureRust(); err != nil {
		return err
	}
	cmd := exec.Command("cargo", "tauri", "--version")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err == nil {
		return nil
	}
	cli.Info("Installing cargo-tauri...")
	install := exec.Command("cargo", "install", "tauri-cli")
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	if err := install.Run(); err != nil {
		return fmt.Errorf("failed to install cargo-tauri: %w", err)
	}
	cli.Success("cargo-tauri installed")
	return nil
}

// ensureAndroidSDK installs the Android SDK components needed by the caller.
// Uses the SDK catalog (sdk-android-list.json) and installSdk() for all installs.
// setupName should be "gio-android" or "tauri-android" from the catalog's setups.
// Idempotent — installSdk/installWithSdkManager skip already-installed items.
func ensureAndroidSDK(setupName string) error {
	cache, err := utils.NewCacheWithDirectories()
	if err != nil {
		return fmt.Errorf("failed to create cache: %w", err)
	}

	// Load the setup group from the catalog
	sdkNames, err := utils.FindSetup(setupName)
	if err != nil {
		return fmt.Errorf("unknown setup %q: %w", setupName, err)
	}

	// Install each SDK in the setup group
	for _, sdkName := range sdkNames {
		if err := installSdk(sdkName, cache); err != nil {
			return fmt.Errorf("failed to install %s: %w", sdkName, err)
		}
	}

	// cmdline-tools compatibility symlinks
	// Our direct-download cmdline-tools land at cmdline-tools/11.0/cmdline-tools/
	// but Tauri expects cmdline-tools/bin/ and cmdline-tools/latest/
	ensureCmdlineToolsSymlinks()

	return nil
}

// ensureCmdlineToolsSymlinks creates symlinks so external tools (Tauri, Android Studio)
// can find sdkmanager at the standard paths. Idempotent.
func ensureCmdlineToolsSymlinks() {
	sdkRoot := config.GetSDKDir()
	cmdToolsSrc := filepath.Join(sdkRoot, "cmdline-tools", "11.0", "cmdline-tools")
	if _, err := os.Stat(cmdToolsSrc); err != nil {
		return
	}
	// cmdline-tools/latest → actual cmdline-tools
	latestLink := filepath.Join(sdkRoot, "cmdline-tools", "latest")
	if _, err := os.Lstat(latestLink); os.IsNotExist(err) {
		os.Symlink(cmdToolsSrc, latestLink)
	}
	// cmdline-tools/bin → actual bin (Tauri checks this path)
	binLink := filepath.Join(sdkRoot, "cmdline-tools", "bin")
	if _, err := os.Lstat(binLink); os.IsNotExist(err) {
		os.Symlink(filepath.Join(cmdToolsSrc, "bin"), binLink)
	}
}

// ensureCocoapods installs CocoaPods if not present. Idempotent.
// Uses mise (since utm-dev is distributed via mise, user has it).
func ensureCocoapods() error {
	if _, err := exec.LookPath("pod"); err == nil {
		return nil
	}
	cli.Info("Installing CocoaPods via mise...")
	cmd := exec.Command("mise", "use", "--global", "cocoapods")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install cocoapods via mise: %w", err)
	}
	// Reshim so `pod` is available in PATH immediately
	reshim := exec.Command("mise", "reshim")
	reshim.Run()
	cli.Success("CocoaPods installed")
	return nil
}

// ensureTauriMobileInit runs `tauri <platform> init` if the platform target
// hasn't been set up yet. Idempotent.
func ensureTauriMobileInit(dir string, platform string) error {
	var checkPath string
	switch platform {
	case "ios":
		checkPath = filepath.Join(dir, "src-tauri", "gen", "apple")
	case "android":
		checkPath = filepath.Join(dir, "src-tauri", "gen", "android")
	default:
		return nil // desktop platforms don't need init
	}

	if _, err := os.Stat(checkPath); err == nil {
		return nil // already initialized
	}

	cli.Info("Tauri %s target not initialized — running init...", platform)

	if platform == "ios" {
		if err := ensureCocoapods(); err != nil {
			return err
		}
	}

	if err := runCargoTauri(dir, platform, "init"); err != nil {
		return fmt.Errorf("tauri %s init failed: %w", platform, err)
	}
	cli.Success("Tauri %s target initialized", platform)
	return nil
}

// ensureIOSSimulator ensures at least one iOS simulator is booted. Idempotent.
func ensureIOSSimulator() (*simctl.Client, error) {
	client := simctl.New()
	if !client.Available() {
		return nil, fmt.Errorf("xcrun simctl not available — Xcode must be installed from App Store")
	}
	if client.HasBooted() {
		return client, nil
	}

	// Find an available iPhone simulator and boot it
	cli.Info("No simulator booted, finding one to boot...")
	devices, err := client.Devices()
	if err != nil {
		return nil, fmt.Errorf("failed to list simulators: %w", err)
	}

	// Prefer iPhone 16 Pro > iPhone 16 > any iPhone > any device
	var best *simctl.Device
	for i := range devices {
		d := &devices[i]
		if d.Name == "iPhone 16 Pro" {
			best = d
			break
		}
		if d.Name == "iPhone 16" && (best == nil || best.Name != "iPhone 16") {
			best = d
		}
		if best == nil && len(d.Name) >= 6 && d.Name[:6] == "iPhone" {
			best = d
		}
		if best == nil {
			best = d
		}
	}
	if best == nil {
		return nil, fmt.Errorf("no simulators available — open Xcode to download a simulator runtime")
	}

	cli.Info("Booting %s (%s)...", best.Name, best.Runtime)
	if err := client.Boot(best.UDID); err != nil {
		return nil, fmt.Errorf("failed to boot simulator: %w", err)
	}
	// Open the Simulator.app so the user can see it
	_ = client.OpenSimulatorApp()
	// Give it a moment to finish booting
	time.Sleep(3 * time.Second)
	cli.Success("Simulator %s booted", best.Name)
	return client, nil
}

// ensureVM ensures UTM is installed, the VM exists, and is started. Idempotent.
func ensureVM(vmName string) error {
	// Ensure UTM is installed
	if err := utm.InstallUTM(false); err != nil {
		return fmt.Errorf("failed to install UTM: %w", err)
	}

	// Check if VM exists
	status, err := utm.GetVMStatus(vmName)
	if err != nil {
		// VM doesn't exist — install the box
		boxKey := "windows-11" // default
		if vmName == "Ubuntu" {
			boxKey = "ubuntu"
		}
		cli.Info("VM '%s' not found, installing...", vmName)
		if err := utm.InstallBox(boxKey, false); err != nil {
			return fmt.Errorf("failed to install VM '%s': %w", vmName, err)
		}
		status, err = utm.GetVMStatus(vmName)
		if err != nil {
			return fmt.Errorf("VM '%s' still not found after install: %w", vmName, err)
		}
	}

	// Ensure VM is started
	if status != "started" {
		cli.Info("Starting VM '%s'...", vmName)
		if err := utm.StartVM(vmName); err != nil {
			return fmt.Errorf("failed to start VM: %w", err)
		}
		cli.Info("Waiting for VM to boot...")
		if err := utm.WaitForWindows("localhost", 5*time.Minute); err != nil {
			cli.Warn("VM may still be booting (WinRM not responding) — continuing...")
		}
		cli.Success("VM '%s' running", vmName)
	}

	return nil
}

// runCargoTauri runs a cargo tauri command in the given directory.
func runCargoTauri(dir string, args ...string) error {
	fullArgs := append([]string{"tauri"}, args...)
	cmd := exec.Command("cargo", fullArgs...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Set Android SDK env vars so Tauri CLI + Gradle can find our managed SDK
	sdkRoot := config.GetSDKDir()
	ndkPath := filepath.Join(sdkRoot, "ndk", "27.2.12479018")
	javaHome := filepath.Join(sdkRoot, "openjdk", "17", "jdk-17.0.11+9", "Contents", "Home")
	env := os.Environ()
	env = append(env, "ANDROID_HOME="+sdkRoot)
	env = append(env, "ANDROID_SDK_ROOT="+sdkRoot)
	env = append(env, "NDK_HOME="+ndkPath)
	env = append(env, "JAVA_HOME="+javaHome)
	cmd.Env = env

	// Write local.properties so Gradle finds the SDK without relying on env vars.
	// Gradle's daemon may have been started before our env vars were set.
	ensureLocalProperties(dir, sdkRoot, ndkPath)

	return cmd.Run()
}

// ensureLocalProperties writes local.properties in the Tauri Android gen dir
// so Gradle can find sdk.dir and ndk.dir. Idempotent — overwrites each time
// to ensure paths stay current.
func ensureLocalProperties(dir, sdkRoot, ndkPath string) {
	androidDir := filepath.Join(dir, "src-tauri", "gen", "android")
	if _, err := os.Stat(androidDir); err != nil {
		return // Android target not initialized yet
	}
	content := fmt.Sprintf("# Auto-generated by utm-dev — do not edit\nsdk.dir=%s\nndk.dir=%s\n", sdkRoot, ndkPath)
	propsPath := filepath.Join(androidDir, "local.properties")
	os.WriteFile(propsPath, []byte(content), 0644)
}

// --- Root tauri command ---

var tauriCmd = &cobra.Command{
	Use:   "tauri",
	Short: "Build and run Tauri applications",
	Long: `Build and run Tauri v2 applications across all platforms.

Desktop (macOS on host, Windows/Linux via UTM VM):
  utm-dev tauri dev <dir>                Start dev mode
  utm-dev tauri build macos <dir>        Build macOS .app/.dmg
  utm-dev tauri build windows <dir>      Build in Windows UTM VM

Mobile (on host Mac):
  utm-dev tauri init ios <dir>           One-time iOS setup
  utm-dev tauri build ios <dir>          Build for iOS
  utm-dev tauri run ios <dir>            Run on iOS simulator
  utm-dev tauri build android <dir>      Build for Android
  utm-dev tauri run android <dir>        Run on Android emulator

Icons:
  utm-dev tauri icons <dir>              Generate all platform icons`,
}

// --- tauri setup ---

var tauriSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Install all prerequisites for Tauri development",
	Long: `Install everything needed for Tauri cross-platform development.

This is idempotent — run it as many times as you want. It only installs
what's missing.

Sets up: Rust, cargo-tauri, Android NDK + platform-tools,
UTM + Windows 11 VM, and checks for Xcode.

Examples:
  utm-dev tauri setup                # Install everything
  utm-dev tauri setup --skip-vm      # Skip UTM VM setup (large download)
  utm-dev tauri setup --mobile-only  # Only install mobile deps`,
	RunE: func(cmd *cobra.Command, args []string) error {
		skipVM, _ := cmd.Flags().GetBool("skip-vm")
		mobileOnly, _ := cmd.Flags().GetBool("mobile-only")

		cli.Info("Setting up Tauri development environment...")

		// Step 1: Rust + cargo-tauri
		cli.Info("[1/4] Ensuring Rust + cargo-tauri...")
		if err := ensureCargoTauri(); err != nil {
			return err
		}
		cli.Success("Rust + cargo-tauri ready")

		// Step 2: Android SDK + NDK + platform-tools
		cli.Info("[2/4] Ensuring Android SDK...")
		if err := ensureAndroidSDK("tauri-android"); err != nil {
			cli.Warn("Android SDK setup issue: %v", err)
		} else {
			cli.Success("Android SDK ready")
		}

		// Step 3: Xcode (can't auto-install, just check)
		cli.Info("[3/4] Checking Xcode...")
		if _, err := exec.LookPath("xcrun"); err != nil {
			cli.Warn("Xcode not found — install from App Store for iOS builds")
		} else {
			cli.Success("Xcode available")
		}

		// Step 4: UTM + Windows VM
		if !skipVM && !mobileOnly {
			cli.Info("[4/4] Ensuring UTM + Windows VM...")
			if err := ensureVM("Windows 11"); err != nil {
				cli.Warn("VM setup issue: %v", err)
			} else {
				cli.Success("UTM + Windows 11 VM ready")
			}
		} else {
			cli.Info("[4/4] Skipping UTM VM setup")
		}

		cli.Success("Tauri development environment ready!")
		return nil
	},
}

// --- tauri dev ---

var tauriDevCmd = &cobra.Command{
	Use:   "dev <app-directory>",
	Short: "Start Tauri dev mode with hot reload",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]
		if !isTauriProject(dir) {
			return fmt.Errorf("not a Tauri project (no src-tauri/tauri.conf.json in %s)", dir)
		}
		if err := ensureCargoTauri(); err != nil {
			return err
		}
		cli.Info("Starting Tauri dev mode in %s...", dir)
		return runCargoTauri(dir, "dev")
	},
}

// --- tauri build ---

var tauriBuildCmd = &cobra.Command{
	Use:   "build <platform> <app-directory>",
	Short: "Build Tauri app for a platform",
	Long: `Build a Tauri application for the specified platform.

All dependencies are installed automatically if missing.

Platforms:
  macos      Build on host Mac → .app + .dmg
  windows    Build in Windows UTM VM → .msi + .exe
  linux      Build in Linux UTM VM → .deb + .AppImage
  ios        Build on host Mac → iOS app
  android    Build on host Mac → Android APK/AAB

Examples:
  utm-dev tauri build macos examples/tauri-basic
  utm-dev tauri build windows examples/tauri-basic
  utm-dev tauri build ios examples/tauri-basic`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		platform := args[0]
		dir := args[1]

		if !isTauriProject(dir) {
			return fmt.Errorf("not a Tauri project (no src-tauri/tauri.conf.json in %s)", dir)
		}

		debug, _ := cmd.Flags().GetBool("debug")

		switch platform {
		case "macos":
			return tauriBuildDesktop(dir, debug)
		case "windows":
			return tauriBuildViaVM(dir, "Windows 11", debug)
		case "linux":
			return tauriBuildViaVM(dir, "Ubuntu", debug)
		case "ios":
			return tauriBuildMobile(dir, "ios", debug)
		case "android":
			return tauriBuildMobile(dir, "android", debug)
		default:
			return fmt.Errorf("unknown platform: %s (valid: macos, windows, linux, ios, android)", platform)
		}
	},
}

func tauriBuildDesktop(dir string, debug bool) error {
	if err := ensureCargoTauri(); err != nil {
		return err
	}
	cli.Info("Building Tauri app for macOS...")
	args := []string{"build"}
	if debug {
		args = append(args, "--debug")
	}
	if err := runCargoTauri(dir, args...); err != nil {
		return fmt.Errorf("tauri build failed: %w", err)
	}
	cli.Success("macOS build complete — check src-tauri/target/release/bundle/")
	return nil
}

func tauriBuildViaVM(dir string, vmName string, debug bool) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to resolve path: %w", err)
	}
	appName := filepath.Base(absDir)

	// Ensure VM is installed + running (idempotent)
	if err := ensureVM(vmName); err != nil {
		return err
	}

	// Bootstrap utm-dev inside VM (idempotent)
	if err := ensureVMToolchain(vmName); err != nil {
		return err
	}

	// Build inside VM using utm-dev (which handles cargo-tauri setup)
	cli.Info("Building Tauri app for %s in VM '%s'...", vmName, vmName)
	buildCmd := fmt.Sprintf("cd %s && utm-dev tauri build windows .", appName)
	if debug {
		buildCmd = fmt.Sprintf("cd %s && utm-dev tauri build windows . --debug", appName)
	}

	if err := utm.ExecInVM(vmName, buildCmd); err != nil {
		return fmt.Errorf("VM build failed: %w", err)
	}

	cli.Success("Build complete in VM '%s'", vmName)
	return nil
}

// ensureVMToolchain ensures utm-dev is installed inside the VM.
// utm-dev inside the VM handles installing Rust, cargo-tauri, etc. idempotently.
func ensureVMToolchain(vmName string) error {
	// Check if utm-dev is already available in the VM
	if err := utm.ExecInVM(vmName, "utm-dev self version"); err == nil {
		return nil // Already installed
	}

	cli.Info("Bootstrapping utm-dev inside VM '%s'...", vmName)

	// Cross-compile utm-dev for Windows ARM
	windowsBinary, err := crossCompileForVM()
	if err != nil {
		return fmt.Errorf("failed to cross-compile utm-dev: %w", err)
	}

	// Push binary into VM
	remotePath := `C:\Users\User\utm-dev.exe`
	cli.Info("Pushing utm-dev to VM...")
	if err := utm.PushFile(vmName, windowsBinary, remotePath); err != nil {
		return fmt.Errorf("failed to push utm-dev to VM: %w", err)
	}

	// Install: copy to a PATH location
	installCmd := fmt.Sprintf(`copy "%s" "C:\Windows\utm-dev.exe"`, remotePath)
	if err := utm.ExecInVM(vmName, installCmd); err != nil {
		// Fallback: just use from user profile
		cli.Warn("Could not install to system path, using user profile location")
	}

	// Verify
	if err := utm.ExecInVM(vmName, "utm-dev self version"); err != nil {
		return fmt.Errorf("utm-dev verification failed in VM: %w", err)
	}

	cli.Success("utm-dev bootstrapped in VM '%s'", vmName)
	return nil
}

// crossCompileForVM builds utm-dev for Windows ARM64.
func crossCompileForVM() (string, error) {
	outPath := filepath.Join(".bin", "utm-dev-windows-arm64.exe")

	// Skip if already built (idempotent)
	if _, err := os.Stat(outPath); err == nil {
		return outPath, nil
	}

	cli.Info("Cross-compiling utm-dev for Windows ARM64...")
	cmd := exec.Command("go", "build", "-o", outPath, ".")
	cmd.Env = append(os.Environ(), "GOOS=windows", "GOARCH=arm64", "CGO_ENABLED=0")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("cross-compile failed: %w", err)
	}

	return outPath, nil
}

func tauriBuildMobile(dir string, platform string, debug bool) error {
	if err := ensureCargoTauri(); err != nil {
		return err
	}

	// For Android, ensure SDK is ready
	if platform == "android" {
		if err := ensureAndroidSDK("tauri-android"); err != nil {
			return err
		}
	}

	// For iOS, ensure cocoapods
	if platform == "ios" {
		if err := ensureCocoapods(); err != nil {
			return err
		}
	}

	// Auto-init if platform target hasn't been set up yet
	if err := ensureTauriMobileInit(dir, platform); err != nil {
		return err
	}

	cli.Info("Building Tauri app for %s...", platform)
	args := []string{platform, "build"}
	if debug {
		args = append(args, "--debug")
	}

	// iOS: build for simulator if no signing cert available
	if platform == "ios" {
		if os.Getenv("APPLE_DEVELOPMENT_TEAM") == "" && !hasXcodeSigningTeam(dir) {
			cli.Info("No code signing cert found — building for iOS simulator (aarch64-sim)")
			args = append(args, "--target", "aarch64-sim")
		}
	}

	if err := runCargoTauri(dir, args...); err != nil {
		return fmt.Errorf("tauri %s build failed: %w", platform, err)
	}
	cli.Success("%s build complete", platform)
	return nil
}

// hasXcodeSigningTeam checks if tauri.conf.json has a developmentTeam configured.
func hasXcodeSigningTeam(dir string) bool {
	confPath := filepath.Join(dir, "src-tauri", "tauri.conf.json")
	data, err := os.ReadFile(confPath)
	if err != nil {
		return false
	}
	// Quick check — if developmentTeam appears with a non-empty value
	return bytes.Contains(data, []byte(`"developmentTeam"`)) &&
		!bytes.Contains(data, []byte(`"developmentTeam": ""`)) &&
		!bytes.Contains(data, []byte(`"developmentTeam":""`))
}

// --- tauri run ---

var tauriRunCmd = &cobra.Command{
	Use:   "run <platform> <app-directory>",
	Short: "Build and run Tauri app on a platform",
	Long: `Build and run a Tauri application on the specified platform.

All dependencies are installed automatically if missing.
Simulators/emulators are booted if not running.

Platforms:
  macos          Build and open on host Mac
  ios            Build and run on iOS simulator
  android        Build and run on Android emulator
  windows        Build and run in Windows UTM VM

Examples:
  utm-dev tauri run macos examples/tauri-basic
  utm-dev tauri run ios examples/tauri-basic
  utm-dev tauri run android examples/tauri-basic`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		platform := args[0]
		dir := args[1]

		if !isTauriProject(dir) {
			return fmt.Errorf("not a Tauri project (no src-tauri/tauri.conf.json in %s)", dir)
		}
		if err := ensureCargoTauri(); err != nil {
			return err
		}

		switch platform {
		case "macos":
			cli.Info("Running Tauri app on macOS...")
			return runCargoTauri(dir, "dev")
		case "ios":
			sim, err := ensureIOSSimulator()
			if err != nil {
				return err
			}
			if err := ensureCocoapods(); err != nil {
				return err
			}
			if err := ensureTauriMobileInit(dir, "ios"); err != nil {
				return err
			}
			// Pass booted simulator name so Tauri doesn't prompt interactively
			booted, _ := sim.BootedDevices()
			if len(booted) > 0 {
				cli.Info("Running Tauri app on iOS simulator (%s)...", booted[0].Name)
				return runCargoTauri(dir, "ios", "dev", booted[0].Name)
			}
			cli.Info("Running Tauri app on iOS simulator...")
			return runCargoTauri(dir, "ios", "dev")
		case "android":
			if err := ensureAndroidSDK("tauri-android"); err != nil {
				return err
			}
			cli.Info("Running Tauri app on Android emulator...")
			return runCargoTauri(dir, "android", "dev")
		case "windows":
			if err := ensureVM("Windows 11"); err != nil {
				return err
			}
			cli.Info("Running Tauri app in Windows UTM VM...")
			return utm.ExecInVM("Windows 11", "cargo tauri dev")
		default:
			return fmt.Errorf("unknown platform: %s (valid: macos, ios, android, windows)", platform)
		}
	},
}

// --- tauri init ---

var tauriInitCmd = &cobra.Command{
	Use:   "init <platform> <app-directory>",
	Short: "Initialize Tauri for a mobile platform (one-time setup)",
	Long: `Initialize Tauri mobile targets. Required once before first build.

Platforms:
  ios        Set up Xcode project for iOS
  android    Set up Gradle project for Android

Examples:
  utm-dev tauri init ios examples/tauri-basic
  utm-dev tauri init android examples/tauri-basic`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		platform := args[0]
		dir := args[1]

		if !isTauriProject(dir) {
			return fmt.Errorf("not a Tauri project (no src-tauri/tauri.conf.json in %s)", dir)
		}
		if err := ensureCargoTauri(); err != nil {
			return err
		}

		switch platform {
		case "ios":
			if err := ensureCocoapods(); err != nil {
				return err
			}
			cli.Info("Initializing Tauri iOS target...")
			if err := runCargoTauri(dir, "ios", "init"); err != nil {
				return fmt.Errorf("iOS init failed: %w", err)
			}
			cli.Success("iOS target initialized")
		case "android":
			if err := ensureAndroidSDK("tauri-android"); err != nil {
				return err
			}
			cli.Info("Initializing Tauri Android target...")
			if err := runCargoTauri(dir, "android", "init"); err != nil {
				return fmt.Errorf("Android init failed: %w", err)
			}
			cli.Success("Android target initialized")
		default:
			return fmt.Errorf("init only needed for mobile platforms: ios, android")
		}
		return nil
	},
}

// --- tauri icons ---

var tauriIconsCmd = &cobra.Command{
	Use:   "icons <app-directory> [source-icon]",
	Short: "Generate platform icons from a source image",
	Long: `Generate all platform-specific icons for a Tauri app.

Uses cargo tauri icon to generate icons for all platforms from a single source PNG.
Default source: src-tauri/icons/icon-source.png

Examples:
  utm-dev tauri icons examples/tauri-basic
  utm-dev tauri icons examples/tauri-basic ./my-icon.png`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := args[0]

		if !isTauriProject(dir) {
			return fmt.Errorf("not a Tauri project (no src-tauri/tauri.conf.json in %s)", dir)
		}
		if err := ensureCargoTauri(); err != nil {
			return err
		}

		cli.Info("Generating Tauri icons...")

		iconArgs := []string{"icon"}
		if len(args) > 1 {
			iconArgs = append(iconArgs, args[1])
		}

		if err := runCargoTauri(dir, iconArgs...); err != nil {
			return fmt.Errorf("icon generation failed: %w", err)
		}
		cli.Success("Icons generated in src-tauri/icons/")
		return nil
	},
}

// --- tauri screenshot ---

var tauriScreenshotCmd = &cobra.Command{
	Use:   "screenshot <platform> [output-file]",
	Short: "Capture a screenshot from the running app",
	Long: `Capture a screenshot from the platform where the app is running.

Automatically boots simulators/emulators if not running.
Automatically installs tools (adb, etc.) if missing.

Platforms:
  macos      Screenshot the macOS desktop (uses screencapture)
  windows    Screenshot the Windows UTM VM
  ios        Screenshot the booted iOS simulator
  android    Screenshot the connected Android device/emulator

Examples:
  utm-dev tauri screenshot ios
  utm-dev tauri screenshot ios app-store-shot.png
  utm-dev tauri screenshot windows windows-check.png
  utm-dev tauri screenshot android --clean-status`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		platform := args[0]
		output := fmt.Sprintf("screenshot-%s.png", platform)
		if len(args) > 1 {
			output = args[1]
		}

		switch platform {
		case "macos":
			return screenshotMacOS(output)
		case "windows":
			vmName, _ := cmd.Flags().GetString("vm")
			return screenshotVM(vmName, output)
		case "ios":
			cleanBar, _ := cmd.Flags().GetBool("clean-status")
			return screenshotIOS(output, cleanBar)
		case "android":
			return screenshotAndroid(output)
		default:
			return fmt.Errorf("unknown platform: %s (valid: macos, windows, ios, android)", platform)
		}
	},
}

func screenshotMacOS(output string) error {
	cli.Info("Capturing macOS screenshot...")
	cmd := exec.Command("screencapture", "-x", output)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("screencapture failed: %w", err)
	}
	cli.Success("Screenshot saved to %s", output)
	return nil
}

func screenshotVM(vmName, output string) error {
	// Ensure VM is running
	if err := ensureVM(vmName); err != nil {
		return err
	}

	remotePath := "C:\\Users\\User\\utm-dev-screenshot.png"

	psScript := fmt.Sprintf(`powershell -Command "Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; $bounds = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds; $bmp = New-Object System.Drawing.Bitmap($bounds.Width, $bounds.Height); $graphics = [System.Drawing.Graphics]::FromImage($bmp); $graphics.CopyFromScreen($bounds.Location, [System.Drawing.Point]::Empty, $bounds.Size); $bmp.Save('%s'); $graphics.Dispose(); $bmp.Dispose()"`, remotePath)

	cli.Info("Capturing screenshot in VM '%s'...", vmName)
	if err := utm.ExecInVM(vmName, psScript); err != nil {
		return fmt.Errorf("screenshot failed in VM: %w", err)
	}

	cli.Info("Pulling screenshot to %s...", output)
	if err := utm.PullFile(vmName, remotePath, output); err != nil {
		return fmt.Errorf("failed to pull screenshot: %w", err)
	}

	_ = utm.ExecInVM(vmName, fmt.Sprintf("del %s", remotePath))

	cli.Success("Screenshot saved to %s", output)
	return nil
}

func screenshotIOS(output string, cleanBar bool) error {
	// Ensure simulator is booted (idempotent)
	client, err := ensureIOSSimulator()
	if err != nil {
		return err
	}

	if cleanBar {
		cli.Info("Setting clean status bar (9:41, full battery)...")
		if err := client.StatusBarOverride(); err != nil {
			cli.Warn("could not override status bar: %v", err)
		}
		defer client.StatusBarClear()
	}

	cli.Info("Capturing iOS simulator screenshot...")
	if err := client.Screenshot(output); err != nil {
		return fmt.Errorf("screenshot failed: %w", err)
	}
	cli.Success("Screenshot saved to %s", output)
	return nil
}

func screenshotAndroid(output string) error {
	// Ensure adb is installed (idempotent)
	if err := ensureAndroidSDK("tauri-android"); err != nil {
		return err
	}

	client := adb.New()
	if !client.HasDevice() {
		return fmt.Errorf("no Android device/emulator connected — start one with: utm-dev android emulator start <avd>")
	}

	cli.Info("Capturing Android screenshot...")
	if err := client.Screenshot(output); err != nil {
		return fmt.Errorf("screenshot failed: %w", err)
	}
	cli.Success("Screenshot saved to %s", output)
	return nil
}

// --- tauri verify ---

var tauriVerifyCmd = &cobra.Command{
	Use:   "verify <platform> <app-directory>",
	Short: "Full cycle: build → run → screenshot",
	Long: `Run the full verification cycle for a Tauri app on a platform.

Everything is automatic — dependencies are installed, simulators booted,
VMs started, builds run, and screenshots captured.

  1. Ensure all platform dependencies
  2. Build the app
  3. Launch it (simulator, emulator, or VM)
  4. Capture a screenshot

Output: screenshots saved to <app-dir>/.screenshots/<platform>.png

Examples:
  utm-dev tauri verify macos examples/tauri-basic
  utm-dev tauri verify ios examples/tauri-basic
  utm-dev tauri verify windows examples/tauri-basic
  utm-dev tauri verify android examples/tauri-basic --delay 10`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		platform := args[0]
		dir := args[1]

		if !isTauriProject(dir) {
			return fmt.Errorf("not a Tauri project (no src-tauri/tauri.conf.json in %s)", dir)
		}

		delay, _ := cmd.Flags().GetInt("delay")
		cleanBar, _ := cmd.Flags().GetBool("clean-status")
		debug, _ := cmd.Flags().GetBool("debug")

		// Create screenshots directory
		screenshotDir := filepath.Join(dir, ".screenshots")
		if err := os.MkdirAll(screenshotDir, 0755); err != nil {
			return fmt.Errorf("failed to create screenshot dir: %w", err)
		}
		output := filepath.Join(screenshotDir, platform+".png")

		switch platform {
		case "macos":
			return verifyMacOS(dir, output, delay, debug)
		case "windows":
			vmName, _ := cmd.Flags().GetString("vm")
			return verifyWindows(dir, vmName, output, delay, debug)
		case "ios":
			return verifyIOS(dir, output, delay, cleanBar, debug)
		case "android":
			return verifyAndroid(dir, output, delay, debug)
		default:
			return fmt.Errorf("unknown platform: %s (valid: macos, windows, ios, android)", platform)
		}
	},
}

func verifyMacOS(dir, output string, delay int, debug bool) error {
	// Step 1: Build (ensureCargoTauri called inside)
	cli.Info("[1/3] Building for macOS...")
	if err := tauriBuildDesktop(dir, debug); err != nil {
		return err
	}

	// Step 2: Launch
	cli.Info("[2/3] Launching app...")
	appName := filepath.Base(dir)
	appPath := filepath.Join(dir, "src-tauri", "target", "release", "bundle", "macos", appName+".app")
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		appPath = filepath.Join(dir, "src-tauri", "target", "release", "bundle", "macos")
		cli.Warn("Looking for .app bundle in %s", appPath)
	}
	launchCmd := exec.Command("open", appPath)
	if err := launchCmd.Run(); err != nil {
		cli.Warn("Could not auto-launch: %v", err)
	}

	// Step 3: Wait + screenshot
	cli.Info("[3/3] Waiting %ds for app to start...", delay)
	time.Sleep(time.Duration(delay) * time.Second)
	return screenshotMacOS(output)
}

func verifyWindows(dir, vmName, output string, delay int, debug bool) error {
	// Step 1: Build in VM (ensureVM called inside tauriBuildViaVM)
	cli.Info("[1/3] Building in VM '%s'...", vmName)
	if err := tauriBuildViaVM(dir, vmName, debug); err != nil {
		return err
	}

	// Step 2: Run the built exe in VM
	cli.Info("[2/3] Launching app in VM...")
	appName := filepath.Base(dir)
	runCmd := fmt.Sprintf("start %s\\src-tauri\\target\\release\\%s.exe", appName, appName)
	if err := utm.ExecInVM(vmName, runCmd); err != nil {
		cli.Warn("Could not auto-launch in VM: %v", err)
	}

	// Step 3: Wait + screenshot (ensureVM already done)
	cli.Info("[3/3] Waiting %ds for app to start...", delay)
	time.Sleep(time.Duration(delay) * time.Second)
	return screenshotVM(vmName, output)
}

func verifyIOS(dir, output string, delay int, cleanBar bool, debug bool) error {
	// Ensure deps (idempotent)
	if err := ensureCargoTauri(); err != nil {
		return err
	}
	if _, err := ensureIOSSimulator(); err != nil {
		return err
	}

	// Step 1: Build
	cli.Info("[1/3] Building for iOS...")
	if err := tauriBuildMobile(dir, "ios", debug); err != nil {
		return err
	}

	// Step 2: Run on simulator (blocks)
	cli.Info("[2/3] Installing and launching on iOS simulator...")
	errCh := make(chan error, 1)
	go func() {
		errCh <- runCargoTauri(dir, "ios", "dev")
	}()

	// Step 3: Wait + screenshot
	cli.Info("[3/3] Waiting %ds for app to start...", delay)
	time.Sleep(time.Duration(delay) * time.Second)

	if err := screenshotIOS(output, cleanBar); err != nil {
		return err
	}

	cli.Success("Verification complete — screenshot: %s", output)
	cli.Info("iOS dev server still running (Ctrl+C to stop)")
	return <-errCh
}

func verifyAndroid(dir, output string, delay int, debug bool) error {
	// Ensure deps (idempotent)
	if err := ensureCargoTauri(); err != nil {
		return err
	}
	if err := ensureAndroidSDK("tauri-android"); err != nil {
		return err
	}

	// Step 1: Build
	cli.Info("[1/3] Building for Android...")
	if err := tauriBuildMobile(dir, "android", debug); err != nil {
		return err
	}

	// Step 2: Run on emulator (blocks)
	cli.Info("[2/3] Installing and launching on Android emulator...")
	errCh := make(chan error, 1)
	go func() {
		errCh <- runCargoTauri(dir, "android", "dev")
	}()

	// Step 3: Wait + screenshot
	cli.Info("[3/3] Waiting %ds for app to start...", delay)
	time.Sleep(time.Duration(delay) * time.Second)

	if err := screenshotAndroid(output); err != nil {
		return err
	}

	cli.Success("Verification complete — screenshot: %s", output)
	cli.Info("Android dev server still running (Ctrl+C to stop)")
	return <-errCh
}

func init() {
	tauriCmd.GroupID = "build"

	// Build flags
	tauriBuildCmd.Flags().Bool("debug", false, "Build in debug mode")

	// Screenshot flags
	tauriScreenshotCmd.Flags().String("vm", "Windows 11", "VM name for Windows screenshots")
	tauriScreenshotCmd.Flags().Bool("clean-status", false, "Clean status bar for iOS (9:41, full battery)")

	// Verify flags
	tauriVerifyCmd.Flags().Int("delay", 5, "Seconds to wait after launch before screenshot")
	tauriVerifyCmd.Flags().Bool("clean-status", false, "Clean status bar for iOS App Store screenshots")
	tauriVerifyCmd.Flags().Bool("debug", false, "Build in debug mode")
	tauriVerifyCmd.Flags().String("vm", "Windows 11", "VM name for Windows builds")

	// Setup flags
	tauriSetupCmd.Flags().Bool("skip-vm", false, "Skip UTM VM setup (large download)")
	tauriSetupCmd.Flags().Bool("mobile-only", false, "Only install mobile dependencies")

	// Wire subcommands
	tauriCmd.AddCommand(tauriSetupCmd)
	tauriCmd.AddCommand(tauriDevCmd)
	tauriCmd.AddCommand(tauriBuildCmd)
	tauriCmd.AddCommand(tauriRunCmd)
	tauriCmd.AddCommand(tauriInitCmd)
	tauriCmd.AddCommand(tauriIconsCmd)
	tauriCmd.AddCommand(tauriScreenshotCmd)
	tauriCmd.AddCommand(tauriVerifyCmd)

	rootCmd.AddCommand(tauriCmd)
}
