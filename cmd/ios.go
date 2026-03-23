package cmd

import (
	"fmt"
	"strings"

	"github.com/joeblew999/utm-dev/pkg/simctl"
	"github.com/spf13/cobra"
)

func newSimctlClient() (*simctl.Client, error) {
	client := simctl.New()
	if !client.Available() {
		return nil, fmt.Errorf("xcrun simctl not available\nInstall Xcode command line tools: xcode-select --install")
	}
	return client, nil
}

var iosCmd = &cobra.Command{
	Use:   "ios",
	Short: "iOS simulator management",
	Long:  `Manage iOS simulators and apps using xcrun simctl.`,
}

var iosDevicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "List available iOS simulators",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newSimctlClient()
		if err != nil {
			return err
		}
		devices, err := client.Devices()
		if err != nil {
			return err
		}
		if len(devices) == 0 {
			fmt.Println("No simulators available.")
			fmt.Println("Install runtimes via Xcode → Settings → Platforms.")
			return nil
		}
		fmt.Printf("%-40s %-10s %-15s %s\n", "NAME", "STATE", "RUNTIME", "UDID")
		for _, d := range devices {
			fmt.Printf("%-40s %-10s %-15s %s\n", d.Name, d.State, d.Runtime, d.UDID)
		}
		return nil
	},
}

var iosBootCmd = &cobra.Command{
	Use:   "boot [udid-or-name]",
	Short: "Boot an iOS simulator",
	Long: `Boot a simulator by UDID or device name.
Use 'utm-dev ios devices' to find available simulators.

Examples:
  utm-dev ios boot "iPhone 15"
  utm-dev ios boot XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newSimctlClient()
		if err != nil {
			return err
		}
		identifier := args[0]

		// Try to resolve name to UDID
		udid, err := resolveSimulatorUDID(client, identifier)
		if err != nil {
			return err
		}

		fmt.Printf("Booting simulator %s...\n", identifier)
		if err := client.Boot(udid); err != nil {
			return fmt.Errorf("boot failed: %w", err)
		}
		// Open the Simulator.app so user can see it
		client.OpenSimulatorApp()
		fmt.Println("Simulator booted")
		return nil
	},
}

var iosShutdownCmd = &cobra.Command{
	Use:   "shutdown [udid-or-name]",
	Short: "Shutdown a running iOS simulator",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newSimctlClient()
		if err != nil {
			return err
		}
		udid, err := resolveSimulatorUDID(client, args[0])
		if err != nil {
			return err
		}
		fmt.Printf("Shutting down simulator...\n")
		if err := client.Shutdown(udid); err != nil {
			return fmt.Errorf("shutdown failed: %w", err)
		}
		fmt.Println("Simulator shut down")
		return nil
	},
}

var iosInstallCmd = &cobra.Command{
	Use:   "install [app-path]",
	Short: "Install an .app bundle on the booted simulator",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newSimctlClient()
		if err != nil {
			return err
		}
		if !client.HasBooted() {
			return fmt.Errorf("no simulator is booted. Boot one with: utm-dev ios boot \"iPhone 15\"")
		}
		fmt.Printf("Installing %s...\n", args[0])
		if err := client.Install(args[0]); err != nil {
			return fmt.Errorf("install failed: %w", err)
		}
		fmt.Println("Installed successfully")
		return nil
	},
}

var iosUninstallCmd = &cobra.Command{
	Use:   "uninstall [bundle-id]",
	Short: "Uninstall an app from the booted simulator",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newSimctlClient()
		if err != nil {
			return err
		}
		fmt.Printf("Uninstalling %s...\n", args[0])
		if err := client.Uninstall(args[0]); err != nil {
			return fmt.Errorf("uninstall failed: %w", err)
		}
		fmt.Println("Uninstalled")
		return nil
	},
}

var iosLaunchCmd = &cobra.Command{
	Use:   "launch [bundle-id]",
	Short: "Launch an app on the booted simulator",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newSimctlClient()
		if err != nil {
			return err
		}
		if !client.HasBooted() {
			return fmt.Errorf("no simulator is booted. Boot one with: utm-dev ios boot \"iPhone 15\"")
		}
		fmt.Printf("Launching %s...\n", args[0])
		return client.Launch(args[0])
	},
}

var iosScreenshotCmd = &cobra.Command{
	Use:   "screenshot [output-file]",
	Short: "Capture a screenshot from the booted simulator",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newSimctlClient()
		if err != nil {
			return err
		}
		if !client.HasBooted() {
			return fmt.Errorf("no simulator is booted. Boot one with: utm-dev ios boot \"iPhone 15\"")
		}

		output := "ios-screenshot.png"
		if len(args) > 0 {
			output = args[0]
		}

		// Set clean status bar for App Store-quality screenshots
		cleanBar, _ := cmd.Flags().GetBool("clean-status")
		if cleanBar {
			fmt.Println("Setting clean status bar (9:41, full battery)...")
			if err := client.StatusBarOverride(); err != nil {
				fmt.Printf("Warning: could not override status bar: %v\n", err)
			}
			defer client.StatusBarClear()
		}

		fmt.Println("Capturing screenshot...")
		if err := client.Screenshot(output); err != nil {
			return fmt.Errorf("screenshot failed: %w", err)
		}
		fmt.Printf("Screenshot saved to %s\n", output)
		return nil
	},
}

var iosLogsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Stream iOS simulator logs (Ctrl+C to stop)",
	Long: `Stream filtered log output from the booted iOS simulator.
Use --all to show all logs instead of just Gio/Go-related ones.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newSimctlClient()
		if err != nil {
			return err
		}
		if !client.HasBooted() {
			return fmt.Errorf("no simulator is booted. Boot one with: utm-dev ios boot \"iPhone 16\"")
		}
		all, _ := cmd.Flags().GetBool("all")
		if all {
			fmt.Println("Streaming all simulator logs (Ctrl+C to stop)...")
			return client.Logs("")
		}
		fmt.Println("Streaming Gio app logs (Ctrl+C to stop)...")
		return client.Logs("processImagePath contains 'localhost'")
	},
}

var iosRuntimesCmd = &cobra.Command{
	Use:   "runtimes",
	Short: "List available iOS runtimes",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := newSimctlClient()
		if err != nil {
			return err
		}
		out, err := client.ListRuntimes()
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	},
}

// resolveSimulatorUDID resolves a name or UDID to a UDID.
// Supports exact match, prefix match, and contains match (in that priority order).
// If the input looks like a UDID (>30 chars), use it directly.
func resolveSimulatorUDID(client *simctl.Client, identifier string) (string, error) {
	// If it looks like a UDID, use directly
	if len(identifier) > 30 {
		return identifier, nil
	}

	devices, err := client.Devices()
	if err != nil {
		return "", err
	}

	idLower := strings.ToLower(identifier)

	// Pass 1: exact match
	for _, d := range devices {
		if strings.ToLower(d.Name) == idLower {
			return d.UDID, nil
		}
	}

	// Pass 2: prefix match (e.g. "iPhone 16" matches "iPhone 16 Pro")
	for _, d := range devices {
		if strings.HasPrefix(strings.ToLower(d.Name), idLower) {
			fmt.Printf("Matched: %s\n", d.Name)
			return d.UDID, nil
		}
	}

	// Pass 3: contains match (e.g. "iPad Pro" matches "iPad Pro 13-inch (M4)")
	for _, d := range devices {
		if strings.Contains(strings.ToLower(d.Name), idLower) {
			fmt.Printf("Matched: %s\n", d.Name)
			return d.UDID, nil
		}
	}

	// Show available devices in the error
	var names []string
	for _, d := range devices {
		names = append(names, fmt.Sprintf("  %s (%s)", d.Name, d.Runtime))
	}
	return "", fmt.Errorf("simulator not found: %q\n\nAvailable simulators:\n%s\n\nRun 'utm-dev ios devices' for full list",
		identifier, strings.Join(names, "\n"))
}

func init() {
	// Screenshot flags
	iosScreenshotCmd.Flags().Bool("clean-status", false, "Set clean status bar (9:41, full battery) for App Store screenshots")

	// Logs flags
	iosLogsCmd.Flags().Bool("all", false, "Show all simulator logs (not just Gio-filtered)")

	// iOS subcommands
	iosCmd.AddCommand(iosDevicesCmd)
	iosCmd.AddCommand(iosBootCmd)
	iosCmd.AddCommand(iosShutdownCmd)
	iosCmd.AddCommand(iosInstallCmd)
	iosCmd.AddCommand(iosUninstallCmd)
	iosCmd.AddCommand(iosLaunchCmd)
	iosCmd.AddCommand(iosScreenshotCmd)
	iosCmd.AddCommand(iosLogsCmd)
	iosCmd.AddCommand(iosRuntimesCmd)

	iosCmd.GroupID = "util"
	rootCmd.AddCommand(iosCmd)
}
