package self

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/joeblew999/utm-dev/pkg/cli"
	"github.com/joeblew999/utm-dev/pkg/installer"
	"github.com/joeblew999/utm-dev/pkg/self/output"
	"github.com/joeblew999/utm-dev/pkg/utils"
)

// BuildOptions contains options for the Build function.
type BuildOptions struct {
	UseLocal  bool // If true, generate bootstrap scripts for local testing
	Obfuscate bool // If true, use garble to obfuscate the binary
}

// Build cross-compiles utm-dev for all supported architectures.
// Generates binaries in the current directory and bootstrap scripts in scripts/.
func Build(opts BuildOptions) error {
	result := output.BuildResult{
		Binaries:         []string{},
		ScriptsGenerated: false,
		LocalMode:        opts.UseLocal,
		Obfuscated:       opts.Obfuscate,
		GarbleInstalled:  false,
	}

	// Get current directory (where utm-dev source is)
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	result.OutputDir = currentDir

	// Check if garble is needed and install if missing
	if opts.Obfuscate {
		if !installer.IsGarbleInstalled() {
			cli.Info("Garble not found, installing...")

			// Create cache for garble installation
			cache, err := utils.NewCacheWithDirectories()
			if err != nil {
				return fmt.Errorf("failed to create cache: %w", err)
			}

			if err := installer.InstallGarble(cache); err != nil {
				return fmt.Errorf("failed to install garble: %w", err)
			}

			result.GarbleInstalled = true
		}
		cli.Info("Building with garble obfuscation...")
	}

	// Build for all supported architectures
	for _, arch := range SupportedArchitectures() {
		outputPath := filepath.Join(currentDir, fmt.Sprintf("utm-dev-%s", arch.Suffix))

		var buildCmd *exec.Cmd
		if opts.Obfuscate {
			// Get garble path from SDK directory
			garblePath, err := installer.GetGarblePath()
			if err != nil {
				return fmt.Errorf("failed to get garble path: %w", err)
			}
			// Use garble build for obfuscation
			buildCmd = exec.Command(garblePath, "build", "-o", outputPath, ".")
		} else {
			// Normal go build
			buildCmd = exec.Command("go", "build", "-o", outputPath, ".")
		}

		buildCmd.Env = append(os.Environ(),
			"GOOS="+arch.GOOS,
			"GOARCH="+arch.GOARCH,
			"CGO_ENABLED=0", // Disable CGO for cross-platform builds
			"GOWORK=off",    // Avoid workspace interference
		)
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr

		if err := buildCmd.Run(); err != nil {
			return fmt.Errorf("failed to build utm-dev for %s/%s: %w", arch.GOOS, arch.GOARCH, err)
		}

		result.Binaries = append(result.Binaries, fmt.Sprintf("utm-dev-%s", arch.Suffix))
	}

	// Generate bootstrap scripts with correct binary names
	if err := generateBootstrapScripts(currentDir, opts); err != nil {
		return fmt.Errorf("failed to generate bootstrap scripts: %w", err)
	}

	result.ScriptsGenerated = true

	// ALWAYS create dist directory with all artifacts
	distDir := filepath.Join(currentDir, DistDir)

	// Create dist directory
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return fmt.Errorf("failed to create %s directory: %w", DistDir, err)
	}

	// Move binaries to dist
	for _, binary := range result.Binaries {
		src := filepath.Join(currentDir, binary)
		dst := filepath.Join(distDir, binary)
		if err := os.Rename(src, dst); err != nil {
			return fmt.Errorf("failed to move %s to %s: %w", binary, DistDir, err)
		}
	}

	// Move bootstrap scripts to dist
	macosScript := filepath.Join(currentDir, MacOSBootstrapScript)
	windowsScript := filepath.Join(currentDir, WindowsBootstrapScript)

	if _, err := os.Stat(macosScript); err == nil {
		dst := filepath.Join(distDir, MacOSBootstrapScript)
		if err := os.Rename(macosScript, dst); err != nil {
			return fmt.Errorf("failed to move %s to %s: %w", MacOSBootstrapScript, DistDir, err)
		}
	}

	if _, err := os.Stat(windowsScript); err == nil {
		dst := filepath.Join(distDir, WindowsBootstrapScript)
		if err := os.Rename(windowsScript, dst); err != nil {
			return fmt.Errorf("failed to move %s to %s: %w", WindowsBootstrapScript, DistDir, err)
		}
	}

	result.OutputDir = distDir
	cli.Success("Release artifacts prepared in: %s", distDir)

	output.OK("self build", result)
	return nil
}

// generateBootstrapScripts creates bootstrap shell/PowerShell scripts
func generateBootstrapScripts(baseDir string, opts BuildOptions) error {
	// Generate scripts in the same directory as binaries
	// Get supported architectures
	allArchs := SupportedArchitectures()
	macOSArchs := ArchsToGoArchList(FilterByOS(allArchs, "darwin"))
	windowsArchs := ArchsToGoArchList(FilterByOS(allArchs, "windows"))

	// Create bootstrap script config
	config := Config{
		Repo:           FullRepoName,
		SupportedArchs: ArchsToString(append(macOSArchs, windowsArchs...)),
		MacOSArchs:     macOSArchs,
		WindowsArchs:   windowsArchs,
		UseLocal:       opts.UseLocal,
		SetupCommand:   SetupCommand, // Single source of truth
	}

	// If local mode, set LocalBinDir
	if opts.UseLocal {
		config.LocalBinDir = baseDir
	}

	return Generate(baseDir, config)
}
