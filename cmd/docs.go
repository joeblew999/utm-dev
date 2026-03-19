package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var docsCmd = &cobra.Command{
	Use:   "docs [output-dir]",
	Short: "Generate CLI documentation",
	Long: `Generate documentation for all utm-dev commands.

Outputs markdown files that can be used in README or documentation sites.

Examples:
  # Generate docs to ./docs/cli/
  utm-dev docs

  # Generate to custom directory
  utm-dev docs ./my-docs/

  # Generate man pages
  utm-dev docs --format man ./man/`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir := "docs/cli"
		if len(args) > 0 {
			outputDir = args[0]
		}

		format, _ := cmd.Flags().GetString("format")

		// Create output directory
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		fmt.Printf("Generating %s documentation to %s/\n", format, outputDir)

		switch format {
		case "markdown", "md":
			if err := doc.GenMarkdownTree(rootCmd, outputDir); err != nil {
				return fmt.Errorf("failed to generate markdown: %w", err)
			}
		case "man":
			header := &doc.GenManHeader{
				Title:   "GOUP-UTIL",
				Section: "1",
			}
			if err := doc.GenManTree(rootCmd, header, outputDir); err != nil {
				return fmt.Errorf("failed to generate man pages: %w", err)
			}
		case "yaml":
			if err := doc.GenYamlTree(rootCmd, outputDir); err != nil {
				return fmt.Errorf("failed to generate YAML: %w", err)
			}
		case "rst":
			if err := doc.GenReSTTree(rootCmd, outputDir); err != nil {
				return fmt.Errorf("failed to generate RST: %w", err)
			}
		default:
			return fmt.Errorf("unknown format: %s (use: markdown, man, yaml, rst)", format)
		}

		// Count generated files
		files, _ := filepath.Glob(filepath.Join(outputDir, "*"))
		fmt.Printf("✓ Generated %d documentation files\n", len(files))

		return nil
	},
}

func init() {
	docsCmd.Flags().StringP("format", "f", "markdown", "Output format: markdown, man, yaml, rst")

	// Group for help organization
	docsCmd.GroupID = "tools"

	rootCmd.AddCommand(docsCmd)
}
