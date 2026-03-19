package utm

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// InstallStatus represents the current UTM installation status
type InstallStatus struct {
	Installed       bool
	InstalledPath   string
	InstalledVersion string
	GalleryVersion  string
	UpdateAvailable bool
}

// GetInstallStatus checks the current UTM installation status
func GetInstallStatus() (*InstallStatus, error) {
	if runtime.GOOS != "darwin" {
		return nil, fmt.Errorf("UTM is only available on macOS")
	}

	gallery, err := LoadGallery()
	if err != nil {
		return nil, err
	}

	paths := GetPaths()
	status := &InstallStatus{
		GalleryVersion: gallery.Meta.UTMApp.Version,
	}

	// Check if UTM is installed
	utmctlPath := filepath.Join(paths.App, "Contents/MacOS/utmctl")
	if _, err := os.Stat(utmctlPath); err == nil {
		status.Installed = true
		status.InstalledPath = paths.App

		// Try to get installed version
		version, err := getInstalledVersion(utmctlPath)
		if err == nil {
			status.InstalledVersion = version
			status.UpdateAvailable = version != gallery.Meta.UTMApp.Version
		}
	}

	return status, nil
}

// getInstalledVersion tries to get the version of installed UTM
func getInstalledVersion(utmctlPath string) (string, error) {
	cmd := exec.Command(utmctlPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	// Parse version from output - format varies
	return string(output), nil
}

// InstallUTM installs the UTM application
func InstallUTM(force bool) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("UTM is only available on macOS")
	}

	gallery, err := LoadGallery()
	if err != nil {
		return fmt.Errorf("failed to load gallery: %w", err)
	}

	utmApp := gallery.Meta.UTMApp
	paths := GetPaths()
	appPath := paths.App

	// Check cache first (idempotent)
	if !force && IsUTMAppCached(utmApp.Version, utmApp.Checksum) {
		fmt.Printf("UTM v%s is already installed and cached at %s\n", utmApp.Version, appPath)
		return nil
	}

	// Check if already installed (but not in cache)
	utmctlPath := filepath.Join(appPath, "Contents/MacOS/utmctl")
	if !force {
		if _, err := os.Stat(utmctlPath); err == nil {
			// Compare installed version to gallery version
			installedVersion, verErr := getInstalledVersion(utmctlPath)
			if verErr == nil && strings.TrimSpace(installedVersion) == utmApp.Version {
				fmt.Printf("UTM v%s is already installed at %s\n", utmApp.Version, appPath)
				if err := AddUTMAppToCache(utmApp.Version, utmApp.Checksum); err != nil {
					fmt.Printf("Warning: failed to update cache: %v\n", err)
				}
				return nil
			}
			// Version mismatch — fall through to reinstall
			fmt.Printf("UTM update available: installed=%s gallery=%s\n",
				strings.TrimSpace(installedVersion), utmApp.Version)
		}
	}

	fmt.Printf("Installing UTM v%s...\n", utmApp.Version)

	// Ensure global directories exist
	if err := EnsureGlobalDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Download DMG
	dmgPath := filepath.Join(filepath.Dir(appPath), "UTM.dmg")
	fmt.Printf("Downloading from %s...\n", utmApp.URL)
	if err := downloadFile(utmApp.URL, dmgPath); err != nil {
		return fmt.Errorf("failed to download UTM: %w", err)
	}
	defer os.Remove(dmgPath) // Clean up DMG after installation

	// Mount DMG
	mountPoint := "/tmp/utm-mount"
	fmt.Println("Mounting DMG...")
	if err := mountDMG(dmgPath, mountPoint); err != nil {
		return fmt.Errorf("failed to mount DMG: %w", err)
	}
	defer unmountDMG(mountPoint)

	// Remove existing installation if force
	if force {
		if err := os.RemoveAll(appPath); err != nil {
			fmt.Printf("Warning: failed to remove existing installation: %v\n", err)
		}
	}

	// Copy UTM.app
	fmt.Printf("Copying UTM.app to %s...\n", appPath)
	srcApp := filepath.Join(mountPoint, "UTM.app")
	if err := copyDir(srcApp, appPath); err != nil {
		return fmt.Errorf("failed to copy UTM.app: %w", err)
	}

	// Add to cache for idempotency
	if err := AddUTMAppToCache(utmApp.Version, utmApp.Checksum); err != nil {
		fmt.Printf("Warning: failed to update cache: %v\n", err)
	}

	fmt.Printf("✓ UTM v%s installed successfully\n", utmApp.Version)
	return nil
}

// UninstallUTM removes the UTM application
func UninstallUTM() error {
	paths := GetPaths()
	appPath := paths.App

	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		fmt.Println("UTM is not installed")
		return nil
	}

	fmt.Printf("Removing %s...\n", appPath)
	if err := os.RemoveAll(appPath); err != nil {
		return fmt.Errorf("failed to remove UTM: %w", err)
	}

	fmt.Println("✓ UTM uninstalled successfully")
	return nil
}

// downloadFile downloads a file from URL to local path
// downloadFile downloads url to destPath with resume support and automatic retries.
// Uses HTTP Range requests to resume partial downloads (handles CDN disconnects).
func downloadFile(url, destPath string) error {
	const maxRetries = 15
	const retryDelay = 3

	client := &http.Client{}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Check existing partial file size for resume
		var offset int64
		if fi, err := os.Stat(destPath); err == nil {
			offset = fi.Size()
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return err
		}
		if offset > 0 {
			req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
		}

		resp, err := client.Do(req)
		if err != nil {
			if attempt < maxRetries {
				fmt.Printf("  retry %d/%d after error: %v\n", attempt, maxRetries, err)
				continue
			}
			return err
		}

		// 416 = Range Not Satisfiable → file already complete
		if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable {
			resp.Body.Close()
			return nil
		}
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
			resp.Body.Close()
			return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
		}

		// Open file for append (or create)
		flag := os.O_CREATE | os.O_WRONLY
		if resp.StatusCode == http.StatusPartialContent {
			flag |= os.O_APPEND
		} else {
			flag |= os.O_TRUNC // server ignored Range, restart
		}
		out, err := os.OpenFile(destPath, flag, 0644)
		if err != nil {
			resp.Body.Close()
			return err
		}

		_, copyErr := io.Copy(out, resp.Body)
		resp.Body.Close()
		out.Close()

		if copyErr != nil {
			if attempt < maxRetries {
				fmt.Printf("  retry %d/%d after disconnect\n", attempt, maxRetries)
				continue
			}
			return fmt.Errorf("download failed after %d attempts: %w", maxRetries, copyErr)
		}

		return nil
	}

	return fmt.Errorf("download failed after %d attempts", maxRetries)
}

// mountDMG mounts a DMG file
func mountDMG(dmgPath, mountPoint string) error {
	cmd := exec.Command("hdiutil", "attach", dmgPath, "-mountpoint", mountPoint, "-quiet")
	return cmd.Run()
}

// unmountDMG unmounts a DMG
func unmountDMG(mountPoint string) error {
	cmd := exec.Command("hdiutil", "detach", mountPoint, "-quiet")
	return cmd.Run()
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	cmd := exec.Command("cp", "-R", src, dst)
	return cmd.Run()
}

// DownloadISO downloads an ISO for a VM from the gallery
func DownloadISO(vmKey string, force bool) error {
	gallery, err := LoadGallery()
	if err != nil {
		return fmt.Errorf("failed to load gallery: %w", err)
	}

	vm, ok := gallery.GetVM(vmKey)
	if !ok {
		return fmt.Errorf("VM '%s' not found in gallery", vmKey)
	}

	if vm.ISO.URL == "" {
		return fmt.Errorf("VM '%s' does not have an ISO URL", vmKey)
	}

	paths := GetPaths()
	isoPath := filepath.Join(paths.ISO, vm.ISO.Filename)

	// Check cache first (idempotent)
	if !force && IsISOCached(vmKey) {
		fmt.Printf("ISO already cached at %s\n", isoPath)
		return nil
	}

	// Check if already downloaded (but not in cache - verify size then add to cache)
	if !force {
		if fi, err := os.Stat(isoPath); err == nil {
			// If expected size is known, validate completeness
			if vm.ISO.Size > 0 && fi.Size() != vm.ISO.Size {
				fmt.Printf("ISO exists but is incomplete (%d/%d bytes) — redownloading\n", fi.Size(), vm.ISO.Size)
			} else {
				fmt.Printf("ISO already exists at %s\n", isoPath)
				if err := AddISOToCache(vmKey); err != nil {
					fmt.Printf("Warning: failed to update cache: %v\n", err)
				}
				return nil
			}
		}
	}

	// Ensure global ISO directory exists
	if err := EnsureGlobalDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	sizeGB := float64(vm.ISO.Size) / 1024 / 1024 / 1024
	fmt.Printf("Downloading %s (%.1f GB)...\n", vm.ISO.Filename, sizeGB)
	fmt.Printf("URL: %s\n", vm.ISO.URL)

	if err := downloadFile(vm.ISO.URL, isoPath); err != nil {
		return fmt.Errorf("failed to download ISO: %w", err)
	}

	// Add to cache for idempotency
	if err := AddISOToCache(vmKey); err != nil {
		fmt.Printf("Warning: failed to update cache: %v\n", err)
	}

	fmt.Printf("✓ ISO downloaded to %s\n", isoPath)
	return nil
}
