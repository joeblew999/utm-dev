// Package utm provides UTM VM management functionality.
//
// AppleScript automation adapted from:
// https://github.com/naveenrajm7/packer-plugin-utm
// See builder/utm/common/scripts/ for the original AppleScript files.
package utm

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

//go:embed scripts/*
var osascripts embed.FS

// ControllerEnumMap maps human-readable controller names to UTM enum codes
var ControllerEnumMap = map[string]string{
	"ide":    "QdIi",
	"scsi":   "QdIs",
	"sd":     "QdId",
	"mtd":    "QdIm",
	"floppy": "QdIf",
	"pflash": "QdIp",
	"virtio": "QdIv",
	"nvme":   "QdIn",
	"usb":    "QdIu",
}

// NetworkModeEnumMap maps human-readable network modes to UTM enum codes
var NetworkModeEnumMap = map[string]string{
	"shared":   "ShRd", // Shared Network - NAT with host access
	"emulated": "EmUd", // Emulated VLAN - isolated, supports port forwarding
	"bridged":  "BrDg", // Bridged - direct network access
	"host":     "HsOn", // Host Only
}

// GetControllerEnumCode returns the UTM enum code for a controller name
func GetControllerEnumCode(controllerName string) (string, error) {
	code, exists := ControllerEnumMap[controllerName]
	if !exists {
		return "", fmt.Errorf("invalid controller name: %s (valid: none, ide, scsi, sd, mtd, floppy, pflash, virtio, nvme, usb)", controllerName)
	}
	return code, nil
}

// GetNetworkModeEnumCode returns the UTM enum code for a network mode
func GetNetworkModeEnumCode(modeName string) (string, error) {
	code, exists := NetworkModeEnumMap[modeName]
	if !exists {
		return "", fmt.Errorf("invalid network mode: %s (valid: shared, emulated, bridged, host)", modeName)
	}
	return code, nil
}

// ExecuteOsaScript executes an embedded AppleScript with the given arguments.
// The first argument should be the script filename (e.g., "create_vm.applescript").
// Subsequent arguments are passed to the script.
func ExecuteOsaScript(command ...string) (string, error) {
	if len(command) == 0 {
		return "", fmt.Errorf("no command provided")
	}

	// Read the script content from the embedded files
	scriptPath := filepath.Join("scripts", command[0])
	scriptContent, err := osascripts.ReadFile(scriptPath)
	if err != nil {
		return "", fmt.Errorf("failed to read script %s: %w", scriptPath, err)
	}

	// Construct the command to execute
	// osascript - reads from stdin, additional args are passed to the script
	cmd := exec.Command("osascript", "-")

	// Append additional arguments to the command
	if len(command) > 1 {
		cmd.Args = append(cmd.Args, command[1:]...)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, string(scriptContent))
	}()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()

	stdoutString := strings.TrimSpace(stdout.String())
	stderrString := strings.TrimSpace(stderr.String())

	if err != nil {
		// Include stderr in error message for debugging
		if stderrString != "" {
			return stdoutString, fmt.Errorf("osascript error: %w: %s", err, stderrString)
		}
		return stdoutString, fmt.Errorf("osascript error: %w", err)
	}

	return stdoutString, nil
}

// ExtractUUID extracts a UUID from AppleScript output
func ExtractUUID(output string) (string, error) {
	re := regexp.MustCompile(`[0-9a-fA-F-]{36}`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 0 {
		return matches[0], nil
	}
	return "", fmt.Errorf("no UUID found in output: %s", output)
}

// GetUTMVersion returns the installed UTM version using AppleScript
func GetUTMVersion() (string, error) {
	var stdout bytes.Buffer

	cmd := exec.Command("osascript", "-e",
		`tell application "System Events" to return version of application "UTM"`)

	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get UTM version (is UTM installed?): %w", err)
	}

	versionOutput := strings.TrimSpace(stdout.String())

	// Check if the output contains the error message
	if strings.Contains(versionOutput, "get application") {
		return "", fmt.Errorf("UTM is not installed")
	}

	versionRe := regexp.MustCompile(`^(\d+\.\d+\.\d+)$`)
	matches := versionRe.FindStringSubmatch(versionOutput)
	if matches == nil || len(matches) != 2 {
		return "", fmt.Errorf("unexpected UTM version format: %s", versionOutput)
	}

	return matches[1], nil
}

// IsUTMRunning checks if UTM is currently running
func IsUTMRunning() bool {
	cmd := exec.Command("pgrep", "-x", "UTM")
	err := cmd.Run()
	return err == nil
}

// LaunchUTM starts UTM if not already running
func LaunchUTM() error {
	if IsUTMRunning() {
		return nil
	}

	paths := GetPaths()
	cmd := exec.Command("open", "-a", paths.App)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to launch UTM: %w", err)
	}

	return nil
}

// VMBackend represents the UTM backend type (AppleScript enum codes)
type VMBackend string

const (
	// BackendQEMU uses QEMU emulation (works for all architectures)
	BackendQEMU VMBackend = "QeMu"
	// BackendApple uses Apple Virtualization framework (native ARM only)
	BackendApple VMBackend = "ApLe"
)

// VMArch represents the VM architecture
type VMArch string

const (
	ArchARM64 VMArch = "aarch64"
	ArchX64   VMArch = "x86_64"
)

// GetBackendForOS returns the appropriate backend for the given OS/arch
func GetBackendForOS(osType, arch string) VMBackend {
	// Use QEMU for all VMs — Apple VZ has ISO sandbox access issues in UTM 5
	return BackendQEMU
}

// GetArchCode returns the UTM architecture code
func GetArchCode(arch string) VMArch {
	switch arch {
	case "arm64", "aarch64":
		return ArchARM64
	case "amd64", "x86_64":
		return ArchX64
	default:
		return ArchARM64
	}
}
