package cmd

import (
	"github.com/joeblew999/utm-dev/pkg/utils"
	"fmt"

	"github.com/joeblew999/utm-dev/pkg/service"
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

		// Get maintenance flags
		autoMaintain, _ := cmd.Flags().GetBool("auto-maintain")
		autoFix, _ := cmd.Flags().GetBool("auto-fix")
		verbose, _ := cmd.Flags().GetBool("verbose")

		// Create service with configuration
		config := service.ServiceConfig{
			Mode:         "cli",
			AutoMaintain: autoMaintain,
			AutoFix:      autoFix,
			Verbose:      verbose,
		}
		svc := service.NewGioServiceWithConfig(config)

		// Use service for icon generation
		req := service.ProjectRequest{
			ProjectPath: projectDir,
			Platform:    platform,
		}

		fmt.Printf("Generating %s icons for project...\n", platform)

		resp, err := svc.GenerateIcons(req)
		if err != nil {
			return fmt.Errorf("service error: %w", err)
		}

		if !resp.Success {
			return fmt.Errorf("failed: %s", resp.Error)
		}

		fmt.Printf("✓ %s\n", resp.Message)
		return nil
	},
}

func init() {
	// Group for help organization
	iconsCmd.GroupID = "tools"

	rootCmd.AddCommand(iconsCmd)

	// Add maintenance flags
	iconsCmd.Flags().Bool("auto-maintain", false, "Enable automatic maintenance checks")
	iconsCmd.Flags().Bool("auto-fix", false, "Automatically fix issues found (requires --auto-maintain)")
	iconsCmd.Flags().BoolP("verbose", "v", false, "Show detailed maintenance actions")
}
