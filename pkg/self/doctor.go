package self

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/joeblew999/utm-dev/pkg/self/output"
)

// Doctor validates the installation and all dependencies
func Doctor() error {
	result := output.DoctorResult{
		Installations: []output.InstallationInfo{},
		Dependencies:  []output.DependencyInfo{},
		Issues:        []string{},
		Suggestions:   []string{},
	}

	// Check utm-dev itself - look for ALL installations
	installations := findAllInstallations()

	if len(installations) == 0 {
		result.Issues = append(result.Issues, "utm-dev not found in PATH")
		result.Suggestions = append(result.Suggestions, "Run: curl -sSL https://github.com/joeblew999/utm-dev/releases/latest/download/macos-bootstrap.sh | bash")
	} else {
		for i, path := range installations {
			info := output.InstallationInfo{
				Path:     path,
				Active:   i == 0,
				Shadowed: i > 0,
			}
			result.Installations = append(result.Installations, info)
		}

		if len(installations) > 1 {
			result.Issues = append(result.Issues, "Multiple utm-dev installations found")
			for i, path := range installations {
				if i > 0 {
					result.Suggestions = append(result.Suggestions, "Remove: "+path)
				}
			}
		}
	}

	// Check platform-specific package manager
	switch runtime.GOOS {
	case "darwin":
		result.Dependencies = append(result.Dependencies, checkDep("Homebrew", "brew", "--version"))
	case "windows":
		result.Dependencies = append(result.Dependencies, checkDep("winget", "winget", "--version"))
	}

	// Check git
	gitDep := checkDep("git", "git", "--version")
	result.Dependencies = append(result.Dependencies, gitDep)
	if !gitDep.Installed {
		result.Issues = append(result.Issues, "git not installed")
		result.Suggestions = append(result.Suggestions, "Install git")
	}

	// Check go
	goDep := checkDep("go", "go", "version")
	result.Dependencies = append(result.Dependencies, goDep)
	if !goDep.Installed {
		result.Issues = append(result.Issues, "go not installed")
		result.Suggestions = append(result.Suggestions, "Install go")
	}

	// Check mise
	miseDep := checkDep("mise", "mise", "version")
	result.Dependencies = append(result.Dependencies, miseDep)
	if !miseDep.Installed {
		result.Issues = append(result.Issues, "mise not installed")
		result.Suggestions = append(result.Suggestions, "Install mise: curl -fsSL https://mise.run | sh")
	}

	output.OK("self doctor", result)
	return nil
}

// checkDep checks if a dependency is installed and gets its version
func checkDep(name, command string, args ...string) output.DependencyInfo {
	dep := output.DependencyInfo{
		Name:      name,
		Installed: false,
	}

	cmd := exec.Command(command, args...)
	out, err := cmd.CombinedOutput()
	if err == nil {
		dep.Installed = true
		// Extract version from output (first line usually)
		lines := strings.Split(string(out), "\n")
		if len(lines) > 0 {
			dep.Version = strings.TrimSpace(lines[0])
		}
	}

	return dep
}

// findAllInstallations finds all utm-dev binaries in PATH
func findAllInstallations() []string {
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

// checkCommand checks if a command exists and runs successfully (for backward compatibility)
func checkCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}
