package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/joeblew999/utm-dev/pkg/config"
	"github.com/joeblew999/utm-dev/pkg/installer"
	"github.com/joeblew999/utm-dev/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	platformFilter string
	categoryFilter string
	compactOutput  bool
	cachedOnly     bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available SDKs",
	Long:  "List all available Android and iOS SDKs from the JSON files, or show cached SDKs.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if cachedOnly {
			return listCachedSDKs()
		}
		return listAllSDKs()
	},
}

func init() {
	listCmd.Flags().StringVarP(&platformFilter, "platform", "p", "", "Filter by platform (android, ios)")
	listCmd.Flags().StringVarP(&categoryFilter, "category", "c", "", "Filter by category (e.g., openjdk, android, build-tools)")
	listCmd.Flags().BoolVar(&compactOutput, "compact", false, "Show compact output without categories")
	listCmd.Flags().BoolVar(&cachedOnly, "cached", false, "Show only cached SDKs from download cache")

	// Alias and group
	listCmd.Aliases = []string{"ls"}
	listCmd.GroupID = "sdk"

	rootCmd.AddCommand(listCmd)
}

func listAllSDKs() error {
	sdkFiles, err := utils.ParseSDKFiles()
	if err != nil {
		return fmt.Errorf("failed to parse SDK files: %w", err)
	}

	// Platform names corresponding to SDK files (order must match ParseSDKFiles)
	platforms := []string{"android", "ios", "build-tools"}

	// Filter by platform if specified
	if platformFilter != "" {
		platformFilter = strings.ToLower(platformFilter)
		if !utils.Contains(platforms, platformFilter) {
			return fmt.Errorf("unknown platform: %s (available: android, ios, build-tools)", platformFilter)
		}
		// Find the index of the filtered platform
		for i, p := range platforms {
			if p == platformFilter {
				platforms = []string{p}
				sdkFiles = []config.SdkFile{sdkFiles[i]}
				break
			}
		}
	}

	for i, sdkFile := range sdkFiles {
		platform := platforms[i]
		if !compactOutput {
			fmt.Printf("=== %s SDKs ===\n", strings.Title(platform))
		}
		err := listSDKsFromSDKFile(sdkFile, platform)
		if err != nil {
			fmt.Printf("Error listing SDKs from %s list: %s\n", platform, err)
		}
		if !compactOutput {
			fmt.Println()
		}
	}

	return nil
}

func listSDKsFromSDKFile(sdkFile config.SdkFile, platform string) error {

	// Sort categories for consistent output
	var categories []string
	for category := range sdkFile.SDKs {
		// Filter by category if specified
		if categoryFilter != "" && !strings.Contains(strings.ToLower(category), strings.ToLower(categoryFilter)) {
			continue
		}
		categories = append(categories, category)
	}
	sort.Strings(categories)

	for _, category := range categories {
		sdkItems := sdkFile.SDKs[category]
		if len(sdkItems) == 0 {
			continue
		}

		if !compactOutput {
			fmt.Printf("📦 %s\n", strings.Title(category))
		}

		for _, item := range sdkItems {
			name := item.GoupName
			if name == "" {
				// This is for android system images
				name = fmt.Sprintf("system-image;api-%d;%s;%s", item.ApiLevel, item.Vendor, item.Abi)
			}

			if compactOutput {
				fmt.Printf("%s\n", name)
			} else {
				version := item.Version
				if version == "" {
					version = "latest"
				}
				fmt.Printf("   • %s (v%s)\n", name, version)
			}
		}

		if !compactOutput {
			fmt.Println()
		}
	}

	return nil
}

func listCachedSDKs() error {
	cache, err := utils.NewCacheWithDirectories()
	if err != nil {
		return fmt.Errorf("failed to load cache: %w", err)
	}

	if len(cache.Entries) == 0 {
		fmt.Println("No SDKs cached yet. Use 'utm-dev install' or 'utm-dev setup' to download SDKs.")
		return nil
	}

	// Sort entries by name for consistent output
	var entries []installer.CacheEntry
	for _, entry := range cache.Entries {
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name < entries[j].Name
	})

	if !compactOutput {
		fmt.Printf("=== Cached SDKs (%d) ===\n", len(entries))
	}

	for _, entry := range entries {
		if compactOutput {
			fmt.Printf("%s\n", entry.Name)
		} else {
			version := entry.Version
			if version == "" {
				version = "unknown"
			}

			// Check if the SDK is still present on disk
			status := "✓ present"
			if resolved, err := installer.ResolveInstallPath(entry.InstallPath); err == nil {
				if _, err := os.Stat(resolved); os.IsNotExist(err) {
					status = "✗ missing"
				}
			}

			fmt.Printf("   • %s (v%s) - %s\n", entry.Name, version, status)
		}
	}

	if !compactOutput {
		fmt.Printf("\nCache location: %s\n", config.GetCachePath())
		fmt.Println("\nNote: 'cached' means downloaded to local cache. 'present' means files exist on disk.")
	}

	return nil
}
