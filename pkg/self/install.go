package self

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/joeblew999/utm-dev/pkg/cli"
	"github.com/joeblew999/utm-dev/pkg/self/output"
)


// InstallSelf installs the current binary to system path.
// For Unix (macOS, Linux): /usr/local/bin/utm-dev
// For Windows: %USERPROFILE%\utm-dev.exe (and adds to PATH if needed)
func InstallSelf() error {
	var installPath string
	var err error

	switch runtime.GOOS {
	case "darwin", "linux":
		installPath, err = installSelfUnix()
	case "windows":
		installPath, err = installSelfWindows()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	if err != nil {
		return err
	}

	// Check if in PATH
	inPath := false
	if foundPath, pathErr := exec.LookPath(BinaryName); pathErr == nil {
		inPath = (foundPath == installPath)
	}

	// Check dependencies
	depsOK := true
	if err := checkCommand("git", "--version"); err != nil {
		depsOK = false
	}
	if err := checkCommand("go", "version"); err != nil {
		depsOK = false
	}
	if err := checkCommand("task", "--version"); err != nil {
		depsOK = false
	}

	result := output.SetupResult{
		Installed:      true,
		Location:       installPath,
		InPath:         inPath,
		DependenciesOK: depsOK,
	}

	output.OK("self setup", result)
	return nil
}

// installSelfUnix installs the binary on Unix systems (macOS, Linux)
func installSelfUnix() (string, error) {
	installPath := UnixInstallPath

	// Get current executable path
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	// Resolve symlinks
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve executable path: %w", err)
	}

	// Check if we can write directly (unlikely)
	if isWritable(UnixInstallDir) {
		if err := copyFile(exePath, installPath); err != nil {
			return "", fmt.Errorf("failed to copy binary: %w", err)
		}
	} else {
		// Need sudo
		cmd := exec.Command("sudo", "cp", exePath, installPath)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("sudo cp failed: %w", err)
		}

		// Ensure executable
		cmd = exec.Command("sudo", "chmod", "+x", installPath)
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("chmod failed: %w", err)
		}
	}

	// macOS: Remove quarantine attribute
	if runtime.GOOS == "darwin" {
		cmd := exec.Command("sudo", "xattr", "-d", "com.apple.quarantine", installPath)
		// Ignore error - attribute may not exist
		_ = cmd.Run()
	}

	// Verify installation
	if _, err := os.Stat(installPath); err != nil {
		return "", fmt.Errorf("installation verification failed: %w", err)
	}

	return installPath, nil
}

// installSelfWindows installs the binary on Windows
func installSelfWindows() (string, error) {
	// Install to user profile directory
	userProfile := os.Getenv("USERPROFILE")
	if userProfile == "" {
		return "", fmt.Errorf("USERPROFILE environment variable not set")
	}

	installPath := filepath.Join(userProfile, "utm-dev.exe")

	// Get current executable path
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	// Copy file
	if err := copyFile(exePath, installPath); err != nil {
		return "", fmt.Errorf("failed to copy binary: %w", err)
	}

	// Unblock file (Windows SmartScreen)
	cmd := exec.Command("powershell", "-Command", "Unblock-File", "-Path", installPath)
	// Ignore error - may not be needed
	_ = cmd.Run()

	// Verify installation
	if _, err := os.Stat(installPath); err != nil {
		return "", fmt.Errorf("installation verification failed: %w", err)
	}

	return installPath, nil
}

// copyFile copies a file from src to dst, preserving permissions
func copyFile(src, dst string) error {
	// Open source file
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer srcFile.Close()

	// Get source file info
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat source: %w", err)
	}

	// Create destination file
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return fmt.Errorf("failed to create destination: %w", err)
	}
	defer dstFile.Close()

	// Copy contents
	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("failed to copy contents: %w", err)
	}

	return nil
}

// isWritable checks if a directory is writable by the current user
func isWritable(path string) bool {
	testFile := filepath.Join(path, ".write-test")
	file, err := os.Create(testFile)
	if err != nil {
		return false
	}
	file.Close()
	os.Remove(testFile)
	return true
}

// DownloadAndInstallLatest downloads the latest release and installs it.
// This is used by the 'self upgrade' command.
func DownloadAndInstallLatest(repo string) error {
	result := output.UpgradeResult{
		PreviousVersion: Version,
		Downloaded:      false,
		Installed:       false,
	}

	// Get latest release info
	releaseURL := GetLatestReleaseURL()
	resp, err := http.Get(releaseURL)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to fetch release info: %s", resp.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to parse release info: %w", err)
	}

	result.NewVersion = release.TagName

	// Get binary name for current platform
	binaryName := getBinaryName()
	if binaryName == "" {
		return fmt.Errorf("unsupported platform: %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Find matching asset
	var downloadURL string
	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("binary not found for %s/%s in release %s", runtime.GOOS, runtime.GOARCH, release.TagName)
	}

	// Download to temp file
	tmpFile, err := os.CreateTemp("", TempFilePattern)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	resp, err = http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed: %s", resp.Status)
	}

	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		return fmt.Errorf("failed to save binary: %w", err)
	}

	result.Downloaded = true

	// Make executable
	if err := tmpFile.Chmod(0755); err != nil {
		return fmt.Errorf("failed to make binary executable: %w", err)
	}

	tmpFile.Close()

	// Replace current binary with downloaded one
	installPath := getInstallPath()
	result.Location = installPath

	// On Unix, we might need sudo
	if runtime.GOOS != "windows" {
		if !isWritable(filepath.Dir(installPath)) {
			cmd := exec.Command("sudo", "cp", tmpFile.Name(), installPath)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("sudo cp failed: %w", err)
			}

			cmd = exec.Command("sudo", "chmod", "+x", installPath)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("chmod failed: %w", err)
			}
		} else {
			if err := copyFile(tmpFile.Name(), installPath); err != nil {
				return fmt.Errorf("failed to install: %w", err)
			}
		}

		// macOS: Remove quarantine
		if runtime.GOOS == "darwin" {
			exec.Command("sudo", "xattr", "-d", "com.apple.quarantine", installPath).Run()
		}
	} else {
		// Windows
		if err := copyFile(tmpFile.Name(), installPath); err != nil {
			return fmt.Errorf("failed to install: %w", err)
		}

		// Unblock file
		exec.Command("powershell", "-Command", "Unblock-File", "-Path", installPath).Run()
	}

	result.Installed = true
	output.OK("self upgrade", result)
	return nil
}

// getBinaryName returns the binary name for the current platform using CurrentArchitecture
func getBinaryName() string {
	arch := CurrentArchitecture()
	if arch == nil {
		// Fallback if platform not supported
		return fmt.Sprintf("utm-dev-%s-%s", runtime.GOOS, runtime.GOARCH)
	}
	return arch.BinaryName()
}

// getInstallPath returns the installation path for the current platform
func getInstallPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("USERPROFILE"), "utm-dev.exe")
	}
	return UnixInstallPath
}

// UninstallSelf removes utm-dev from the system path.
// For Unix (macOS, Linux): removes /usr/local/bin/utm-dev
// For Windows: removes %USERPROFILE%\utm-dev.exe
func UninstallSelf() error {
	result := output.UninstallResult{
		Removed: []string{},
		Failed:  []string{},
	}

	// Find ALL installations
	installations := findAllGoupUtilInstallations()

	if len(installations) == 0 {
		output.OK("self uninstall", result)
		return nil
	}

	// Remove all installations
	for _, installPath := range installations {
		if err := removeBinary(installPath); err != nil {
			result.Failed = append(result.Failed, installPath)
		} else {
			result.Removed = append(result.Removed, installPath)
		}
	}

	output.OK("self uninstall", result)
	return nil
}

// findAllGoupUtilInstallations finds all utm-dev binaries in PATH
func findAllGoupUtilInstallations() []string {
	var installations []string

	// Get PATH
	pathEnv := os.Getenv("PATH")
	paths := filepath.SplitList(pathEnv)

	// Check each directory in PATH
	for _, dir := range paths {
		binaryPath := filepath.Join(dir, BinaryName)

		// Check if file exists and is executable
		if info, err := os.Stat(binaryPath); err == nil && !info.IsDir() {
			// Check if executable
			if info.Mode()&0111 != 0 {
				installations = append(installations, binaryPath)
			}
		}
	}

	return installations
}

// removeBinary removes a binary file, using sudo if needed
func removeBinary(path string) error {
	// Try to remove directly
	if err := os.Remove(path); err != nil {
		// On Unix, might need sudo
		if runtime.GOOS != "windows" {
			cli.Info("Need sudo privileges to remove %s", path)
			cmd := exec.Command("sudo", "rm", path)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to remove: %w", err)
			}
		} else {
			return fmt.Errorf("failed to remove: %w", err)
		}
	}

	// Verify removal
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("removal verification failed - file still exists")
	}

	return nil
}
