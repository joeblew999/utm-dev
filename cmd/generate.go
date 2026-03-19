package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate project artifacts (docs, etc.)",
	Long: `Generate various project artifacts.

This command is used internally to keep generated files up-to-date.
It's typically run as part of CI/CD or release preparation.

Examples:
  # Generate everything
  utm-dev generate all

  # Generate only documentation
  utm-dev generate docs`,
}

var generateAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Generate all artifacts (docs, etc.)",
	Long: `Generate all project artifacts.

This runs all generation tasks:
- CLI documentation (markdown)

Examples:
  utm-dev generate all
  utm-dev generate all --output-dir ./custom-docs`,
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir, _ := cmd.Flags().GetString("output-dir")

		fmt.Println("=== Generate All Artifacts ===")
		fmt.Println()

		// Generate docs
		fmt.Println("1. Generating CLI documentation...")
		if err := generateDocs(outputDir); err != nil {
			return fmt.Errorf("failed to generate docs: %w", err)
		}

		fmt.Println()
		fmt.Println("=== All artifacts generated successfully! ===")
		return nil
	},
}

var generateDocsCmd = &cobra.Command{
	Use:   "docs [output-dir]",
	Short: "Generate CLI documentation",
	Long: `Generate CLI documentation in markdown format.

This generates documentation for all utm-dev commands.
Output is written to docs/cli/ by default.

Examples:
  utm-dev generate docs
  utm-dev generate docs ./my-docs`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outputDir := "docs/cli"
		if len(args) > 0 {
			outputDir = args[0]
		}

		return generateDocs(outputDir)
	},
}

func generateDocs(outputDir string) error {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	fmt.Printf("   Generating markdown to %s/\n", outputDir)

	if err := doc.GenMarkdownTree(rootCmd, outputDir); err != nil {
		return fmt.Errorf("failed to generate markdown: %w", err)
	}

	// Count generated files
	files, _ := filepath.Glob(filepath.Join(outputDir, "*.md"))
	fmt.Printf("   ✓ Generated %d documentation files\n", len(files))

	return nil
}

func init() {
	generateAllCmd.Flags().String("output-dir", "docs/cli", "Output directory for generated docs")

	generateCmd.AddCommand(generateAllCmd)
	generateCmd.AddCommand(generateDocsCmd)

	// Group for help organization
	generateCmd.GroupID = "tools"

	rootCmd.AddCommand(generateCmd)
}
