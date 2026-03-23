package cmd

import (
	"fmt"

	"github.com/joeblew999/utm-dev/pkg/cli"
	"github.com/joeblew999/utm-dev/pkg/icons"
	"github.com/joeblew999/utm-dev/pkg/utils"
	"github.com/spf13/cobra"
)

// iconsCmd represents the project-aware icons command
var iconsCmd = &cobra.Command{
	Use:   "icons [platform] [project-directory]",
	Short: "Generate platform-specific icons for a Gio project",
	Long: `Generate platform-specific icons for a Gio project using project-aware paths.
This command automatically manages source icons and output directories based on the project structure.

Examples:
  utm-dev icons android ./my-gio-app
  utm-dev icons ios ./my-gio-app
  utm-dev icons windows ./my-gio-app
  utm-dev icons macos ./my-gio-app`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		platform := args[0]
		projectDir := args[1]

		// Validate platform
		validPlatforms := []string{"android", "ios", "macos", "windows", "windows-msix", "windows-ico"}
		if !utils.Contains(validPlatforms, platform) {
			return fmt.Errorf("invalid platform: %s. Valid platforms: %v", platform, validPlatforms)
		}

		cli.Info("Generating %s icons for project...", platform)

		err := icons.GenerateForProject(icons.ProjectConfig{
			ProjectPath: projectDir,
			Platform:    platform,
		})
		if err != nil {
			return fmt.Errorf("failed to generate icons: %w", err)
		}

		cli.Success("Generated %s icons", platform)
		return nil
	},
}

func init() {
	iconsCmd.GroupID = "tools"
	rootCmd.AddCommand(iconsCmd)
}
