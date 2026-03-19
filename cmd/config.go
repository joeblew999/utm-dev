package cmd

import (
	"fmt"
	"os"

	"github.com/joeblew999/utm-dev/pkg/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show configuration and directory information",
	Long:  "Display configuration details including SDK installation paths, cache locations, and current setup.",
	Run:   runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) {
	info := config.GetDirectoryInfo()

	fmt.Println("=== utm-dev Configuration ===")
	fmt.Println()
	
	fmt.Println("📁 Directory Locations:")
	fmt.Printf("  Cache Directory: %s\n", info.CacheDir)
	fmt.Printf("  SDK Directory:   %s\n", info.SDKDir)
	fmt.Println()
	
	fmt.Println("📊 Directory Status:")
	fmt.Printf("  Cache exists: %t\n", info.CacheExists)
	fmt.Printf("  SDKs exist:   %t\n", info.SDKExists)
	
	if info.CacheSize > 0 {
		fmt.Printf("  Cache size:   %s\n", formatBytes(info.CacheSize))
	}
	if info.SDKSize > 0 {
		fmt.Printf("  SDKs size:    %s\n", formatBytes(info.SDKSize))
	}
	fmt.Println()
	
	fmt.Println("🔧 Platform Information:")
	fmt.Printf("  OS:           %s\n", os.Getenv("GOOS"))
	fmt.Printf("  Architecture: %s\n", os.Getenv("GOARCH"))
	fmt.Println()
	
	// Show actual directory contents if they exist
	fmt.Println("📂 Current Contents:")
	showDirectoryContents(info.SDKDir, "SDKs")
	showDirectoryContents(info.CacheDir, "Cache")
	
	// Show environment variables that might be relevant
	fmt.Println("🌍 Relevant Environment Variables:")
	showEnvVars := []string{"JAVA_HOME", "ANDROID_HOME", "ANDROID_SDK_ROOT", "XCODE_PATH"}
	for _, envVar := range showEnvVars {
		if value := os.Getenv(envVar); value != "" {
			fmt.Printf("  %s: %s\n", envVar, value)
		} else {
			fmt.Printf("  %s: (not set)\n", envVar)
		}
	}
}

func showDirectoryContents(path, label string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("  %s: Directory does not exist\n", label)
		return
	}
	
	entries, err := os.ReadDir(path)
	if err != nil {
		fmt.Printf("  %s: Error reading directory: %v\n", label, err)
		return
	}
	
	if len(entries) == 0 {
		fmt.Printf("  %s: Directory is empty\n", label)
		return
	}
	
	fmt.Printf("  %s:\n", label)
	for _, entry := range entries {
		if entry.IsDir() {
			fmt.Printf("    📁 %s/\n", entry.Name())
		} else {
			fmt.Printf("    📄 %s\n", entry.Name())
		}
	}
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}