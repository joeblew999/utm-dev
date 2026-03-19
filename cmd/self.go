package cmd

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/joeblew999/utm-dev/pkg/self"
	"github.com/spf13/cobra"
)

var selfCmd = &cobra.Command{
	Use:   "self",
	Short: "Manage utm-dev itself",
	Long: `Commands for managing utm-dev itself.

For Users:
  version        - Show version and check for updates
  upgrade        - Download and install latest release
  doctor         - Validate dependencies

For Developers:
  build          - Build utm-dev binaries for all platforms
  release        - Create git tag and trigger GitHub Actions
  release-check  - Check if GitHub release is ready (async monitoring)`,
}

var (
	buildLocal     bool // Flag for local mode
	buildObfuscate bool // Flag for garble obfuscation
)

var selfBuildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build utm-dev binaries for all platforms",
	Long: `Cross-compile utm-dev for all supported architectures and generate bootstrap scripts.

Output: All artifacts are placed in .dist/ directory
- Binaries: .dist/utm-dev-<platform>
- Scripts: .dist/macos-bootstrap.sh, .dist/windows-bootstrap.ps1

Flags:
  --local      Generate scripts that use local binaries (for testing)
  --obfuscate  Use garble to obfuscate code (auto-installs if needed)

This is a LOCAL build command - it does NOT create releases or push to GitHub.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := self.BuildOptions{
			UseLocal:  buildLocal,
			Obfuscate: buildObfuscate,
		}
		return self.Build(opts)
	},
}

var selfVersionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show utm-dev version",
	Long:  "Display the currently installed version of utm-dev.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return self.ShowVersion()
	},
}

var selfDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Validate installation and dependencies",
	Long:  "Check that utm-dev and all dependencies (Homebrew, git, go, task) are properly installed.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return self.Doctor()
	},
}

var selfReleaseCheckCmd = &cobra.Command{
	Use:   "release-check [tag]",
	Short: "Check if a GitHub release is ready",
	Long: `Check if a GitHub release exists and has assets.

This is useful after running 'self release' to monitor the async GitHub Actions workflow.

Examples:
  utm-dev self release-check v1.5.0
  utm-dev self release-check          # checks latest tag

Returns JSON with:
- exists: whether release exists on GitHub
- published: whether it's published
- assets: list of available files
- release_url: direct link to release
- workflow_url: link to monitor GitHub Actions`,
	RunE: func(cmd *cobra.Command, args []string) error {
		tag := ""
		if len(args) > 0 {
			tag = args[0]
		} else {
			// Get latest tag
			out, err := exec.Command("git", "describe", "--tags", "--abbrev=0").Output()
			if err != nil {
				return fmt.Errorf("no tag specified and couldn't get latest tag: %w", err)
			}
			tag = strings.TrimSpace(string(out))
		}
		return self.CheckRelease(tag)
	},
}

var selfUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Download and install latest release from GitHub",
	Long: `Download and install the latest utm-dev release from GitHub.

This downloads the pre-built binary for your platform from the GitHub Releases page
and installs it to your system PATH (~/.local/bin/ or ~/bin/).

Use this command to update utm-dev after a new release has been published.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return self.DownloadAndInstallLatest(self.FullRepoName)
	},
}

var selfReleaseCmd = &cobra.Command{
	Use:   "release [patch|minor|major|v1.2.3]",
	Short: "Create git tag and trigger GitHub Actions release",
	Long: `Create a git tag and push it to trigger GitHub Actions release workflow.

This command does:
1. Validates working directory is clean
2. Creates a git tag (e.g., v1.5.0)
3. Pushes tag to GitHub

GitHub Actions workflow then:
- Runs tests
- Builds obfuscated binaries for all platforms  
- Creates a GitHub Release
- Uploads artifacts to the release

Version options (defaults to 'minor'):
  patch      - Increment patch version (1.0.0 → 1.0.1)
  minor      - Increment minor version (1.0.0 → 1.1.0)
  major      - Increment major version (1.0.0 → 2.0.0)
  v1.2.3     - Use specific version

This is a TRIGGER ONLY - no local builds or tests. GitHub Actions does all the work.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		version := "minor" // Default to minor release
		if len(args) == 1 {
			version = args[0]
		}
		return self.Release(version)
	},
}

func init() {
	// Group for help organization
	selfCmd.GroupID = "self"

	rootCmd.AddCommand(selfCmd)

	// User commands
	selfCmd.AddCommand(selfVersionCmd)
	selfCmd.AddCommand(selfUpgradeCmd)
	selfCmd.AddCommand(selfDoctorCmd)

	// Developer commands
	selfCmd.AddCommand(selfBuildCmd)
	selfCmd.AddCommand(selfReleaseCmd)
	selfCmd.AddCommand(selfReleaseCheckCmd)

	// Add flags
	selfBuildCmd.Flags().BoolVar(&buildLocal, "local", false, "Generate bootstrap scripts for local testing (uses local binaries instead of GitHub releases)")
	selfBuildCmd.Flags().BoolVar(&buildObfuscate, "obfuscate", false, "Use garble to obfuscate binaries (auto-installs garble if needed)")
}
