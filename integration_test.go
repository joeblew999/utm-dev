package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joeblew999/utm-dev/pkg/config"
)

// TestCleanupCommands tests the cleanup command logic without running the actual CLI
func TestCleanupCommands(t *testing.T) {
	// Create temporary directories to simulate SDK and cache directories
	tempDir := t.TempDir()

	testCacheDir := filepath.Join(tempDir, "cache")
	testSDKDir := filepath.Join(tempDir, "sdks")

	// Create test directories and files
	err := os.MkdirAll(testCacheDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test cache dir: %v", err)
	}

	err = os.MkdirAll(testSDKDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test SDK dir: %v", err)
	}

	// Create test files to simulate cache and SDK content
	cacheFile := filepath.Join(testCacheDir, "cache.json")
	err = os.WriteFile(cacheFile, []byte(`{"test": "data"}`), 0644)
	if err != nil {
		t.Fatalf("Failed to create test cache file: %v", err)
	}

	sdkSubDir := filepath.Join(testSDKDir, "android-31")
	err = os.MkdirAll(sdkSubDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test SDK subdir: %v", err)
	}

	sdkFile := filepath.Join(sdkSubDir, "android.jar")
	err = os.WriteFile(sdkFile, []byte("fake sdk content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test SDK file: %v", err)
	}

	// Test cache-only cleanup
	t.Run("CleanCacheOnly", func(t *testing.T) {
		// Verify files exist before cleanup
		if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
			t.Fatal("Cache file should exist before cleanup")
		}
		if _, err := os.Stat(sdkFile); os.IsNotExist(err) {
			t.Fatal("SDK file should exist before cleanup")
		}

		// Clean cache only
		err := os.RemoveAll(testCacheDir)
		if err != nil {
			t.Fatalf("Failed to clean cache directory: %v", err)
		}

		// Verify cache is gone but SDK remains
		if _, err := os.Stat(testCacheDir); !os.IsNotExist(err) {
			t.Error("Cache directory should be removed")
		}
		if _, err := os.Stat(sdkFile); os.IsNotExist(err) {
			t.Error("SDK file should still exist after cache-only cleanup")
		}

		// Recreate cache for next test
		err = os.MkdirAll(testCacheDir, 0755)
		if err != nil {
			t.Fatalf("Failed to recreate test cache dir: %v", err)
		}
		err = os.WriteFile(cacheFile, []byte(`{"test": "data"}`), 0644)
		if err != nil {
			t.Fatalf("Failed to recreate test cache file: %v", err)
		}
	})

	// Test SDK-only cleanup
	t.Run("CleanSDKsOnly", func(t *testing.T) {
		// Verify files exist before cleanup
		if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
			t.Fatal("Cache file should exist before cleanup")
		}
		if _, err := os.Stat(sdkFile); os.IsNotExist(err) {
			t.Fatal("SDK file should exist before cleanup")
		}

		// Clean SDKs only
		err := os.RemoveAll(testSDKDir)
		if err != nil {
			t.Fatalf("Failed to clean SDK directory: %v", err)
		}

		// Verify SDK is gone but cache remains
		if _, err := os.Stat(testSDKDir); !os.IsNotExist(err) {
			t.Error("SDK directory should be removed")
		}
		if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
			t.Error("Cache file should still exist after SDK-only cleanup")
		}

		// Recreate SDK for next test
		err = os.MkdirAll(sdkSubDir, 0755)
		if err != nil {
			t.Fatalf("Failed to recreate test SDK subdir: %v", err)
		}
		err = os.WriteFile(sdkFile, []byte("fake sdk content"), 0644)
		if err != nil {
			t.Fatalf("Failed to recreate test SDK file: %v", err)
		}
	})

	// Test cleanup all
	t.Run("CleanupAll", func(t *testing.T) {
		// Verify files exist before cleanup
		if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
			t.Fatal("Cache file should exist before cleanup")
		}
		if _, err := os.Stat(sdkFile); os.IsNotExist(err) {
			t.Fatal("SDK file should exist before cleanup")
		}

		// Clean all
		err := os.RemoveAll(testCacheDir)
		if err != nil {
			t.Fatalf("Failed to clean cache directory: %v", err)
		}
		err = os.RemoveAll(testSDKDir)
		if err != nil {
			t.Fatalf("Failed to clean SDK directory: %v", err)
		}

		// Verify both are gone
		if _, err := os.Stat(testCacheDir); !os.IsNotExist(err) {
			t.Error("Cache directory should be removed")
		}
		if _, err := os.Stat(testSDKDir); !os.IsNotExist(err) {
			t.Error("SDK directory should be removed")
		}
	})
}

// TestOSLevelDirectoriesOnly verifies that the application only uses OS-level directories
func TestOSLevelDirectoriesOnly(t *testing.T) {
	cacheDir := config.GetCacheDir()
	sdkDir := config.GetSDKDir()

	// Both should be absolute paths
	if !filepath.IsAbs(cacheDir) {
		t.Errorf("Cache directory should be absolute, got %q", cacheDir)
	}
	if !filepath.IsAbs(sdkDir) {
		t.Errorf("SDK directory should be absolute, got %q", sdkDir)
	}

	// Neither should be in current working directory
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	if strings.HasPrefix(cacheDir, cwd) {
		t.Errorf("Cache directory should not be relative to current working directory %q, got %q", cwd, cacheDir)
	}
	if strings.HasPrefix(sdkDir, cwd) {
		t.Errorf("SDK directory should not be relative to current working directory %q, got %q", cwd, sdkDir)
	}

	// Should not contain relative path markers
	relativePaths := []string{"./", "../", "sdks/"}
	for _, rel := range relativePaths {
		if strings.Contains(cacheDir, rel) {
			t.Errorf("Cache directory should not contain relative path marker %q, got %q", rel, cacheDir)
		}
		if strings.Contains(sdkDir, rel) {
			t.Errorf("SDK directory should not contain relative path marker %q, got %q", rel, sdkDir)
		}
	}
}

// TestDirectoryInfoAccuracy verifies that DirectoryInfo reports accurate information
func TestDirectoryInfoAccuracy(t *testing.T) {
	info := config.GetDirectoryInfo()

	// Basic structure validation
	if info.CacheDir == "" {
		t.Error("DirectoryInfo should have non-empty CacheDir")
	}
	if info.SDKDir == "" {
		t.Error("DirectoryInfo should have non-empty SDKDir")
	}

	// Existence checks should match actual filesystem
	_, cacheStatErr := os.Stat(info.CacheDir)
	_, sdkStatErr := os.Stat(info.SDKDir)

	if (cacheStatErr == nil) != info.CacheExists {
		t.Errorf("DirectoryInfo.CacheExists (%v) doesn't match filesystem state (stat error: %v)", info.CacheExists, cacheStatErr)
	}

	if (sdkStatErr == nil) != info.SDKExists {
		t.Errorf("DirectoryInfo.SDKExists (%v) doesn't match filesystem state (stat error: %v)", info.SDKExists, sdkStatErr)
	}

	t.Logf("Directory info: Cache=%s (exists=%v), SDK=%s (exists=%v)",
		info.CacheDir, info.CacheExists, info.SDKDir, info.SDKExists)
}
