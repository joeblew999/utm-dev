package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joeblew999/utm-dev/pkg/self"
	"github.com/joeblew999/utm-dev/pkg/utils"
)

func TestBootstrapScriptGenerationLocal(t *testing.T) {
	// Test LOCAL mode bootstrap script generation
	projectRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}

	// Go up one level to project root if we're in cmd/
	if filepath.Base(projectRoot) == "cmd" {
		projectRoot = filepath.Dir(projectRoot)
	}

	config := self.Config{
		Repo:           "joeblew999/utm-dev",
		SupportedArchs: "darwin-amd64, darwin-arm64, linux-amd64, linux-arm64, windows-amd64, windows-arm64",
		MacOSArchs:     []string{"amd64", "arm64"},
		WindowsArchs:   []string{"amd64", "arm64"},
		UseLocal:       true,
		LocalBinDir:    projectRoot,
		SetupCommand:   self.SetupCommand,
	}

	// Test macOS script generation
	t.Run("macOS LOCAL mode", func(t *testing.T) {
		script, err := self.GenerateMacOSScript(config)
		if err != nil {
			t.Fatalf("failed to generate macOS script: %v", err)
		}

		// Verify script contains LOCAL mode marker
		if !strings.Contains(script, "LOCAL MODE") {
			t.Error("script should contain LOCAL MODE marker")
		}

		// Verify it uses self setup command
		if !strings.Contains(script, "self setup") {
			t.Error("script should use 'self setup' command")
		}

		// Verify it references the local binary directory
		if !strings.Contains(script, config.LocalBinDir) {
			t.Error("script should reference LocalBinDir")
		}

		// Verify script structure
		if !strings.Contains(script, "#!/usr/bin/env bash") {
			t.Error("script should have bash shebang")
		}
		if !strings.Contains(script, "set -euo pipefail") {
			t.Error("script should use strict mode")
		}
	})

	// Test Windows script generation
	t.Run("Windows LOCAL mode", func(t *testing.T) {
		script, err := self.GenerateWindowsScript(config)
		if err != nil {
			t.Fatalf("failed to generate Windows script: %v", err)
		}

		// Verify script contains LOCAL mode marker
		if !strings.Contains(script, "LOCAL MODE") {
			t.Error("script should contain LOCAL MODE marker")
		}

		// Verify it uses bootstrap command
		if !strings.Contains(script, "bootstrap") && !strings.Contains(script, "install") {
			t.Error("script should use 'bootstrap install' command")
		}

		// Verify script structure
		if !strings.Contains(script, "#Requires -RunAsAdministrator") {
			t.Error("script should require admin privileges")
		}
		if !strings.Contains(script, "$ErrorActionPreference") {
			t.Error("script should set error action preference")
		}
	})
}

func TestBootstrapScriptGenerationRemote(t *testing.T) {
	// Test REMOTE mode bootstrap script generation
	config := self.Config{
		Repo:           "joeblew999/utm-dev",
		SupportedArchs: "darwin-amd64, darwin-arm64, linux-amd64, linux-arm64, windows-amd64, windows-arm64",
		MacOSArchs:     []string{"amd64", "arm64"},
		WindowsArchs:   []string{"amd64", "arm64"},
		UseLocal:       false, // REMOTE mode
		SetupCommand:   self.SetupCommand,
	}

	// Test macOS script generation
	t.Run("macOS REMOTE mode", func(t *testing.T) {
		script, err := self.GenerateMacOSScript(config)
		if err != nil {
			t.Fatalf("failed to generate macOS script: %v", err)
		}

		// Verify script contains RELEASE MODE marker
		if !strings.Contains(script, "RELEASE MODE") {
			t.Error("script should contain RELEASE MODE marker")
		}

		// Verify it downloads from GitHub
		if !strings.Contains(script, "api.github.com") {
			t.Error("script should download from GitHub API")
		}

		// Verify it uses self setup command
		if !strings.Contains(script, "self setup") {
			t.Error("script should use 'self setup' command")
		}

		// Should not contain LOCAL mode references
		if strings.Contains(script, "LOCAL MODE") {
			t.Error("script should not contain LOCAL MODE in REMOTE mode")
		}
	})

	// Test Windows script generation
	t.Run("Windows REMOTE mode", func(t *testing.T) {
		script, err := self.GenerateWindowsScript(config)
		if err != nil {
			t.Fatalf("failed to generate Windows script: %v", err)
		}

		// Verify script contains RELEASE MODE marker
		if !strings.Contains(script, "RELEASE MODE") {
			t.Error("script should contain RELEASE MODE marker")
		}

		// Verify it downloads from GitHub
		if !strings.Contains(script, "api.github.com") {
			t.Error("script should download from GitHub API")
		}

		// Should not contain LOCAL mode references
		if strings.Contains(script, "LOCAL MODE") {
			t.Error("script should not contain LOCAL MODE in REMOTE mode")
		}
	})
}

func TestSupportedArchitectures(t *testing.T) {
	// Test that self.SupportedArchitectures returns expected platforms
	archs := self.SupportedArchitectures()

	if len(archs) == 0 {
		t.Fatal("self.SupportedArchitectures should return at least one architecture")
	}

	expectedCount := 6 // darwin-amd64, darwin-arm64, linux-amd64, linux-arm64, windows-amd64, windows-arm64
	if len(archs) != expectedCount {
		t.Errorf("expected %d architectures, got %d", expectedCount, len(archs))
	}

	// Verify each architecture is valid
	for _, arch := range archs {
		if err := arch.Validate(); err != nil {
			t.Errorf("invalid architecture %v: %v", arch, err)
		}

		// Verify binary name format
		binaryName := arch.BinaryName()
		if !strings.HasPrefix(binaryName, "utm-dev-") {
			t.Errorf("binary name should start with 'utm-dev-', got: %s", binaryName)
		}

		// Verify Windows binaries have .exe extension
		if arch.GOOS == "windows" && !strings.HasSuffix(binaryName, ".exe") {
			t.Errorf("Windows binary should have .exe extension: %s", binaryName)
		}
	}

	// Test self.FilterByOS
	darwinArchs := self.FilterByOS(archs, "darwin")
	if len(darwinArchs) != 2 {
		t.Errorf("expected 2 darwin architectures, got %d", len(darwinArchs))
	}

	windowsArchs := self.FilterByOS(archs, "windows")
	if len(windowsArchs) != 2 {
		t.Errorf("expected 2 windows architectures, got %d", len(windowsArchs))
	}

	// Test self.ArchsToGoArchList
	goArchList := self.ArchsToGoArchList(darwinArchs)
	if len(goArchList) != 2 {
		t.Errorf("expected 2 GOARCH values, got %d", len(goArchList))
	}
	if !utils.Contains(goArchList, "amd64") || !utils.Contains(goArchList, "arm64") {
		t.Errorf("expected amd64 and arm64, got %v", goArchList)
	}
}

func TestConfigValidation(t *testing.T) {
	// Test Config validation
	t.Run("valid config", func(t *testing.T) {
		config := self.Config{
			Repo:           "joeblew999/utm-dev",
			SupportedArchs: "darwin-arm64",
			MacOSArchs:     []string{"arm64"},
			WindowsArchs:   []string{"amd64"},
			UseLocal:       false,
			SetupCommand:   self.SetupCommand,
		}
		if err := config.Validate(); err != nil {
			t.Errorf("valid config should not error: %v", err)
		}
	})

	t.Run("missing repo", func(t *testing.T) {
		config := self.Config{
			SupportedArchs: "darwin-arm64",
		}
		if err := config.Validate(); err == nil {
			t.Error("config without Repo should error")
		}
	})

	t.Run("LOCAL mode missing LocalBinDir", func(t *testing.T) {
		config := self.Config{
			Repo:     "joeblew999/utm-dev",
			UseLocal: true,
			// Missing LocalBinDir
		}
		if err := config.Validate(); err == nil {
			t.Error("LOCAL mode config without LocalBinDir should error")
		}
	})
}
