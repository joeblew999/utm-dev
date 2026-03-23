package installer

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/joeblew999/utm-dev/pkg/cli"
	"github.com/joeblew999/utm-dev/pkg/config"
	"github.com/schollz/progressbar/v3"
)

// SDK represents a software development kit.
type SDK struct {
	Name        string
	Version     string
	URL         string
	Checksum    string
	InstallPath string
}

// Install downloads and installs an SDK.
func Install(sdk *SDK, cache *Cache) error {
	// Check if the SDK is already cached
	if cache.IsCached(sdk) {
		cli.Info("%s %s is already installed and up-to-date.", sdk.Name, sdk.Version)
		return nil
	}

	// First, resolve the installation path and check if the SDK already exists.
	dest, err := ResolveInstallPath(sdk.InstallPath)
	if err != nil {
		return err
	}

	// Check if already installed and complete
	if _, err := os.Stat(dest); err == nil {
		// Verify it's actually complete by checking for expected files
		if isSDKComplete(dest, sdk.Name) {
			cli.Success("%s %s is already installed at %s", sdk.Name, sdk.Version, dest)
			cache.Add(sdk)
			if err := cache.Save(); err != nil {
				return fmt.Errorf("failed to save cache: %w", err)
			}
			return nil
		}
		cli.Warn("%s found but appears incomplete, reinstalling...", sdk.Name)
		os.RemoveAll(dest)
	}

	// If the SDK doesn't exist and there's no URL, it's a manual installation.
	if sdk.URL == "" {
		return fmt.Errorf("cannot automatically install SDK %s. Please install it manually (e.g., by installing or updating Xcode) and ensure it is available at %s", sdk.Name, dest)
	}

	cli.Info("Downloading %s %s...", sdk.Name, sdk.Version)

	// Create a temporary file with the correct extension from the URL
	fileExt := filepath.Ext(sdk.URL)
	tmpFile, err := os.CreateTemp("", "sdk-download-*"+fileExt)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tmpFile.Name()) // Clean up the temp file

	// Get the data with retry and progress
	client := &http.Client{Timeout: 60 * time.Minute} // Extended timeout for large downloads like NDK
	req, err := http.NewRequest("GET", sdk.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Retry logic with exponential backoff
	maxRetries := 3
	var resp *http.Response
	for attempt := 1; attempt <= maxRetries; attempt++ {
		cli.Info("Download attempt %d/%d...", attempt, maxRetries)

		resp, err = client.Do(req)
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		if attempt < maxRetries {
			if resp != nil {
				resp.Body.Close()
			}
			backoff := time.Duration(attempt) * time.Second
			cli.Warn("Retrying in %v...", backoff)
			time.Sleep(backoff)
		} else {
			if resp != nil {
				resp.Body.Close()
			}
			return fmt.Errorf("failed to download SDK after %d attempts: %w", maxRetries, err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download SDK: received status code %d", resp.StatusCode)
	}

	// Get content length for progress bar
	contentLength := resp.ContentLength
	var bar *progressbar.ProgressBar
	if contentLength > 0 {
		bar = progressbar.NewOptions64(contentLength,
			progressbar.OptionSetDescription("Downloading"),
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(50),
			progressbar.OptionThrottle(65*time.Millisecond),
			progressbar.OptionShowCount(),
			progressbar.OptionOnCompletion(func() {
				fmt.Fprint(os.Stderr, "\n")
			}),
		)
	} else {
		bar = progressbar.NewOptions(-1,
			progressbar.OptionSetDescription("Downloading"),
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(50),
			progressbar.OptionThrottle(65*time.Millisecond),
		)
	}

	// Create a tee reader to write to the file and the hash simultaneously
	hasher := sha256.New()
	progressReader := progressbar.NewReader(resp.Body, bar)
	teeReader := io.TeeReader(&progressReader, hasher)

	// Write the body to the temporary file
	_, err = io.Copy(tmpFile, teeReader)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write to temporary file: %w", err)
	}
	tmpFile.Close()

	// Verify the checksum
	calculatedChecksum := hex.EncodeToString(hasher.Sum(nil))
	expectedChecksum := strings.TrimPrefix(sdk.Checksum, "sha256:")

	if expectedChecksum != "" && calculatedChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, calculatedChecksum)
	}
	cli.Success("Checksum verified.")

	cli.Info("Downloaded %s %s (%.1f MB)", sdk.Name, sdk.Version, float64(contentLength)/1024/1024)

	// The destination is already resolved, so we don't need to call ResolveInstallPath again.

	// Extract the archive
	if err := os.MkdirAll(dest, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	cli.Info("Extracting to %s...", dest)
	if err := Extract(tmpFile.Name(), dest); err != nil {
		return fmt.Errorf("failed to extract SDK: %w", err)
	}
	cli.Success("Extraction complete.")

	// Add to cache and save
	cache.Add(sdk)
	if err := cache.Save(); err != nil {
		return fmt.Errorf("failed to save cache: %w", err)
	}

	cli.Success("Successfully installed %s %s to %s", sdk.Name, sdk.Version, dest)

	// If OpenJDK was installed, print instructions for setting JAVA_HOME
	if strings.Contains(sdk.Name, "openjdk") {
		cli.Info("\n---------------------------------------------------------------------")
		cli.Info("IMPORTANT: To use this JDK for Android development with Gio,")
		cli.Info("you need to set the JAVA_HOME environment variable.")
		cli.Info("\nFor your current shell session, run:")
		cli.Info("export JAVA_HOME=\"%s\"", dest)
		cli.Info("\nTo make this change permanent, add the line above to your")
		cli.Info("shell profile file (e.g., ~/.zshrc, ~/.bash_profile).")
		cli.Info("---------------------------------------------------------------------")
	}

	return nil
}

// isSDKComplete checks if an SDK installation appears complete
func isSDKComplete(dest, sdkName string) bool {
	// Different checks for different SDK types
	switch {
	case strings.Contains(sdkName, "openjdk"):
		// Check for Java executable
		javaPath := filepath.Join(dest, "bin", "java")
		if runtime.GOOS == "windows" {
			javaPath += ".exe"
		}
		_, err := os.Stat(javaPath)
		return err == nil
	case strings.Contains(sdkName, "android"):
		// Check for platform directory
		_, err := os.Stat(filepath.Join(dest, "android.jar"))
		return err == nil
	case strings.Contains(sdkName, "build-tools"):
		// Check for aapt
		aaptPath := filepath.Join(dest, "aapt")
		if runtime.GOOS == "windows" {
			aaptPath += ".exe"
		}
		_, err := os.Stat(aaptPath)
		return err == nil
	case strings.Contains(sdkName, "platform-tools"):
		// Check for adb
		adbPath := filepath.Join(dest, "adb")
		if runtime.GOOS == "windows" {
			adbPath += ".exe"
		}
		_, err := os.Stat(adbPath)
		return err == nil
	case strings.Contains(sdkName, "ndk"):
		// Check for NDK build tools
		ndkBuildPath := filepath.Join(dest, "ndk-build")
		if runtime.GOOS == "windows" {
			ndkBuildPath += ".cmd"
		}
		_, err := os.Stat(ndkBuildPath)
		return err == nil
	default:
		// For other SDKs, just check directory exists
		return true
	}
}

// InstallAndroidSDK installs Android SDK components using sdkmanager with proper path handling
func InstallAndroidSDK(sdkName, sdkManagerName, sdkRoot string) error {
	cli.Info("Installing %s via Android SDK Manager...", sdkName)

	// Ensure paths are properly formatted for Java
	sdkRoot = filepath.Clean(sdkRoot)
	cmdlineToolsPath := filepath.Join(sdkRoot, "cmdline-tools", "11.0", "cmdline-tools", "bin")
	javaHome := filepath.Join(sdkRoot, "openjdk", "17", "jdk-17.0.11+9", "Contents", "Home")

	// Check if sdkmanager exists
	sdkManagerPath := filepath.Join(cmdlineToolsPath, "sdkmanager")
	if _, err := os.Stat(sdkManagerPath); os.IsNotExist(err) {
		return fmt.Errorf("sdkmanager not found at %s", sdkManagerPath)
	}

	// Set up environment with proper path handling
	env := os.Environ()
	env = append(env, "JAVA_HOME="+javaHome)
	env = append(env, "ANDROID_SDK_ROOT="+sdkRoot)
	env = append(env, "ANDROID_HOME="+sdkRoot)

	// Add tools to PATH
	pathEnv := "PATH=" + cmdlineToolsPath + string(os.PathListSeparator) + os.Getenv("PATH")
	env = append(env, pathEnv)

	// Create command with proper argument handling
	cmd := exec.Command(sdkManagerPath, sdkManagerName, "--sdk_root="+sdkRoot)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Add retry logic
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		cli.Info("Attempt %d/%d...", attempt, maxRetries)

		err := cmd.Run()
		if err == nil {
			cli.Success("Successfully installed %s", sdkName)
			return nil
		}

		if attempt < maxRetries {
			cli.Error("Attempt %d failed: %v", attempt, err)
			time.Sleep(time.Duration(attempt) * time.Second)
		} else {
			return fmt.Errorf("failed to install %s after %d attempts: %w", sdkName, maxRetries, err)
		}
	}

	return nil
}

func ResolveInstallPath(path string) (string, error) {
	if path == "" {
		// Default to OS-appropriate SDK directory
		return config.GetSDKDir(), nil
	}

	// Expand environment variables first
	expandedPath := os.ExpandEnv(path)

	// If it's already absolute, return as-is
	if filepath.IsAbs(expandedPath) {
		return expandedPath, nil
	}

	// For relative paths starting with "sdks/", use OS-specific SDK directory
	if strings.HasPrefix(expandedPath, "sdks/") {
		return filepath.Join(config.GetSDKDir(), strings.TrimPrefix(expandedPath, "sdks/")), nil
	}

	// For other relative paths, make them relative to current directory
	if !filepath.IsAbs(expandedPath) {
		if cwd, err := os.Getwd(); err == nil {
			return filepath.Join(cwd, expandedPath), nil
		}
	}

	return expandedPath, nil
}
