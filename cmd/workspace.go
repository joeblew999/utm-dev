package cmd

import (
	"fmt"
	"os"

	"github.com/joeblew999/utm-dev/pkg/workspace"
	"github.com/spf13/cobra"
)

// workspaceCmd represents the workspace command
var workspaceCmd = &cobra.Command{
	Use:   "workspace",
	Short: "Manage Go workspace files",
	Long: `Manage Go workspace files by detecting, inspecting, and modifying go.work files.

This command helps you work with Go workspaces in monorepos by automatically finding
go.work files and managing module entries.`,
}

// workspaceInfoCmd shows information about the current workspace
var workspaceInfoCmd = &cobra.Command{
	Use:   "info [path]",
	Short: "Show information about the Go workspace",
	Long: `Display information about the Go workspace, including the path to go.work
and all modules currently in the workspace.

If no path is provided, searches from the current directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		var searchPath string
		if len(args) > 0 {
			searchPath = args[0]
		}

		ws, err := workspace.FindWorkspace(searchPath)
		if err != nil {
			fmt.Printf("Error finding workspace: %v\n", err)
			os.Exit(1)
		}

		if !ws.Exists {
			fmt.Println("No go.work file found")
			return
		}

		fmt.Printf("Workspace: %s\n", ws.FilePath)
		fmt.Printf("Root: %s\n", ws.WorkspaceRoot())
		fmt.Printf("Modules: %d\n", len(ws.Modules))

		if len(ws.Modules) > 0 {
			fmt.Println("\nModules:")
			for _, module := range ws.Modules {
				fmt.Printf("  %s\n", module)
			}
		}
	},
}

// workspaceListCmd lists all modules in the workspace
var workspaceListCmd = &cobra.Command{
	Use:   "list [path]",
	Short: "List all modules in the workspace",
	Long: `List all modules currently included in the Go workspace.

If no path is provided, searches from the current directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		var searchPath string
		if len(args) > 0 {
			searchPath = args[0]
		}

		ws, err := workspace.FindWorkspace(searchPath)
		if err != nil {
			fmt.Printf("Error finding workspace: %v\n", err)
			os.Exit(1)
		}

		if !ws.Exists {
			fmt.Println("No go.work file found")
			return
		}

		modules := ws.ListModules()
		if len(modules) == 0 {
			fmt.Println("No modules in workspace")
			return
		}

		for _, module := range modules {
			fmt.Println(module)
		}
	},
}

// workspaceAddCmd adds a module to the workspace
var workspaceAddCmd = &cobra.Command{
	Use:   "add <module-path>",
	Short: "Add a module to the workspace",
	Long: `Add a module to the Go workspace using 'go work use'.

The module path should be relative to the workspace root.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		modulePath := args[0]
		force, _ := cmd.Flags().GetBool("force")

		ws, err := workspace.FindWorkspace("")
		if err != nil {
			fmt.Printf("Error finding workspace: %v\n", err)
			os.Exit(1)
		}

		if !ws.Exists {
			fmt.Println("No go.work file found")
			os.Exit(1)
		}

		if ws.HasModule(modulePath) {
			fmt.Printf("Module %s already in workspace\n", modulePath)
			return
		}

		err = ws.AddModule(modulePath, force)
		if err != nil {
			fmt.Printf("Error adding module: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Added module %s to workspace\n", modulePath)
	},
}

// workspaceRemoveCmd removes a module from the workspace
var workspaceRemoveCmd = &cobra.Command{
	Use:   "remove <module-path>",
	Short: "Remove a module from the workspace",
	Long: `Remove a module from the Go workspace using 'go work drop'.

The module path should match exactly what's in the go.work file.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		modulePath := args[0]
		force, _ := cmd.Flags().GetBool("force")

		ws, err := workspace.FindWorkspace("")
		if err != nil {
			fmt.Printf("Error finding workspace: %v\n", err)
			os.Exit(1)
		}

		if !ws.Exists {
			fmt.Println("No go.work file found")
			os.Exit(1)
		}

		if !ws.HasModule(modulePath) {
			fmt.Printf("Module %s not in workspace\n", modulePath)
			return
		}

		err = ws.RemoveModule(modulePath, force)
		if err != nil {
			fmt.Printf("Error removing module: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Removed module %s from workspace\n", modulePath)
	},
}

// workspaceCheckCmd checks if a module is in the workspace
var workspaceCheckCmd = &cobra.Command{
	Use:   "check <module-path>",
	Short: "Check if a module is in the workspace",
	Long: `Check if a specific module is included in the Go workspace.

Returns exit code 0 if the module is in the workspace, 1 if not.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		modulePath := args[0]

		ws, err := workspace.FindWorkspace("")
		if err != nil {
			fmt.Printf("Error finding workspace: %v\n", err)
			os.Exit(1)
		}

		if !ws.Exists {
			fmt.Println("No go.work file found")
			os.Exit(1)
		}

		if ws.HasModule(modulePath) {
			fmt.Printf("Module %s is in workspace\n", modulePath)
			os.Exit(0)
		} else {
			fmt.Printf("Module %s is NOT in workspace\n", modulePath)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(workspaceCmd)

	// Add subcommands
	workspaceCmd.AddCommand(workspaceInfoCmd)
	workspaceCmd.AddCommand(workspaceListCmd)
	workspaceCmd.AddCommand(workspaceAddCmd)
	workspaceCmd.AddCommand(workspaceRemoveCmd)
	workspaceCmd.AddCommand(workspaceCheckCmd)

	// Add flags
	workspaceAddCmd.Flags().BoolP("force", "f", false, "Force addition without confirmation")
	workspaceRemoveCmd.Flags().BoolP("force", "f", false, "Force removal without confirmation")
}
