package utm

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joeblew999/utm-dev/pkg/config"
	"github.com/joeblew999/utm-dev/pkg/installer"
)

// Cache key prefixes
const (
	cacheKeyPrefixApp = "utm-app"
	cacheKeyPrefixISO = "utm-iso"
)

// GetUTMCache returns a cache instance for UTM entries
func GetUTMCache() (*installer.Cache, error) {
	return installer.NewCache(config.GetCachePath())
}

// MakeUTMAppCacheKey generates cache key for UTM app
func MakeUTMAppCacheKey(version string) string {
	return fmt.Sprintf("%s-%s", cacheKeyPrefixApp, version)
}

// MakeISOCacheKey generates cache key for ISO
func MakeISOCacheKey(vmKey string) string {
	return fmt.Sprintf("%s-%s", cacheKeyPrefixISO, vmKey)
}

// IsUTMAppCached checks if UTM app version is cached and installed
func IsUTMAppCached(version, checksum string) bool {
	cache, err := GetUTMCache()
	if err != nil {
		return false
	}

	paths := GetPaths()
	sdk := &installer.SDK{
		Name:        MakeUTMAppCacheKey(version),
		Version:     version,
		Checksum:    checksum,
		InstallPath: paths.App,
	}

	// Check cache entry
	if !cache.IsCached(sdk) {
		return false
	}

	// Also verify the app actually exists
	utmctlPath := filepath.Join(paths.App, "Contents/MacOS/utmctl")
	if _, err := os.Stat(utmctlPath); os.IsNotExist(err) {
		return false
	}

	return true
}

// IsISOCached checks if an ISO is cached and exists
func IsISOCached(vmKey string) bool {
	gallery, err := LoadGallery()
	if err != nil {
		return false
	}

	vm, ok := gallery.GetVM(vmKey)
	if !ok {
		return false
	}

	cache, err := GetUTMCache()
	if err != nil {
		return false
	}

	paths := GetPaths()
	isoPath := filepath.Join(paths.ISO, vm.ISO.Filename)

	sdk := &installer.SDK{
		Name:        MakeISOCacheKey(vmKey),
		Version:     vmKey,
		Checksum:    vm.ISO.Checksum,
		InstallPath: isoPath,
	}

	// Check cache entry
	if !cache.IsCached(sdk) {
		return false
	}

	// Also verify the ISO actually exists
	if _, err := os.Stat(isoPath); os.IsNotExist(err) {
		return false
	}

	return true
}

// AddUTMAppToCache adds UTM app to cache after installation
func AddUTMAppToCache(version, checksum string) error {
	cache, err := GetUTMCache()
	if err != nil {
		return err
	}

	paths := GetPaths()
	sdk := &installer.SDK{
		Name:        MakeUTMAppCacheKey(version),
		Version:     version,
		Checksum:    checksum,
		InstallPath: paths.App,
	}

	cache.Add(sdk)
	return cache.Save()
}

// AddISOToCache adds ISO to cache after download
func AddISOToCache(vmKey string) error {
	gallery, err := LoadGallery()
	if err != nil {
		return fmt.Errorf("failed to load gallery: %w", err)
	}

	vm, ok := gallery.GetVM(vmKey)
	if !ok {
		return fmt.Errorf("VM '%s' not found in gallery", vmKey)
	}

	cache, err := GetUTMCache()
	if err != nil {
		return err
	}

	paths := GetPaths()
	isoPath := filepath.Join(paths.ISO, vm.ISO.Filename)

	sdk := &installer.SDK{
		Name:        MakeISOCacheKey(vmKey),
		Version:     vmKey,
		Checksum:    vm.ISO.Checksum,
		InstallPath: isoPath,
	}

	cache.Add(sdk)
	return cache.Save()
}

// GetCachedISOPath returns the path to a cached ISO if it exists
func GetCachedISOPath(vmKey string) (string, bool) {
	if !IsISOCached(vmKey) {
		return "", false
	}

	gallery, err := LoadGallery()
	if err != nil {
		return "", false
	}

	vm, ok := gallery.GetVM(vmKey)
	if !ok {
		return "", false
	}

	paths := GetPaths()
	return filepath.Join(paths.ISO, vm.ISO.Filename), true
}
