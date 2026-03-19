package cmd

import (
	"fmt"
	"os"

	"github.com/joeblew999/utm-dev/pkg/service"
	"github.com/spf13/cobra"
)

// createExampleCmd represents the create-example command
var createExampleCmd = &cobra.Command{
	Use:   "create-example <name>",
	Short: "Create a new example project",
	Long: `Create a new example project with proper structure and optionally add it to the workspace.

This command creates a new example in the examples/ directory and can automatically
update the go.work file to include the new module.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		exampleName := args[0]
		updateWorkspace, _ := cmd.Flags().GetBool("workspace")
		force, _ := cmd.Flags().GetBool("force")

		svc := service.NewGioService()

		req := service.CreateExampleRequest{
			ExampleName:          exampleName,
			UpdateWorkspace:      updateWorkspace,
			ForceWorkspaceUpdate: force,
		}

		resp, err := svc.CreateExample(req)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		if !resp.Success {
			fmt.Printf("Failed: %s\n", resp.Error)
			os.Exit(1)
		}

		fmt.Printf("✅ %s\n", resp.Message)
		if updateWorkspace {
			fmt.Println("📝 Workspace updated")
		}
	},
}

// ensureWorkspaceCmd ensures a project is in the workspace
var ensureWorkspaceCmd = &cobra.Command{
	Use:   "ensure-workspace <module-path>",
	Short: "Ensure a module is included in the workspace",
	Long: `Ensure that a specific module is included in the Go workspace.

The module path should be relative to the workspace root.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		modulePath := args[0]
		force, _ := cmd.Flags().GetBool("force")

		svc := service.NewGioService()

		req := service.WorkspaceRequest{
			ModulePath: modulePath,
			Force:      force,
		}

		resp, err := svc.EnsureInWorkspace(req)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		if !resp.Success {
			fmt.Printf("Failed: %s\n", resp.Error)
			if !force {
				fmt.Println("💡 Use --force to automatically add to workspace")
			}
			os.Exit(1)
		}

		fmt.Printf("✅ %s\n", resp.Message)
	},
}

func init() {
	rootCmd.AddCommand(createExampleCmd)
	rootCmd.AddCommand(ensureWorkspaceCmd)

	// Flags for create-example
	createExampleCmd.Flags().BoolP("workspace", "w", false, "Update workspace to include new example")
	createExampleCmd.Flags().BoolP("force", "f", false, "Force workspace update without confirmation")

	// Flags for ensure-workspace
	ensureWorkspaceCmd.Flags().BoolP("force", "f", false, "Force addition to workspace")
}
