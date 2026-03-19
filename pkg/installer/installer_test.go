package installer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/joeblew999/utm-dev/pkg/config"
)

func TestResolveInstallPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		{
			name:     "empty path uses SDK dir",
			input:    "",
			expected: config.GetSDKDir(),
			wantErr:  false,
		},
		{
			name:     "sdks/ prefix uses OS SDK dir",
			input:    "sdks/android-31",
			expected: filepath.Join(config.GetSDKDir(), "android-31"),
			wantErr:  false,
		},
		{
			name:     "absolute path returned as-is",
			input:    "/opt/android-sdk",
			expected: "/opt/android-sdk",
			wantErr:  false,
		},
		{
			name:     "relative path uses current directory",
			input:    "local-sdk",
			expected: "", // We'll check this dynamically
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ResolveInstallPath(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ResolveInstallPath() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ResolveInstallPath() unexpected error: %v", err)
				return
			}

			if tt.name == "relative path uses current directory" {
				// For relative paths, check that it's made absolute
				if !filepath.IsAbs(result) {
					t.Errorf("ResolveInstallPath() with relative path should return absolute path, got %q", result)
				}
				if !strings.HasSuffix(result, tt.input) {
					t.Errorf("ResolveInstallPath() with relative path should contain input %q, got %q", tt.input, result)
				}
			} else {
				if result != tt.expected {
					t.Errorf("ResolveInstallPath() = %q, want %q", result, tt.expected)
				}
			}
		})
	}
}

func TestResolveInstallPathUsesOSLevel(t *testing.T) {
	testCases := []string{
		"",
		"sdks/android-31",
		"sdks/build-tools/31.0.0",
	}

	for _, testCase := range testCases {
		t.Run(testCase, func(t *testing.T) {
			result, err := ResolveInstallPath(testCase)
			if err != nil {
				t.Fatalf("ResolveInstallPath(%q) failed: %v", testCase, err)
			}

			// Should always return an absolute path
			if !filepath.IsAbs(result) {
				t.Errorf("ResolveInstallPath(%q) should return absolute path, got %q", testCase, result)
			}

			// Should use the OS-level SDK directory
			sdkDir := config.GetSDKDir()
			if testCase == "" || strings.HasPrefix(testCase, "sdks/") {
				if !strings.HasPrefix(result, sdkDir) {
					t.Errorf("ResolveInstallPath(%q) should use OS SDK dir %q, got %q", testCase, sdkDir, result)
				}
			}

			// Should NOT contain workspace-relative patterns
			workspaceRelativeMarkers := []string{"./", "../"}
			for _, marker := range workspaceRelativeMarkers {
				if strings.Contains(result, marker) {
					t.Errorf("ResolveInstallPath(%q) should not contain workspace-relative marker %q, got %q", testCase, marker, result)
				}
			}
		})
	}
}

func TestResolveInstallPathExpandsEnvVars(t *testing.T) {
	// Set a test environment variable
	testEnvVar := "GOUP_TEST_PATH"
	testValue := "/test/path"
	os.Setenv(testEnvVar, testValue)
	defer os.Unsetenv(testEnvVar)

	input := "$" + testEnvVar + "/sdk"
	result, err := ResolveInstallPath(input)
	if err != nil {
		t.Fatalf("ResolveInstallPath() failed: %v", err)
	}

	expected := testValue + "/sdk"
	if result != expected {
		t.Errorf("ResolveInstallPath() = %q, want %q", result, expected)
	}
}

func TestSDKInstallationUsesOSLevelPaths(t *testing.T) {
	// Create a test SDK
	sdk := &SDK{
		Name:        "test-sdk",
		Version:     "1.0.0",
		URL:         "", // No URL means manual installation
		Checksum:    "",
		InstallPath: "sdks/test-sdk",
	}

	// Test that ResolveInstallPath returns OS-level path
	resolvedPath, err := ResolveInstallPath(sdk.InstallPath)
	if err != nil {
		t.Fatalf("ResolveInstallPath() failed: %v", err)
	}

	// Should be under OS-level SDK directory
	expectedPrefix := config.GetSDKDir()
	if !strings.HasPrefix(resolvedPath, expectedPrefix) {
		t.Errorf("Resolved path %q should start with OS SDK dir %q", resolvedPath, expectedPrefix)
	}

	// Should be absolute
	if !filepath.IsAbs(resolvedPath) {
		t.Errorf("Resolved path %q should be absolute", resolvedPath)
	}
}
