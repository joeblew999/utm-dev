package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/joeblew999/utm-dev/pkg/adb"
	"github.com/joeblew999/utm-dev/pkg/cli"
	"github.com/joeblew999/utm-dev/pkg/config"
	"github.com/joeblew999/utm-dev/pkg/simctl"
	"github.com/joeblew999/utm-dev/pkg/utm"
	"github.com/spf13/cobra"
)

// isTauriProject checks for src-tauri/tauri.conf.json in the given directory.
func isTauriProject(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "src-tauri", "tauri.conf.json"))
	return err == nil
}

// ensureCargoTauri checks that cargo-tauri is installed.
func ensureCargoTauri() error {
	if _, err := exec.LookPath("cargo"); err != nil {
		return fmt.Errorf("cargo not found — install Rust: https://rustup.rs")
	}
	// cargo tauri --version to check if tauri-cli is installed
	cmd := exec.Command("cargo", "tauri", "--version")
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cargo-tauri not found\n\nInstall with:\n  cargo install tauri-cli\n\nOr via mise:\n  mise use cargo:tauri-cli@2")
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
	return cmd.Run()
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

This sets up:
  1. Rust toolchain (via rustup)
  2. cargo-tauri CLI
  3. Android SDK + NDK (for mobile builds)
  4. UTM + Windows 11 VM (for Windows desktop builds)
  5. Xcode check (can't auto-install, but verifies it's there)

Run this once on a fresh machine to get everything working.

Examples:
  utm-dev tauri setup                # Install everything
  utm-dev tauri setup --skip-vm      # Skip UTM VM setup (large download)
  utm-dev tauri setup --mobile-only  # Only install mobile deps`,
	RunE: func(cmd *cobra.Command, args []string) error {
		skipVM, _ := cmd.Flags().GetBool("skip-vm")
		mobileOnly, _ := cmd.Flags().GetBool("mobile-only")

		cli.Info("Setting up Tauri development environment...")

		// Step 1: Rust
		cli.Info("[1/5] Checking Rust toolchain...")
		if _, err := exec.LookPath("cargo"); err != nil {
			cli.Info("Installing Rust via rustup...")
			rustup := exec.Command("sh", "-c", "curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y")
			rustup.Stdout = os.Stdout
			rustup.Stderr = os.Stderr
			if err := rustup.Run(); err != nil {
				return fmt.Errorf("failed to install Rust: %w\n\nInstall manually: https://rustup.rs", err)
			}
			cli.Success("Rust installed")
		} else {
			cli.Success("Rust already installed")
		}

		// Step 2: cargo-tauri
		cli.Info("[2/5] Checking cargo-tauri...")
		checkCmd := exec.Command("cargo", "tauri", "--version")
		checkCmd.Stdout = nil
		checkCmd.Stderr = nil
		if err := checkCmd.Run(); err != nil {
			cli.Info("Installing cargo-tauri...")
			install := exec.Command("cargo", "install", "tauri-cli")
			install.Stdout = os.Stdout
			install.Stderr = os.Stderr
			if err := install.Run(); err != nil {
				return fmt.Errorf("failed to install cargo-tauri: %w", err)
			}
			cli.Success("cargo-tauri installed")
		} else {
			cli.Success("cargo-tauri already installed")
		}

		// Step 3: Android SDK + NDK (for mobile builds)
		cli.Info("[3/5] Checking Android SDK...")
		sdkRoot := config.GetSDKDir()
		ndkPath := filepath.Join(sdkRoot, "ndk-bundle")
		if _, err := os.Stat(ndkPath); os.IsNotExist(err) {
			cli.Info("Installing Android NDK...")
			if err := installNDK(sdkRoot); err != nil {
				cli.Warn("Android NDK install failed: %v — install manually: utm-dev install ndk-bundle", err)
			} else {
				cli.Success("Android NDK installed")
			}
		} else {
			cli.Success("Android NDK already installed")
		}

		// Step 4: Xcode check (iOS)
		cli.Info("[4/5] Checking Xcode...")
		if _, err := exec.LookPath("xcrun"); err != nil {
			cli.Warn("Xcode not found — install from App Store for iOS builds")
		} else {
			cli.Success("Xcode available")
		}

		// Step 5: UTM + Windows VM (for desktop cross-platform testing)
		if !skipVM && !mobileOnly {
			cli.Info("[5/5] Checking UTM + Windows VM...")
			if err := utm.InstallUTM(false); err != nil {
				cli.Warn("UTM install issue: %v — install manually: utm-dev utm install", err)
			} else {
				cli.Success("UTM ready")
			}

			cli.Info("Importing Windows 11 VM (this downloads ~6 GB)...")
			if err := utm.InstallBox("windows-11", false); err != nil {
				cli.Warn("Windows VM import failed: %v — run later: utm-dev utm install windows-11", err)
			} else {
				cli.Success("Windows 11 VM ready")
			}
		} else {
			cli.Info("[5/5] Skipping UTM VM setup (use --skip-vm=false to include)")
		}

		cli.Success("Tauri development environment ready!")
		cli.Info("")
		cli.Info("Quick start:")
		cli.Info("  utm-dev tauri dev examples/tauri-basic           # Dev mode")
		cli.Info("  utm-dev tauri build macos examples/tauri-basic    # macOS build")
		cli.Info("  utm-dev tauri verify ios examples/tauri-basic     # Full iOS cycle")
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

	// Step 1: Ensure VM is running
	cli.Info("Checking VM '%s'...", vmName)
	status, err := utm.GetVMStatus(vmName)
	if err != nil {
		return fmt.Errorf("VM '%s' not found — install with: utm-dev utm install windows-11", vmName)
	}
	if status != "started" {
		cli.Info("Starting VM '%s'...", vmName)
		if err := utm.StartVM(vmName); err != nil {
			return fmt.Errorf("failed to start VM: %w", err)
		}
		// Wait for Windows to be ready (WinRM responding)
		cli.Info("Waiting for Windows to boot...")
		if err := utm.WaitForWindows("localhost", 5*time.Minute); err != nil {
			cli.Warn("WinRM not responding — VM may still be booting. Continuing anyway...")
		}
	}

	// Step 2: Build inside VM
	// The VM should have a shared folder or the project cloned via git.
	// Use utm exec to run cargo tauri build in the project directory.
	cli.Info("Building Tauri app for %s in VM '%s'...", vmName, vmName)
	cli.Info("Project: %s", absDir)

	buildCmd := fmt.Sprintf("cd %s && cargo tauri build", appName)
	if debug {
		buildCmd = fmt.Sprintf("cd %s && cargo tauri build --debug", appName)
	}

	if err := utm.ExecInVM(vmName, buildCmd); err != nil {
		return fmt.Errorf("VM build failed: %w\n\nTroubleshooting:\n"+
			"  1. Ensure Rust + cargo-tauri are installed in the VM\n"+
			"  2. Ensure the project is synced to the VM (shared folder or git clone)\n"+
			"  3. Check VM status: utm-dev utm status \"%s\"", err, vmName)
	}

	cli.Success("Build complete in VM '%s'", vmName)
	cli.Info("Pull artifacts with: utm-dev utm pull \"%s\" <remote-path> ./artifacts/", vmName)
	return nil
}

func tauriBuildMobile(dir string, platform string, debug bool) error {
	if err := ensureCargoTauri(); err != nil {
		return err
	}
	cli.Info("Building Tauri app for %s...", platform)
	args := []string{platform, "build"}
	if debug {
		args = append(args, "--debug")
	}
	if err := runCargoTauri(dir, args...); err != nil {
		return fmt.Errorf("tauri %s build failed: %w", platform, err)
	}
	cli.Success("%s build complete", platform)
	return nil
}

// --- tauri run ---

var tauriRunCmd = &cobra.Command{
	Use:   "run <platform> <app-directory>",
	Short: "Build and run Tauri app on a platform",
	Long: `Build and run a Tauri application on the specified platform.

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
			cli.Info("Running Tauri app on iOS simulator...")
			return runCargoTauri(dir, "ios", "dev")
		case "android":
			cli.Info("Running Tauri app on Android emulator...")
			return runCargoTauri(dir, "android", "dev")
		case "windows":
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
			cli.Info("Initializing Tauri iOS target...")
			if err := runCargoTauri(dir, "ios", "init"); err != nil {
				return fmt.Errorf("iOS init failed: %w", err)
			}
			cli.Success("iOS target initialized — run with: utm-dev tauri run ios %s", dir)
		case "android":
			cli.Info("Initializing Tauri Android target...")
			if err := runCargoTauri(dir, "android", "init"); err != nil {
				return fmt.Errorf("Android init failed: %w", err)
			}
			cli.Success("Android target initialized — run with: utm-dev tauri run android %s", dir)
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
	// Remote temp path in the VM
	remotePath := "C:\\Users\\User\\utm-dev-screenshot.png"

	// PowerShell screen capture (no utm-dev needed in VM)
	psScript := fmt.Sprintf(`powershell -Command "Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; $bounds = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds; $bmp = New-Object System.Drawing.Bitmap($bounds.Width, $bounds.Height); $graphics = [System.Drawing.Graphics]::FromImage($bmp); $graphics.CopyFromScreen($bounds.Location, [System.Drawing.Point]::Empty, $bounds.Size); $bmp.Save('%s'); $graphics.Dispose(); $bmp.Dispose()"`, remotePath)

	cli.Info("Capturing screenshot in VM '%s'...", vmName)
	if err := utm.ExecInVM(vmName, psScript); err != nil {
		return fmt.Errorf("screenshot failed in VM: %w", err)
	}

	cli.Info("Pulling screenshot to %s...", output)
	if err := utm.PullFile(vmName, remotePath, output); err != nil {
		return fmt.Errorf("failed to pull screenshot: %w", err)
	}

	// Best-effort cleanup
	_ = utm.ExecInVM(vmName, fmt.Sprintf("del %s", remotePath))

	cli.Success("Screenshot saved to %s", output)
	return nil
}

func screenshotIOS(output string, cleanBar bool) error {
	client := simctl.New()
	if !client.Available() {
		return fmt.Errorf("xcrun simctl not available — install Xcode command line tools")
	}
	if !client.HasBooted() {
		return fmt.Errorf("no simulator booted — boot one with: utm-dev ios boot \"iPhone 16\"")
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
	client := adb.New()
	if !client.Available() {
		return fmt.Errorf("adb not found — install with: utm-dev install platform-tools")
	}
	if !client.HasDevice() {
		return fmt.Errorf("no Android device connected — start an emulator: utm-dev android emulator start <avd>")
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

This command automates the complete build-run-verify workflow:
  1. Build the app for the target platform
  2. Launch it on the platform (simulator, emulator, or VM)
  3. Wait for the app to start
  4. Capture a screenshot to verify it works

Platforms:
  macos      Build + open + screenshot on host Mac
  windows    Build + run + screenshot in Windows UTM VM
  ios        Build + install + launch + screenshot on iOS simulator
  android    Build + install + launch + screenshot on Android emulator

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
	// Step 1: Build
	cli.Info("[1/3] Building for macOS...")
	if err := tauriBuildDesktop(dir, debug); err != nil {
		return err
	}

	// Step 2: Launch — find and open the built .app
	cli.Info("[2/3] Launching app...")
	appName := filepath.Base(dir)
	appPath := filepath.Join(dir, "src-tauri", "target", "release", "bundle", "macos", appName+".app")
	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		// Try with product name from tauri.conf.json
		appPath = filepath.Join(dir, "src-tauri", "target", "release", "bundle", "macos")
		cli.Warn("Looking for .app bundle in %s", appPath)
	}
	launchCmd := exec.Command("open", appPath)
	if err := launchCmd.Run(); err != nil {
		cli.Warn("Could not auto-launch: %v — take screenshot manually", err)
	}

	// Step 3: Wait + screenshot
	cli.Info("[3/3] Waiting %ds for app to start...", delay)
	time.Sleep(time.Duration(delay) * time.Second)
	return screenshotMacOS(output)
}

func verifyWindows(dir, vmName, output string, delay int, debug bool) error {
	// Step 1: Build in VM
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

	// Step 3: Wait + screenshot
	cli.Info("[3/3] Waiting %ds for app to start...", delay)
	time.Sleep(time.Duration(delay) * time.Second)
	return screenshotVM(vmName, output)
}

func verifyIOS(dir, output string, delay int, cleanBar bool, debug bool) error {
	if err := ensureCargoTauri(); err != nil {
		return err
	}

	// Step 1: Build
	cli.Info("[1/3] Building for iOS...")
	if err := tauriBuildMobile(dir, "ios", debug); err != nil {
		return err
	}

	// Step 2: Run on simulator — cargo tauri ios dev builds and launches
	cli.Info("[2/3] Installing and launching on iOS simulator...")
	// Use a goroutine to run cargo tauri ios dev (it blocks), then screenshot after delay
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
	// Wait for the dev server to be killed
	return <-errCh
}

func verifyAndroid(dir, output string, delay int, debug bool) error {
	if err := ensureCargoTauri(); err != nil {
		return err
	}

	// Step 1: Build
	cli.Info("[1/3] Building for Android...")
	if err := tauriBuildMobile(dir, "android", debug); err != nil {
		return err
	}

	// Step 2: Run on emulator
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
