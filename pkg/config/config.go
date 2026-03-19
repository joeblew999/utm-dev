package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

//go:embed sdk-android-list.json
var AndroidSdkList []byte

//go:embed sdk-ios-list.json
var IosSdkList []byte

//go:embed sdk-build-tools.json
var BuildToolsSdkList []byte

// Platform defines the structure for platform-specific SDK details.
type Platform struct {
	DownloadURL string `json:"downloadUrl"`
	Checksum    string `json:"checksum"`
}

// SdkItem defines the structure for an SDK entry in the JSON file.
type SdkItem struct {
	Version        string              `json:"version"`
	GoupName       string              `json:"goupName"`
	DownloadURL    string              `json:"downloadUrl,omitempty"`
	Checksum       string              `json:"checksum,omitempty"`
	InstallPath    string              `json:"installPath"`
	ApiLevel       int                 `json:"apiLevel"`
	Abi            string              `json:"abi"`
	Vendor         string              `json:"vendor"`
	Platforms      map[string]Platform `json:"platforms,omitempty"`
	SdkManagerName string              `json:"sdkmanagerName,omitempty"`
}

// SdkFile defines the top-level structure of the JSON file.
type SdkFile struct {
	SDKs map[string][]SdkItem `json:"sdks"`
}

// MetaFile defines the structure for setup configurations
type MetaFile struct {
	Meta struct {
		Setups map[string][]string `json:"setups"`
	} `json:"meta"`
}

// AndroidBuildDefaults contains build configuration for Android
type AndroidBuildDefaults struct {
	MinSdk    int `json:"minSdk"`
	TargetSdk int `json:"targetSdk"`
}

// IOSBuildDefaults contains build configuration for iOS
type IOSBuildDefaults struct {
	MinOS string `json:"minOS"`
}

// AndroidSdkFile defines the structure for android SDK list with meta
type AndroidSdkFile struct {
	SDKs map[string][]SdkItem `json:"sdks"`
	Meta struct {
		SchemaVersion string               `json:"schemaVersion"`
		BuildDefaults AndroidBuildDefaults `json:"buildDefaults"`
		Setups        map[string][]string  `json:"setups"`
	} `json:"meta"`
}

// IOSSdkFile defines the structure for iOS SDK list with meta
type IOSSdkFile struct {
	SDKs map[string][]SdkItem `json:"sdks"`
	Meta struct {
		SchemaVersion string           `json:"schemaVersion"`
		BuildDefaults IOSBuildDefaults `json:"buildDefaults"`
		Setups        map[string][]string  `json:"setups"`
	} `json:"meta"`
}

// GetAndroidBuildDefaults returns the Android build defaults from config
func GetAndroidBuildDefaults() AndroidBuildDefaults {
	var sdkFile AndroidSdkFile
	if err := json.Unmarshal(AndroidSdkList, &sdkFile); err != nil {
		// Return sensible defaults if parsing fails
		return AndroidBuildDefaults{MinSdk: 21, TargetSdk: 34}
	}
	return sdkFile.Meta.BuildDefaults
}

// GetIOSBuildDefaults returns the iOS build defaults from config
func GetIOSBuildDefaults() IOSBuildDefaults {
	var sdkFile IOSSdkFile
	if err := json.Unmarshal(IosSdkList, &sdkFile); err != nil {
		// Return sensible defaults if parsing fails
		return IOSBuildDefaults{MinOS: "15.0"}
	}
	return sdkFile.Meta.BuildDefaults
}

// GetAndroidMinSdk returns the Android minimum SDK as a string for gogio
func GetAndroidMinSdk() string {
	defaults := GetAndroidBuildDefaults()
	return strconv.Itoa(defaults.MinSdk)
}

// GetIOSMinOS returns the iOS minimum OS version as a string for gogio
func GetIOSMinOS() string {
	defaults := GetIOSBuildDefaults()
	if defaults.MinOS == "" {
		return "15.0"
	}
	// Strip the minor version if present (e.g., "15.0" -> "15")
	// gogio expects just the major version number
	for i, c := range defaults.MinOS {
		if c == '.' {
			return defaults.MinOS[:i]
		}
	}
	return defaults.MinOS
}

// GetCacheDir returns the OS-appropriate cache directory for utm-dev
func GetCacheDir() string {
	switch runtime.GOOS {
	case "darwin": // macOS
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, "utm-dev-cache")
		}
	case "linux":
		if cacheHome := os.Getenv("XDG_CACHE_HOME"); cacheHome != "" {
			return filepath.Join(cacheHome, "utm-dev")
		}
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, ".cache", "utm-dev")
		}
	case "windows":
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, "utm-dev")
		}
	}

	// Fallback to the old behavior if we can't determine OS-specific path
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".utm-dev")
	}

	// Last resort fallback
	return ".utm-dev"
}

// GetSDKDir returns the OS-appropriate SDK storage directory for utm-dev
func GetSDKDir() string {
	switch runtime.GOOS {
	case "darwin": // macOS
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, "utm-dev-sdks")
		}
	case "linux":
		if dataHome := os.Getenv("XDG_DATA_HOME"); dataHome != "" {
			return filepath.Join(dataHome, "utm-dev", "sdks")
		}
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, ".local", "share", "utm-dev", "sdks")
		}
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "utm-dev", "sdks")
		}
	}

	// Fallback - use cache dir + sdks subdirectory
	return filepath.Join(GetCacheDir(), "sdks")
}

// GetCachePath returns the full path to the cache.json file
func GetCachePath() string {
	return filepath.Join(GetCacheDir(), "cache.json")
}

// DirectoryInfo contains information about utm-dev directories
type DirectoryInfo struct {
	CacheDir    string `json:"cache_dir"`
	SDKDir      string `json:"sdk_dir"`
	CacheExists bool   `json:"cache_exists"`
	SDKExists   bool   `json:"sdk_exists"`
	CacheSize   int64  `json:"cache_size,omitempty"`
	SDKSize     int64  `json:"sdk_size,omitempty"`
}

// EnsureDirectories creates cache and SDK directories if they don't exist
func EnsureDirectories() error {
	cacheDir := GetCacheDir()
	sdkDir := GetSDKDir()

	// Create cache directory
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory %s: %w", cacheDir, err)
	}

	// Create SDK directory
	if err := os.MkdirAll(sdkDir, 0755); err != nil {
		return fmt.Errorf("failed to create SDK directory %s: %w", sdkDir, err)
	}

	return nil
}

// CleanDirectories removes all cache and SDK directories
func CleanDirectories() error {
	var errors []error

	// Remove SDK directory
	sdkDir := GetSDKDir()
	if _, err := os.Stat(sdkDir); err == nil {
		if err := os.RemoveAll(sdkDir); err != nil {
			errors = append(errors, fmt.Errorf("failed to remove SDK directory %s: %w", sdkDir, err))
		}
	}

	// Remove cache directory
	cacheDir := GetCacheDir()
	if _, err := os.Stat(cacheDir); err == nil {
		if err := os.RemoveAll(cacheDir); err != nil {
			errors = append(errors, fmt.Errorf("failed to remove cache directory %s: %w", cacheDir, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	return nil
}

// CleanCache removes only the cache directory
func CleanCache() error {
	cacheDir := GetCacheDir()
	if _, err := os.Stat(cacheDir); err == nil {
		if err := os.RemoveAll(cacheDir); err != nil {
			return fmt.Errorf("failed to remove cache directory %s: %w", cacheDir, err)
		}
	}
	return nil
}

// GetDirectoryInfo returns size and health information about directories
func GetDirectoryInfo() DirectoryInfo {
	cacheDir := GetCacheDir()
	sdkDir := GetSDKDir()

	info := DirectoryInfo{
		CacheDir: cacheDir,
		SDKDir:   sdkDir,
	}

	// Check if directories exist and get sizes
	if stat, err := os.Stat(cacheDir); err == nil && stat.IsDir() {
		info.CacheExists = true
		if size, err := getDirSize(cacheDir); err == nil {
			info.CacheSize = size
		}
	}

	if stat, err := os.Stat(sdkDir); err == nil && stat.IsDir() {
		info.SDKExists = true
		if size, err := getDirSize(sdkDir); err == nil {
			info.SDKSize = size
		}
	}

	return info
}

// getDirSize calculates the total size of a directory
func getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}
