package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joeblew999/utm-dev/pkg/config"
	"github.com/joeblew999/utm-dev/pkg/installer"
)

// utils.go provides centralized utility functions to reduce code duplication
// and maintain DRY principles across the utm-dev codebase.

// Platform definitions - centralized list of supported platforms
var (
	IconPlatforms  = []string{"android", "ios", "macos", "windows", "windows-msix", "windows-ico"}
	BuildPlatforms = []string{"macos", "android", "ios", "ios-simulator", "windows", "all"}
)

// Contains checks if a slice contains a specific item
func Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// ValidatePlatform validates a platform against a list of valid platforms
func ValidatePlatform(platform string, validPlatforms []string) error {
	if !Contains(validPlatforms, platform) {
		return fmt.Errorf("invalid platform: %s. Valid platforms: %v", platform, validPlatforms)
	}
	return nil
}

// NewCacheWithDirectories creates cache and ensures directories exist
func NewCacheWithDirectories() (*installer.Cache, error) {
	if err := config.EnsureDirectories(); err != nil {
		return nil, fmt.Errorf("could not create directories: %w", err)
	}

	cache, err := installer.NewCache(config.GetCachePath())
	if err != nil {
		return nil, fmt.Errorf("could not load cache: %w", err)
	}

	return cache, nil
}

// ParseSDKFiles parses all SDK files (Android, iOS, Build Tools) and returns them
func ParseSDKFiles() ([]config.SdkFile, error) {
	sdkFileContents := [][]byte{config.AndroidSdkList, config.IosSdkList, config.BuildToolsSdkList}
	var sdkFiles []config.SdkFile

	for _, sdkFileContent := range sdkFileContents {
		var sdkFile config.SdkFile
		err := json.Unmarshal(sdkFileContent, &sdkFile)
		if err != nil {
			// Skip malformed files but continue processing
			continue
		}
		sdkFiles = append(sdkFiles, sdkFile)
	}

	return sdkFiles, nil
}

// ParseMetaFiles parses all SDK files (Android, iOS, Build Tools) as MetaFiles for setup functionality
func ParseMetaFiles() ([]config.MetaFile, error) {
	sdkFileContents := [][]byte{config.AndroidSdkList, config.IosSdkList, config.BuildToolsSdkList}
	var metaFiles []config.MetaFile

	for _, sdkFileContent := range sdkFileContents {
		var metaFile config.MetaFile
		err := json.Unmarshal(sdkFileContent, &metaFile)
		if err != nil {
			// Skip malformed files but continue processing
			continue
		}
		metaFiles = append(metaFiles, metaFile)
	}

	return metaFiles, nil
}

// FindSetup searches for a setup by name across all SDK files
func FindSetup(setupName string) ([]string, error) {
	metaFiles, err := ParseMetaFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to parse meta files: %w", err)
	}

	for _, metaFile := range metaFiles {
		if setup, ok := metaFile.Meta.Setups[setupName]; ok {
			return setup, nil
		}
	}

	return nil, fmt.Errorf("setup '%s' not found in any sdk list", setupName)
}

// FindSDKItem searches for an SDK item by name across all SDK files
func FindSDKItem(sdkName string) (*config.SdkItem, error) {
	sdkFiles, err := ParseSDKFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to parse SDK files: %w", err)
	}

	for _, sdkFile := range sdkFiles {
		for _, sdkItems := range sdkFile.SDKs {
			for _, item := range sdkItems {
				var currentSdkName string
				if item.GoupName != "" {
					currentSdkName = item.GoupName
				} else if item.ApiLevel > 0 {
					currentSdkName = fmt.Sprintf("system-image;api-%d;%s;%s", item.ApiLevel, item.Vendor, item.Abi)
				}

				if currentSdkName == sdkName {
					return &item, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("SDK '%s' not found", sdkName)
}

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// EnsureDirForFile creates the parent directory for a file if it doesn't exist
func EnsureDirForFile(filePath string) error {
	return os.MkdirAll(filepath.Dir(filePath), 0755)
}
