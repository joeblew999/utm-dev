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

	"github.com/joeblew999/utm-dev/pkg/cli"
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
		cli.Info("UTM v%s is already installed and cached at %s", utmApp.Version, appPath)
		return nil
	}

	// Check if already installed (but not in cache)
	utmctlPath := filepath.Join(appPath, "Contents/MacOS/utmctl")
	if !force {
		if _, err := os.Stat(utmctlPath); err == nil {
			// Compare installed version to gallery version
			installedVersion, verErr := getInstalledVersion(utmctlPath)
			if verErr == nil && strings.TrimSpace(installedVersion) == utmApp.Version {
				cli.Info("UTM v%s is already installed at %s", utmApp.Version, appPath)
				if err := AddUTMAppToCache(utmApp.Version, utmApp.Checksum); err != nil {
					cli.Warn("failed to update cache: %v", err)
				}
				return nil
			}
			// Version mismatch — fall through to reinstall
			cli.Info("UTM update available: installed=%s gallery=%s",
				strings.TrimSpace(installedVersion), utmApp.Version)
		}
	}

	cli.Info("Installing UTM v%s...", utmApp.Version)

	// Ensure global directories exist
	if err := EnsureGlobalDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Download DMG
	dmgPath := filepath.Join(filepath.Dir(appPath), "UTM.dmg")
	cli.Info("Downloading from %s...", utmApp.URL)
	if err := downloadFile(utmApp.URL, dmgPath); err != nil {
		return fmt.Errorf("failed to download UTM: %w", err)
	}
	defer os.Remove(dmgPath) // Clean up DMG after installation

	// Mount DMG
	mountPoint := "/tmp/utm-mount"
	cli.Info("Mounting DMG...")
	if err := mountDMG(dmgPath, mountPoint); err != nil {
		return fmt.Errorf("failed to mount DMG: %w", err)
	}
	defer unmountDMG(mountPoint)

	// Remove existing installation if force
	if force {
		if err := os.RemoveAll(appPath); err != nil {
			cli.Warn("failed to remove existing installation: %v", err)
		}
	}

	// Copy UTM.app
	cli.Info("Copying UTM.app to %s...", appPath)
	srcApp := filepath.Join(mountPoint, "UTM.app")
	if err := copyDir(srcApp, appPath); err != nil {
		return fmt.Errorf("failed to copy UTM.app: %w", err)
	}

	// Add to cache for idempotency
	if err := AddUTMAppToCache(utmApp.Version, utmApp.Checksum); err != nil {
		cli.Warn("failed to update cache: %v", err)
	}

	cli.Success("UTM v%s installed successfully", utmApp.Version)
	return nil
}

// UninstallUTM removes the UTM application
func UninstallUTM() error {
	paths := GetPaths()
	appPath := paths.App

	if _, err := os.Stat(appPath); os.IsNotExist(err) {
		cli.Info("UTM is not installed")
		return nil
	}

	cli.Info("Removing %s...", appPath)
	if err := os.RemoveAll(appPath); err != nil {
		return fmt.Errorf("failed to remove UTM: %w", err)
	}

	cli.Success("UTM uninstalled successfully")
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
				cli.Warn("retry %d/%d after error: %v", attempt, maxRetries, err)
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
				cli.Warn("retry %d/%d after disconnect", attempt, maxRetries)
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
		cli.Info("ISO already cached at %s", isoPath)
		return nil
	}

	// Check if already downloaded (but not in cache - verify size then add to cache)
	if !force {
		if fi, err := os.Stat(isoPath); err == nil {
			// If expected size is known, validate completeness
			if vm.ISO.Size > 0 && fi.Size() != vm.ISO.Size {
				cli.Warn("ISO exists but is incomplete (%d/%d bytes) — redownloading", fi.Size(), vm.ISO.Size)
			} else {
				cli.Info("ISO already exists at %s", isoPath)
				if err := AddISOToCache(vmKey); err != nil {
					cli.Warn("failed to update cache: %v", err)
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
	cli.Info("Downloading %s (%.1f GB)...", vm.ISO.Filename, sizeGB)
	cli.Info("URL: %s", vm.ISO.URL)

	if err := downloadFile(vm.ISO.URL, isoPath); err != nil {
		return fmt.Errorf("failed to download ISO: %w", err)
	}

	// Add to cache for idempotency
	if err := AddISOToCache(vmKey); err != nil {
		cli.Warn("failed to update cache: %v", err)
	}

	cli.Success("ISO downloaded to %s", isoPath)
	return nil
}
