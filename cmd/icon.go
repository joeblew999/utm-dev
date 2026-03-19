package cmd

import (
	"fmt"

	"github.com/joeblew999/utm-dev/pkg/icons"
	"github.com/spf13/cobra"
)

// iconCmd represents the icon command (DEPRECATED)
var iconCmd = &cobra.Command{
	Use:   "icon",
	Short: "[DEPRECATED] Generate platform-specific icons from a source image. Use 'icons' instead.",
	Long: `[DEPRECATED] Generates platform-specific icons from a source PNG image.

This command is deprecated in favor of the project-aware 'icons' command.
Please use 'utm-dev icons [platform] [project-directory]' instead.

Examples (deprecated):
  utm-dev icon --input icon.png --output ./out --platform android

Recommended (new):
  utm-dev icons android ./my-project`,
	Run: func(cmd *cobra.Command, args []string) {
		// Show deprecation warning
		fmt.Println("⚠️  WARNING: The 'icon' command is deprecated.")
		fmt.Println("   Please use 'utm-dev icons [platform] [project-directory]' instead.")
		fmt.Println("   Example: utm-dev icons android ./my-project")
		fmt.Println()

		inputPath, _ := cmd.Flags().GetString("input")
		outputPath, _ := cmd.Flags().GetString("output")
		platform, _ := cmd.Flags().GetString("platform")

		err := icons.Generate(icons.Config{
			InputPath:  inputPath,
			OutputPath: outputPath,
			Platform:   platform,
		})

		if err != nil {
			fmt.Printf("Error generating icons: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(iconCmd)
	iconCmd.Flags().StringP("input", "i", "", "Input PNG image file")
	iconCmd.Flags().StringP("output", "o", "", "Output file or directory")
	iconCmd.Flags().StringP("platform", "p", "", "Platform (macos, android, ios, windows-ico, windows-msix)")
}
