// Package updater provides self-update from GitHub releases.
// This is the reusable core extracted from pkg/self/install.go.
// Any binary (utm-dev, webviewer shell, hybrid-dashboard) can use this
// to update itself from GitHub release assets.
//
// For standalone examples (separate go.mod), the same logic is inlined
// in their main.go. This package is the canonical source of truth.
package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Config tells the updater where to find releases.
type Config struct {
	Repo  string // GitHub owner/repo (e.g. "joeblew999/utm-dev")
	Asset string // Asset name prefix (e.g. "webviewer-shell")
}

// Result holds the outcome of an update check or update.
type Result struct {
	CurrentVersion string
	LatestVersion  string
	UpdateAvailable bool
	Downloaded     bool
	Installed      bool
	AssetName      string
}

// Check queries GitHub for the latest release and returns whether an update is available.
func Check(cfg Config) (*Result, error) {
	release, err := fetchLatestRelease(cfg.Repo)
	if err != nil {
		return nil, err
	}

	assetName := findAsset(release, cfg.Asset)
	return &Result{
		LatestVersion:   release.TagName,
		UpdateAvailable: assetName != "",
		AssetName:       assetName,
	}, nil
}

// CanSelfUpdate returns true if the current platform supports self-update.
// Mobile platforms (iOS, Android) use app stores instead.
func CanSelfUpdate() bool {
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		return true
	default:
		// android, ios, js/wasm — can't self-update
		return false
	}
}

// Update downloads the latest release asset and extracts it to the executable's directory.
func Update(cfg Config) (*Result, error) {
	if !CanSelfUpdate() {
		return nil, fmt.Errorf("self-update not supported on %s (use app store)", runtime.GOOS)
	}
	if cfg.Repo == "" || cfg.Asset == "" {
		return nil, fmt.Errorf("update not configured (need repo and asset)")
	}

	release, err := fetchLatestRelease(cfg.Repo)
	if err != nil {
		return nil, err
	}

	result := &Result{
		LatestVersion: release.TagName,
	}

	// Find matching asset for current platform
	assetName := findAsset(release, cfg.Asset)
	if assetName == "" {
		return nil, fmt.Errorf("no matching asset for %s-%s in release %s",
			cfg.Asset, platformName(), release.TagName)
	}

	var downloadURL string
	for _, a := range release.Assets {
		if a.Name == assetName {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}

	result.AssetName = assetName
	fmt.Printf("Downloading %s (%s)...\n", assetName, release.TagName)

	// Download to temp file
	tmpFile, err := os.CreateTemp("", "app-update-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	dlResp, err := http.Get(downloadURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != 200 {
		return nil, fmt.Errorf("download failed: %s", dlResp.Status)
	}

	if _, err := io.Copy(tmpFile, dlResp.Body); err != nil {
		return nil, fmt.Errorf("failed to save download: %w", err)
	}
	tmpFile.Close()
	result.Downloaded = true

	// Extract to executable's directory
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get executable path: %w", err)
	}
	exePath, _ = filepath.EvalSymlinks(exePath)
	exeDir := filepath.Dir(exePath)

	if err := extractArchive(tmpFile.Name(), exeDir); err != nil {
		return nil, fmt.Errorf("failed to extract update: %w", err)
	}

	result.Installed = true
	fmt.Printf("Updated to %s\n", release.TagName)
	return result, nil
}

type githubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func fetchLatestRelease(repo string) (*githubRelease, error) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to fetch release info: %s", resp.Status)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}
	return &release, nil
}

// findAsset looks for an asset matching the prefix + current platform name.
func findAsset(release *githubRelease, assetPrefix string) string {
	wantPrefix := fmt.Sprintf("%s-%s", assetPrefix, platformName())
	for _, a := range release.Assets {
		if len(a.Name) >= len(wantPrefix) && a.Name[:len(wantPrefix)] == wantPrefix {
			return a.Name
		}
	}
	return ""
}

// platformName returns the platform name used in release asset names.
func platformName() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	default:
		return runtime.GOOS
	}
}

// extractArchive extracts a zip file to the destination directory.
func extractArchive(archivePath, destDir string) error {
	switch runtime.GOOS {
	case "darwin", "linux":
		cmd := exec.Command("unzip", "-o", archivePath, "-d", destDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	case "windows":
		cmd := exec.Command("powershell", "-Command",
			fmt.Sprintf("Expand-Archive -Force -Path '%s' -DestinationPath '%s'", archivePath, destDir))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
