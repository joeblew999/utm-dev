package cmd

import (
	"os"

	"github.com/joeblew999/utm-dev/pkg/cli"
	"github.com/joeblew999/utm-dev/pkg/schema"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "utm-dev",
	Short: "Build and test cross-platform apps (Tauri desktop + Gio mobile)",
	Long: `utm-dev - Cross-platform build tooling for Tauri and Gio apps.

Tauri desktop apps tested via UTM VMs on Apple Silicon.
Gio mobile apps (iOS/Android) built on host Mac.
SDK management so devs don't pollute their OS.

TAURI (desktop + mobile):
  utm-dev tauri dev examples/tauri-basic           Dev mode with hot reload
  utm-dev tauri build macos examples/tauri-basic    Build macOS .app/.dmg
  utm-dev tauri build windows examples/tauri-basic  Build in Windows UTM VM
  utm-dev tauri build ios examples/tauri-basic      Build for iOS
  utm-dev tauri run ios examples/tauri-basic        Run on iOS simulator

GIO (mobile):
  utm-dev build android examples/hybrid-dashboard  Build APK
  utm-dev run android examples/hybrid-dashboard    Build + install + launch

UTM VMs:
  utm-dev utm install windows-11                   Download + import VM
  utm-dev utm start "Windows 11"                   Start VM
  utm-dev utm exec "Windows 11" -- whoami          Run command in VM

SDK MANAGEMENT:
  utm-dev install ndk-bundle                       Install Android NDK
  utm-dev list                                     List available SDKs`,
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	// Enable suggestions for typos (e.g., "buld" → "build")
	rootCmd.SuggestionsMinimumDistance = 2

	// Add command groups for better help organization
	rootCmd.AddGroup(
		&cobra.Group{ID: "build", Title: "Build Commands:"},
		&cobra.Group{ID: "sdk", Title: "SDK Management:"},
		&cobra.Group{ID: "tools", Title: "Development Tools:"},
		&cobra.Group{ID: "vm", Title: "Virtual Machines:"},
		&cobra.Group{ID: "self", Title: "Self Management:"},
	)

	// Enable shell completion descriptions
	rootCmd.CompletionOptions.DisableDefaultCmd = false
	rootCmd.CompletionOptions.HiddenDefaultCmd = false

	// Verbosity flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Show debug output")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress info/success output")
	rootCmd.PersistentPreRun = func(cmd *cobra.Command, args []string) {
		if v, _ := cmd.Flags().GetBool("verbose"); v {
			cli.SetVerbose(true)
		}
		if q, _ := cmd.Flags().GetBool("quiet"); q {
			cli.SetQuiet(true)
		}
	}

	// Version flag
	rootCmd.Version = getVersion()
	rootCmd.SetVersionTemplate(`{{.Name}} {{.Version}}
`)
}

func getVersion() string {
	// This will be overridden by build flags in release
	return "dev"
}

// SetVersion allows setting version from main or build flags
func SetVersion(v string) {
	rootCmd.Version = v
}

// Helper to get completion for VM names
func getVMNameCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Try to get VM list for completion
	// This is a placeholder - implement actual VM listing
	return []string{}, cobra.ShellCompDirectiveNoFileComp
}

// Helper to get completion for example directories
func getExampleCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	entries, err := os.ReadDir("examples")
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []string
	for _, e := range entries {
		if e.IsDir() {
			completions = append(completions, "examples/"+e.Name())
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}

// Helper to get completion for platforms - uses shared schema
func getPlatformCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var completions []string
	for _, p := range schema.Platforms {
		desc := schema.PlatformDescriptions[p]
		completions = append(completions, p+"\t"+desc)
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}
