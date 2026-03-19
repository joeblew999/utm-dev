package cmd

import (
	"fmt"
	"os"

	"github.com/joeblew999/utm-dev/pkg/config"
	"github.com/spf13/cobra"
)

var cleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Clean up utm-dev data",
	Long:  `Clean up various utm-dev data directories`,
}

var cleanupAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Remove ALL SDKs and cache (DESTRUCTIVE)",
	Long:  `WARNING: This will remove ALL installed SDKs and cache files. This is a destructive operation that cannot be undone.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sdkDir := config.GetSDKDir()
		cacheDir := config.GetCacheDir()

		fmt.Printf("⚠️  WARNING: This will permanently delete:\n")
		fmt.Printf("   • All SDKs in: %s\n", sdkDir)
		fmt.Printf("   • All cache in: %s\n", cacheDir)
		fmt.Printf("\nThis action cannot be undone. Continue? (y/N): ")

		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cleanup cancelled.")
			return nil
		}

		fmt.Printf("Removing SDKs directory: %s\n", sdkDir)
		fmt.Printf("Removing cache directory: %s\n", cacheDir)

		if err := config.CleanDirectories(); err != nil {
			return fmt.Errorf("cleanup failed: %w", err)
		}

		fmt.Println("✓ Complete cleanup finished.")
		return nil
	},
}

var cleanupCacheCmd = &cobra.Command{
	Use:   "cache-only",
	Short: "Remove only cache files (keeps SDKs)",
	Long:  `Removes only the cache directory, keeping all installed SDKs intact.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cacheDir := config.GetCacheDir()
		fmt.Printf("Removing cache directory: %s\n", cacheDir)

		if _, err := os.Stat(cacheDir); err == nil {
			if err := os.RemoveAll(cacheDir); err != nil {
				return fmt.Errorf("failed to remove cache directory: %w", err)
			}
		}

		fmt.Println("✓ Cache cleanup complete.")
		return nil
	},
}

var cleanupSDKsCmd = &cobra.Command{
	Use:   "sdks-only",
	Short: "Remove only SDKs (keeps cache)",
	Long:  `WARNING: Removes only the SDK directory, keeping cache files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		sdkDir := config.GetSDKDir()

		fmt.Printf("⚠️  WARNING: This will permanently delete all SDKs in: %s\n", sdkDir)
		fmt.Printf("Continue? (y/N): ")

		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cleanup cancelled.")
			return nil
		}

		fmt.Printf("Removing SDKs directory: %s\n", sdkDir)

		if _, err := os.Stat(sdkDir); err == nil {
			if err := os.RemoveAll(sdkDir); err != nil {
				return fmt.Errorf("failed to remove SDK directory: %w", err)
			}
		}

		fmt.Println("✓ SDKs cleanup complete.")
		return nil
	},
}

func init() {
	// Add the main cleanup command with subcommands
	cleanupCmd.AddCommand(cleanupAllCmd)
	cleanupCmd.AddCommand(cleanupCacheCmd)
	cleanupCmd.AddCommand(cleanupSDKsCmd)
	rootCmd.AddCommand(cleanupCmd)

	// Keep the old command for backward compatibility but mark as deprecated
	var deprecatedCleanupCacheCmd = &cobra.Command{
		Use:        "cleanup-cache",
		Short:      "Remove all downloaded SDKs and the cache",
		Long:       `Removes the SDK directory and the cache directory.`,
		Deprecated: "Use 'cleanup all' instead",
		Hidden:     true,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("⚠️  WARNING: This command is deprecated. Use 'utm-dev cleanup all' instead.")
			return cleanupAllCmd.RunE(cmd, args)
		},
	}
	rootCmd.AddCommand(deprecatedCleanupCacheCmd)
}
