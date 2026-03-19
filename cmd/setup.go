package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/joeblew999/utm-dev/pkg/installer"
	"github.com/joeblew999/utm-dev/pkg/utils"
	"github.com/spf13/cobra"
)

var noEmulator bool

var setupCmd = &cobra.Command{
	Use:   "setup [setup-name]",
	Short: "Install a predefined set of SDKs",
	Long:  `Install a predefined set of SDKs defined in the sdk-android-list.json and sdk-ios-list.json files.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		setupName := args[0]

		// Ensure directories exist and create cache
		cache, err := utils.NewCacheWithDirectories()
		if err != nil {
			return err
		}

		fmt.Printf("Running setup: %s...\n", setupName)

		return installSetup(setupName, cache, noEmulator)
	},
}

func init() {
	setupCmd.Flags().BoolVar(&noEmulator, "no-emulator", false, "Skip installing the emulator and system-images (Android only)")
	rootCmd.AddCommand(setupCmd)
}

func installSetup(setupName string, cache *installer.Cache, skipEmulator bool) error {
	sdks, err := findSetup(setupName)
	if err != nil {
		return err
	}

	for _, sdkName := range sdks {
		if skipEmulator && (sdkName == "emulator" || strings.HasPrefix(sdkName, "system-images")) {
			fmt.Printf("--- Skipping %s as requested ---\n", sdkName)
			continue
		}

		if sdkName == "xcode-command-line-tools" {
			fmt.Println("--- Checking for Xcode Command Line Tools ---")
			if runtime.GOOS != "darwin" {
				fmt.Println("Skipping: Xcode Command Line Tools can only be installed on macOS.")
				continue
			}

			// First, check if the tools are already installed to avoid unnecessary pop-ups
			checkCmd := exec.Command("xcode-select", "-p")
			if err := checkCmd.Run(); err == nil {
				fmt.Println("Xcode Command Line Tools are already installed.")
				continue
			}

			// If the check failed, the tools are not installed, so attempt to install them
			fmt.Println("Xcode Command Line Tools not found. Attempting to install...")
			installCmd := exec.Command("xcode-select", "--install")
			installCmd.Stdout = os.Stdout
			installCmd.Stderr = os.Stderr
			if err := installCmd.Run(); err != nil {
				// Installation can fail if user cancels the dialog or if already installed
				fmt.Println("Installation might have been cancelled or tools are already installed.")
			} else {
				fmt.Println("--- Finished installing Xcode Command Line Tools ---")
			}
			continue
		}

		fmt.Printf("--- Installing %s ---\n", sdkName)
		if err := installSdk(sdkName, cache); err != nil {
			return fmt.Errorf("failed to install %s: %w", sdkName, err)
		}
		fmt.Printf("--- Finished installing %s ---\n\n", sdkName)
	}

	fmt.Printf("Setup '%s' completed successfully.\n", setupName)
	return nil
}

func findSetup(setupName string) ([]string, error) {
	return utils.FindSetup(setupName)
}
