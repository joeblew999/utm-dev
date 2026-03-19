package cmd

import (
	"os"

	"github.com/joeblew999/utm-dev/pkg/schema"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "utm-dev",
	Short: "Build cross-platform hybrid apps with Go",
	Long: `utm-dev - Build cross-platform hybrid applications using Go and Gio UI.

Write HTML/CSS once → Deploy everywhere: Web, iOS, Android, Desktop

QUICK START:
  utm-dev build macos examples/hybrid-dashboard   Build for macOS
  utm-dev run macos examples/hybrid-dashboard     Build and run
  utm-dev icons examples/hybrid-dashboard         Generate icons

SDK MANAGEMENT:
  utm-dev install ndk-bundle                      Install Android NDK
  utm-dev list                                    List available SDKs

DOCUMENTATION:
  utm-dev docs                                    Generate CLI docs
  https://github.com/joeblew999/utm-dev`,
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
