package self

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/joeblew999/utm-dev/pkg/self/output"
)

// Version is set by the build process
var Version = "dev"

// ShowVersion displays the current version of utm-dev
func ShowVersion() error {
	output.Run("self version", func() (*output.VersionResult, error) {
		location := ""
		if path, err := exec.LookPath("utm-dev"); err == nil {
			location = path
		}

		return &output.VersionResult{
			Version:  Version,
			OS:       runtime.GOOS,
			Arch:     runtime.GOARCH,
			Location: location,
		}, nil
	})
	return nil
}

// ShowStatus checks installation status and available updates
func ShowStatus() error {
	output.Run("self status", func() (*output.StatusResult, error) {
		result := &output.StatusResult{}

		// Check if installed
		installPath, err := exec.LookPath("utm-dev")
		if err != nil {
			result.Installed = false
			result.UpdateAvailable = false
			return result, nil
		}

		result.Installed = true
		result.CurrentVersion = normalizeVersion(Version)
		result.Location = installPath

		// Check for updates (from GitHub)
		latest, err := getLatestVersion(FullRepoName)
		if err == nil && latest != "" {
			result.LatestVersion = latest
			result.UpdateAvailable = (result.CurrentVersion != latest)
		} else {
			result.LatestVersion = ""
			result.UpdateAvailable = false
		}

		return result, nil
	})
	return nil
}

// getLatestVersion fetches the latest release tag from GitHub
func getLatestVersion(repo string) (string, error) {
	cmd := exec.Command("git", "ls-remote", "--tags", "--refs",
		fmt.Sprintf("https://github.com/%s.git", repo))

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse output to find latest semver tag
	lines := strings.Split(string(output), "\n")
	var latest string

	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		tag := filepath.Base(parts[1]) // Extract tag name from refs/tags/v1.2.3
		if strings.HasPrefix(tag, "v") && strings.Contains(tag, ".") {
			// Simple comparison - just use last tag (sorted by git)
			latest = tag
		}
	}

	return latest, nil
}
