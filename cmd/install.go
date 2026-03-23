package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/joeblew999/utm-dev/pkg/config"
	"github.com/joeblew999/utm-dev/pkg/installer"
	"github.com/joeblew999/utm-dev/pkg/utils"
	"github.com/spf13/cobra"
)

const cmdLineTools = "cmdline-tools-11.0"

var installCmd = &cobra.Command{
	Use:   "install [sdk-name]",
	Short: "Install an SDK",
	Long:  `Install a specified Android or iOS SDK.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sdkName := args[0]

		// Ensure directories exist and create cache
		cache, err := utils.NewCacheWithDirectories()
		if err != nil {
			return err
		}

		fmt.Printf("Installing SDK: %s...\n", sdkName)

		return installSdk(sdkName, cache)
	},
}

func init() {
	// Group for help organization
	installCmd.GroupID = "util"

	rootCmd.AddCommand(installCmd)
}

func installSdk(sdkName string, cache *installer.Cache) error {
	// Special case for garble - uses go install
	if sdkName == "garble" {
		return installer.InstallGarble(cache)
	}

	sdk, sdkManagerName, err := findSdk(sdkName)
	if err != nil {
		return err
	}

	if sdkManagerName != "" {
		return installWithSdkManager(sdk, sdkManagerName, cache)
	}

	return installer.Install(sdk, cache)
}

func getJavaHome(cache *installer.Cache) (string, error) {
	jdkEntry, ok := cache.Entries["openjdk-17"]
	if !ok {
		return "", nil // Not an error, just not installed
	}

	jfrPath, err := installer.ResolveInstallPath(jdkEntry.InstallPath)
	if err != nil {
		return "", fmt.Errorf("could not resolve openjdk-17 install path: %w", err)
	}

	files, err := os.ReadDir(jfrPath)
	if err != nil {
		return "", fmt.Errorf("could not read openjdk-17 install directory: %w", err)
	}

	for _, f := range files {
		if !f.IsDir() {
			continue
		}

		var homePath string
		if runtime.GOOS == "darwin" {
			homePath = filepath.Join(jfrPath, f.Name(), "Contents", "Home")
		} else {
			homePath = filepath.Join(jfrPath, f.Name())
		}

		if _, err := os.Stat(homePath); err == nil {
			return homePath, nil
		}
	}

	return "", fmt.Errorf("could not find a valid JAVA_HOME in %s", jfrPath)
}

func installWithSdkManager(sdk *installer.SDK, sdkManagerName string, cache *installer.Cache) error {
	// Check if SDK is already installed and directory exists
	if entry, ok := cache.Entries[sdk.Name]; ok {
		installPath, err := installer.ResolveInstallPath(entry.InstallPath)
		if err == nil {
			if _, err := os.Stat(installPath); err == nil {
				fmt.Printf("%s is already installed.\n", sdk.Name)
				return nil
			}
		}
		fmt.Printf("%s cache entry found, but directory is missing or path is invalid. Re-installing.\n", sdk.Name)
	}

	// Ensure openjdk-17 is installed for sdkmanager
	if _, ok := cache.Entries["openjdk-17"]; !ok {
		fmt.Println("openjdk-17 not found in cache, installing for sdkmanager...")
		if err := installSdk("openjdk-17", cache); err != nil {
			return fmt.Errorf("failed to install openjdk-17 for sdkmanager: %w", err)
		}
	}

	fmt.Println("Checking for command-line tools...")
	cmdToolsEntry, ok := cache.Entries[cmdLineTools]
	if !ok {
		fmt.Println("Command-line tools not found, installing them first...")
		if err := installSdk(cmdLineTools, cache); err != nil {
			return fmt.Errorf("failed to install command-line tools: %w", err)
		}
		cmdToolsEntry, ok = cache.Entries[cmdLineTools]
		if !ok {
			return fmt.Errorf("could not find command-line tools in cache even after installation")
		}
	}

	fmt.Println("Command-line tools found.")
	cmdToolsPath, err := installer.ResolveInstallPath(cmdToolsEntry.InstallPath)
	if err != nil {
		return fmt.Errorf("could not resolve command-line tools install path: %w", err)
	}

	if !filepath.IsAbs(cmdToolsPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("could not get current working directory: %w", err)
		}
		cmdToolsPath = filepath.Join(cwd, cmdToolsPath)
	}

	sdksRoot := config.GetSDKDir()
	sdkManagerPath := filepath.Join(cmdToolsPath, "cmdline-tools", "bin", "sdkmanager")

	// Set JAVA_HOME for sdkmanager
	javaHome, err := getJavaHome(cache)
	if err != nil {
		return err
	}

	fmt.Printf("Using sdkmanager at: %s\n", sdkManagerPath)
	fmt.Printf("Using SDK root at: %s\n", sdksRoot)

	cmd := exec.Command(sdkManagerPath, fmt.Sprintf("--sdk_root=%s", sdksRoot), sdkManagerName)
	if javaHome != "" {
		cmd.Env = append(os.Environ(), "JAVA_HOME="+javaHome)
		fmt.Printf("Setting JAVA_HOME for sdkmanager to: %s\n", javaHome)
	}
	cmd.Stdin = strings.NewReader(strings.Repeat("y\n", 10))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run sdkmanager for %s: %w", sdkManagerName, err)
	}

	fmt.Printf("%s installed successfully.\n", sdk.Name)

	// After successful installation, update the cache with the correct install path
	// This is crucial for sdkmanager-installed items where the path is not known beforehand
	var installPath string
	if sdk.InstallPath != "" {
		installPath = sdk.InstallPath
	} else {
		// For items like platforms, system-images, the path is constructed
		// based on the sdkmanagerName.
		// e.g., "platforms;android-31" -> "platforms/android-31"
		installPath = filepath.Join(config.GetSDKDir(), strings.ReplaceAll(sdkManagerName, ";", "/"))
	}

	cache.Entries[sdk.Name] = installer.CacheEntry{
		Name:        sdk.Name,
		Version:     sdk.Version,
		InstallPath: installPath,
	}

	return cache.Save()
}

func findSdk(sdkName string) (*installer.SDK, string, error) {
	item, err := utils.FindSDKItem(sdkName)
	if err != nil {
		return nil, "", err
	}

	var platform config.Platform
	var ok bool

	if item.Platforms != nil {
		platform, ok = item.Platforms[runtime.GOOS+"/"+runtime.GOARCH]
		if !ok {
			// Fallback to just GOOS if the specific arch is not found
			platform, ok = item.Platforms[runtime.GOOS]
		}
	}

	// If platform-specific URL/Checksum is not available, use the general one.
	downloadURL := item.DownloadURL
	checksum := item.Checksum
	if ok {
		downloadURL = platform.DownloadURL
		checksum = platform.Checksum
	}

	var currentSdkName string
	if item.GoupName != "" {
		currentSdkName = item.GoupName
	} else if item.ApiLevel > 0 {
		currentSdkName = fmt.Sprintf("system-image;api-%d;%s;%s", item.ApiLevel, item.Vendor, item.Abi)
	}

	return &installer.SDK{
		Name:        currentSdkName,
		Version:     item.Version,
		URL:         downloadURL,
		Checksum:    checksum,
		InstallPath: item.InstallPath,
	}, item.SdkManagerName, nil
}
