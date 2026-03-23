package cmd

import (
	"fmt"
	"os"

	"github.com/joeblew999/utm-dev/pkg/adb"
	"github.com/joeblew999/utm-dev/pkg/cli"
	"github.com/joeblew999/utm-dev/pkg/utils"
	"github.com/spf13/cobra"
)

func newADBClient() (*adb.Client, error) {
	client := adb.New()
	if !client.Available() {
		// Auto-install platform-tools (idempotent)
		cli.Info("adb not found, installing platform-tools...")
		if err := ensureAndroidSDK("gio-android"); err != nil {
			return nil, fmt.Errorf("failed to install platform-tools: %w", err)
		}
		// Re-check after install
		client = adb.New()
		if !client.Available() {
			return nil, fmt.Errorf("adb still not found at %s after install", client.ADBPath())
		}
	}
	return client, nil
}

// ensureEmulator ensures adb + emulator are installed. Idempotent.
func ensureEmulator() (*adb.Client, error) {
	if err := ensureAndroidSDK("gio-android"); err != nil {
		return nil, err
	}
	client := adb.New()
	if !client.EmulatorAvailable() {
		cli.Info("Emulator not found, installing...")
		cache, err := utils.NewCacheWithDirectories()
		if err != nil {
			return nil, fmt.Errorf("failed to create cache: %w", err)
		}
		if err := installSdk("emulator", cache); err != nil {
			return nil, fmt.Errorf("failed to install emulator: %w", err)
		}
		cli.Success("Emulator installed")
		client = adb.New()
		if !client.EmulatorAvailable() {
			return nil, fmt.Errorf("emulator still not found at %s after install", client.EmulatorPath())
		}
	}
	return client, nil
}

var androidCmd = &cobra.Command{
	Use:   "android",
	Short: "Android device and emulator management",
	Long:  `Manage Android devices, emulators, and apps using utm-dev's managed SDK.`,
}

var androidDevicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "List connected Android devices",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newADBClient()
		if err != nil {
			return err
		}
		devices, err := client.Devices()
		if err != nil {
			return err
		}
		if len(devices) == 0 {
			fmt.Println("No devices connected.")
			fmt.Println("Start an emulator with: utm-dev android emulator start <avd-name>")
			return nil
		}
		for _, d := range devices {
			model := d.Model
			if model == "" {
				model = "(unknown)"
			}
			fmt.Printf("%s\t%s\t%s\n", d.Serial, d.State, model)
		}
		return nil
	},
}

var androidInstallCmd = &cobra.Command{
	Use:   "install [apk-path]",
	Short: "Install an APK on the connected device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newADBClient()
		if err != nil {
			return err
		}
		apkPath := args[0]
		if _, err := os.Stat(apkPath); os.IsNotExist(err) {
			return fmt.Errorf("APK not found: %s", apkPath)
		}
		fmt.Printf("Installing %s...\n", apkPath)
		if err := client.Install(apkPath); err != nil {
			return fmt.Errorf("install failed: %w", err)
		}
		fmt.Println("✓ Installed successfully")
		return nil
	},
}

var androidUninstallCmd = &cobra.Command{
	Use:   "uninstall [package-name]",
	Short: "Uninstall an app from the connected device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newADBClient()
		if err != nil {
			return err
		}
		fmt.Printf("Uninstalling %s...\n", args[0])
		if err := client.Uninstall(args[0]); err != nil {
			return fmt.Errorf("uninstall failed: %w", err)
		}
		fmt.Println("✓ Uninstalled")
		return nil
	},
}

var androidLaunchCmd = &cobra.Command{
	Use:   "launch [package-name]",
	Short: "Launch an app on the connected device",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newADBClient()
		if err != nil {
			return err
		}
		fmt.Printf("Launching %s...\n", args[0])
		return client.Launch(args[0])
	},
}

var androidScreenshotCmd = &cobra.Command{
	Use:   "screenshot [output-file]",
	Short: "Capture a screenshot from the connected device",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newADBClient()
		if err != nil {
			return err
		}
		output := "android-screenshot.png"
		if len(args) > 0 {
			output = args[0]
		}
		fmt.Printf("Capturing screenshot...\n")
		if err := client.Screenshot(output); err != nil {
			return fmt.Errorf("screenshot failed: %w", err)
		}
		fmt.Printf("✓ Screenshot saved to %s\n", output)
		return nil
	},
}

var androidLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Stream Android logs for Gio apps (Ctrl+C to stop)",
	Long: `Stream filtered logcat output showing only Gio/Go-related log messages.
Use --all to show all device logs instead of just Gio-filtered ones.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newADBClient()
		if err != nil {
			return err
		}
		all, _ := cmd.Flags().GetBool("all")
		if all {
			fmt.Println("Streaming all device logs (Ctrl+C to stop)...")
			return client.Logcat()
		}
		fmt.Println("Streaming Gio app logs (Ctrl+C to stop)...")
		return client.Logcat("GoLog:V", "GioView:V", "System.err:W")
	},
}

var androidWebviewCmd = &cobra.Command{
	Use:   "webview",
	Short: "Show WebView version on the connected device",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newADBClient()
		if err != nil {
			return err
		}
		version, err := client.WebViewVersion()
		if err != nil {
			return fmt.Errorf("failed to get webview version: %w", err)
		}
		fmt.Println(version)
		return nil
	},
}

// Emulator subcommands

var androidEmulatorCmd = &cobra.Command{
	Use:   "emulator",
	Short: "Manage Android emulators",
}

var androidEmulatorListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available Android emulators (AVDs)",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := ensureEmulator()
		if err != nil {
			return err
		}
		avds, err := client.EmulatorList()
		if err != nil {
			return err
		}
		if len(avds) == 0 {
			fmt.Println("No AVDs found.")
			fmt.Println("Create one with Android Studio or avdmanager.")
			return nil
		}
		fmt.Println("Available AVDs:")
		for _, avd := range avds {
			fmt.Printf("  %s\n", avd)
		}
		return nil
	},
}

var androidEmulatorStartCmd = &cobra.Command{
	Use:   "start [avd-name]",
	Short: "Start an Android emulator",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := ensureEmulator()
		if err != nil {
			return err
		}
		avdName := args[0]
		fmt.Printf("Starting emulator %s...\n", avdName)
		pid, err := client.EmulatorStart(avdName)
		if err != nil {
			return err
		}
		fmt.Printf("Emulator started (PID: %d)\n", pid)
		fmt.Println("Waiting for device to come online...")
		if err := client.WaitForDevice(); err != nil {
			return fmt.Errorf("device did not come online: %w", err)
		}
		fmt.Println("✓ Emulator is ready")
		return nil
	},
}

func init() {
	// Logs flags
	androidLogsCmd.Flags().Bool("all", false, "Show all device logs (not just Gio-filtered)")

	// Emulator subcommands
	androidEmulatorCmd.AddCommand(androidEmulatorListCmd)
	androidEmulatorCmd.AddCommand(androidEmulatorStartCmd)

	// Android subcommands
	androidCmd.AddCommand(androidDevicesCmd)
	androidCmd.AddCommand(androidInstallCmd)
	androidCmd.AddCommand(androidUninstallCmd)
	androidCmd.AddCommand(androidLaunchCmd)
	androidCmd.AddCommand(androidScreenshotCmd)
	androidCmd.AddCommand(androidLogsCmd)
	androidCmd.AddCommand(androidWebviewCmd)
	androidCmd.AddCommand(androidEmulatorCmd)

	androidCmd.GroupID = "util"
	rootCmd.AddCommand(androidCmd)
}
