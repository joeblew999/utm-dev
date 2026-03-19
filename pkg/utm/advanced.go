// Package utm provides UTM VM management functionality.
// Advanced features: port forwarding, export/import
//
// Patterns adapted from:
// https://github.com/naveenrajm7/packer-plugin-utm
package utm

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ProtocolEnumMap maps protocol names to UTM enum codes
var ProtocolEnumMap = map[string]string{
	"tcp": "TcPp",
	"udp": "UdPp",
}

// PortForward represents a port forwarding rule
type PortForward struct {
	Protocol     string // "tcp" or "udp"
	GuestAddress string // Guest IP (usually empty for any)
	GuestPort    int    // Port inside VM
	HostAddress  string // Host IP (usually "127.0.0.1")
	HostPort     int    // Port on host
}

// AddPortForward adds a port forwarding rule to a VM
// Requires the VM to have an "emulated" network interface (index 1 typically)
func AddPortForward(vmName string, networkIndex int, rule PortForward) error {
	// Get VM UUID
	vmID, err := GetVMUUID(vmName)
	if err != nil {
		return fmt.Errorf("failed to get VM UUID: %w", err)
	}

	// Get protocol enum code
	protocolCode, ok := ProtocolEnumMap[strings.ToLower(rule.Protocol)]
	if !ok {
		return fmt.Errorf("invalid protocol: %s (use tcp or udp)", rule.Protocol)
	}

	// Format: "protocol,guestAddress,guestPort,hostAddress,hostPort"
	ruleStr := fmt.Sprintf("%s,%s,%d,%s,%d",
		protocolCode,
		rule.GuestAddress,
		rule.GuestPort,
		rule.HostAddress,
		rule.HostPort,
	)

	cmd := []string{
		"add_port_forwards.applescript", vmID,
		"--index", strconv.Itoa(networkIndex),
		ruleStr,
	}

	_, err = ExecuteOsaScript(cmd...)
	if err != nil {
		return fmt.Errorf("failed to add port forward: %w", err)
	}

	return nil
}

// SetupSSHPortForward sets up SSH port forwarding for a VM
// This is a convenience function for the common SSH use case
func SetupSSHPortForward(vmName string, hostPort int) error {
	rule := PortForward{
		Protocol:     "tcp",
		GuestAddress: "",
		GuestPort:    22,
		HostAddress:  "127.0.0.1",
		HostPort:     hostPort,
	}

	// Network index 1 is typically the emulated VLAN interface
	return AddPortForward(vmName, 1, rule)
}

// ClearPortForwards removes all port forwarding rules from a VM's network interface
func ClearPortForwards(vmName string, networkIndex int) error {
	vmID, err := GetVMUUID(vmName)
	if err != nil {
		return fmt.Errorf("failed to get VM UUID: %w", err)
	}

	cmd := []string{
		"clear_port_forwards.applescript", vmID,
		"--index", strconv.Itoa(networkIndex),
	}

	_, err = ExecuteOsaScript(cmd...)
	if err != nil {
		return fmt.Errorf("failed to clear port forwards: %w", err)
	}

	return nil
}

// ExportVM exports a VM to a .utm file (UTM 4.6+)
func ExportVM(vmName, outputPath string) error {
	vmID, err := GetVMUUID(vmName)
	if err != nil {
		return fmt.Errorf("failed to get VM UUID: %w", err)
	}

	// Ensure output path is absolute
	absPath, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("failed to resolve output path: %w", err)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Export via AppleScript
	var stdout bytes.Buffer
	cmd := exec.Command(
		"osascript", "-e",
		fmt.Sprintf(`tell application "UTM" to export virtual machine id "%s" to POSIX file "%s"`, vmID, absPath),
	)
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to export VM: %w", err)
	}

	return nil
}

// ImportVM imports a VM from a .utm file (UTM 4.6+)
// Returns the new VM's UUID
func ImportVM(utmPath string) (string, error) {
	// Ensure path is absolute
	absPath, err := filepath.Abs(utmPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path: %w", err)
	}

	// Check file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return "", fmt.Errorf("UTM file not found: %s", absPath)
	}

	// Import via AppleScript
	var stdout bytes.Buffer
	cmd := exec.Command(
		"osascript", "-e",
		fmt.Sprintf(`tell application "UTM" to import new virtual machine from POSIX file "%s"`, absPath),
	)
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to import VM: %w", err)
	}

	// Extract VM ID from output
	output := stdout.String()
	re := regexp.MustCompile(`virtual machine id ([A-F0-9-]+)`)
	matches := re.FindStringSubmatch(output)
	if len(matches) > 1 {
		return matches[1], nil
	}

	return "", fmt.Errorf("failed to get imported VM ID from output: %s", output)
}

// GetVMUUID returns the UUID for a VM by name
func GetVMUUID(vmName string) (string, error) {
	output, err := RunUTMCtl("list")
	if err != nil {
		return "", fmt.Errorf("failed to list VMs: %w", err)
	}

	// Parse output: UUID    Status    Name
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, vmName) {
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				return parts[0], nil
			}
		}
	}

	return "", fmt.Errorf("VM not found: %s", vmName)
}

// GetGuestToolsISOPath returns the path to UTM's guest tools ISO
func GetGuestToolsISOPath() (string, error) {
	// Default path where UTM downloads guest tools
	guestToolsPath := filepath.Join(
		os.Getenv("HOME"),
		"Library/Containers/com.utmapp.UTM/Data/Library/Application Support/GuestSupportTools/utm-guest-tools-latest.iso",
	)

	if _, err := os.Stat(guestToolsPath); os.IsNotExist(err) {
		return "", fmt.Errorf("guest tools ISO not found at: %s", guestToolsPath)
	}

	return guestToolsPath, nil
}

// SetupWindowsPortForwards configures shared + emulated VLAN network with RDP/WinRM
// port forwards on a Windows VM. Retries to handle the delay after import.
func SetupWindowsPortForwards(vmName string) error {
	// Retry — utmctl list may not see a freshly imported VM immediately
	var vmID string
	var err error
	for i := 0; i < 6; i++ {
		vmID, err = GetVMUUID(vmName)
		if err == nil {
			break
		}
		time.Sleep(3 * time.Second)
	}
	if err != nil {
		return fmt.Errorf("VM '%s' not found after import: %w", vmName, err)
	}

	// Clear existing interfaces
	if _, err := ExecuteOsaScript("clear_network_interfaces.applescript", vmID); err != nil {
		return fmt.Errorf("failed to clear network interfaces: %w", err)
	}
	// Add shared (internet)
	sharedCode, _ := GetNetworkModeEnumCode("shared")
	if _, err := ExecuteOsaScript("add_network_interface.applescript", vmID, sharedCode); err != nil {
		return fmt.Errorf("failed to add shared network: %w", err)
	}
	// Add emulated VLAN (port forwards)
	emulatedCode, _ := GetNetworkModeEnumCode("emulated")
	if _, err := ExecuteOsaScript("add_network_interface.applescript", vmID, emulatedCode); err != nil {
		return fmt.Errorf("failed to add emulated network: %w", err)
	}
	// Add RDP + WinRM port forwards on the emulated interface (index 1)
	for _, fwd := range []PortForward{
		{Protocol: "tcp", GuestPort: 3389, HostAddress: "127.0.0.1", HostPort: 3389},
		{Protocol: "tcp", GuestPort: 5985, HostAddress: "127.0.0.1", HostPort: 5985},
	} {
		if err := AddPortForward(vmName, 1, fwd); err != nil {
			return fmt.Errorf("failed to add port forward %d: %w", fwd.GuestPort, err)
		}
	}
	return nil
}

// SetupEmulatedNetwork adds an emulated VLAN network interface for port forwarding
// This is needed before adding port forwards
func SetupEmulatedNetwork(vmName string) error {
	vmID, err := GetVMUUID(vmName)
	if err != nil {
		return fmt.Errorf("failed to get VM UUID: %w", err)
	}

	// First clear existing network interfaces
	if _, err := ExecuteOsaScript("clear_network_interfaces.applescript", vmID); err != nil {
		return fmt.Errorf("failed to clear network interfaces: %w", err)
	}

	// Add shared network (index 0) for internet access
	networkCode, _ := GetNetworkModeEnumCode("shared")
	if _, err := ExecuteOsaScript("add_network_interface.applescript", vmID, networkCode); err != nil {
		return fmt.Errorf("failed to add shared network: %w", err)
	}

	// Add emulated VLAN (index 1) for port forwarding
	emulatedCode, _ := GetNetworkModeEnumCode("emulated")
	if _, err := ExecuteOsaScript("add_network_interface.applescript", vmID, emulatedCode); err != nil {
		return fmt.Errorf("failed to add emulated network: %w", err)
	}

	return nil
}
