package self

import (
	"os"
	"path/filepath"
	"runtime"
)

// Repository configuration
const (
	GitHubOwner  = "joeblew999"
	GitHubRepo   = "utm-dev"
	FullRepoName = GitHubOwner + "/" + GitHubRepo
)

// GitHub URLs
const (
	GitHubAPIBase = "https://api.github.com"
	GitHubBase    = "https://github.com"
)

// Binary configuration
const (
	BinaryName = "utm-dev"
)

// Installation paths (Unix)
const (
	UnixInstallDir  = "/usr/local/bin"
	UnixInstallPath = UnixInstallDir + "/" + BinaryName
)

// Directory and file names
const (
	ScriptsDir             = "scripts"
	DistDir                = ".dist"
	MacOSBootstrapScript   = "macos-bootstrap.sh"
	WindowsBootstrapScript = "windows-bootstrap.ps1"
)

// Temp file pattern
const TempFilePattern = "utm-dev-*"

// GetInstallPath returns the installation path for the current platform
func GetInstallPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("USERPROFILE"), BinaryName+".exe")
	}
	return UnixInstallPath
}

// GetLatestReleaseURL returns the GitHub API URL for latest release
func GetLatestReleaseURL() string {
	return GitHubAPIBase + "/repos/" + FullRepoName + "/releases/latest"
}

// GetRepoGitURL returns the git clone URL
func GetRepoGitURL() string {
	return GitHubBase + "/" + FullRepoName + ".git"
}

// GetWindowsInstallPath returns the Windows installation path
func GetWindowsInstallPath() string {
	return filepath.Join(os.Getenv("USERPROFILE"), BinaryName+".exe")
}

// GetUnixInstallPath returns the Unix installation path
func GetUnixInstallPath() string {
	return UnixInstallPath
}
