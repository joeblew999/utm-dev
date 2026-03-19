package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joeblew999/utm-dev/pkg/icons"
	"github.com/joeblew999/utm-dev/pkg/project"
	"github.com/spf13/cobra"
)

var generateTestIconCmd = &cobra.Command{
	Use:   "generate-test-icon [app-directory]",
	Short: "Generate a test icon for a Gio project.",
	Long:  `Generate a test icon for a Gio project. If no directory is specified, uses 'example-gio-app'.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Determine app directory
		appDir := "example-gio-app" // Default
		if len(args) > 0 {
			appDir = args[0]
		}

		// Create project instance
		proj, err := project.NewGioProject(appDir)
		if err != nil {
			return fmt.Errorf("failed to create project: %w", err)
		}

		// Generate test icon
		paths := proj.Paths()
		outputPath := paths.GetSourceIcon()

		// Ensure directory exists
		if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		if err := icons.GenerateTestIcon(outputPath); err != nil {
			return fmt.Errorf("failed to generate test icon: %w", err)
		}

		fmt.Printf("Generated test icon: %s\n", outputPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(generateTestIconCmd)
}
