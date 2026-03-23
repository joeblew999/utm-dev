// Package utm provides UTM VM management functionality.
// AppleScript automation adapted from github.com/naveenrajm7/packer-plugin-utm
package utm

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joeblew999/utm-dev/pkg/cli"
)

// CreateVMOptions contains options for VM creation
type CreateVMOptions struct {
	// Force recreation if VM already exists
	Force bool

	// Manual mode - show instructions instead of automating
	Manual bool

	// Verbose output
	Verbose bool
}

// CreateVM creates a VM from gallery template using AppleScript automation
func CreateVM(vmKey string, opts CreateVMOptions) error {
	gallery, err := LoadGallery()
	if err != nil {
		return fmt.Errorf("failed to load gallery: %w", err)
	}

	vm, ok := gallery.GetVM(vmKey)
	if !ok {
		return fmt.Errorf("VM '%s' not found in gallery. Run 'utm-dev utm gallery' to see available VMs", vmKey)
	}

	paths := GetPaths()

	// Check if ISO is downloaded
	isoPath := filepath.Join(paths.ISO, vm.ISO.Filename)
	if _, err := os.Stat(isoPath); os.IsNotExist(err) {
		return fmt.Errorf("ISO not found. Run 'utm-dev utm install %s' first", vmKey)
	}

	// Ensure directories exist
	if err := EnsureGlobalDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Create per-VM share folder
	shareDir := filepath.Join(paths.Share, vmKey)
	if err := os.MkdirAll(shareDir, 0755); err != nil {
		return fmt.Errorf("failed to create share directory: %w", err)
	}

	// Compute disk size in MB
	diskSizeMB := vm.Template.Disk
	if diskSizeMB < 20480 { // minimum 20GB
		diskSizeMB = 20480
	}

	// If manual mode requested, show instructions
	if opts.Manual {
		return showManualInstructions(vmKey, vm, isoPath, shareDir, diskSizeMB)
	}

	// Automated creation using AppleScript
	return createVMAutomated(vmKey, vm, isoPath, shareDir, diskSizeMB, opts)
}

// createVMAutomated creates a VM using AppleScript automation
func createVMAutomated(vmKey string, vm *VMEntry, isoPath, shareDir string, diskSizeMB int, opts CreateVMOptions) error {
	// Check UTM version
	version, err := GetUTMVersion()
	if err != nil {
		return fmt.Errorf("failed to get UTM version: %w\nMake sure UTM is installed. Run 'utm-dev utm install' to install it", err)
	}

	cli.Debug("UTM version: %s", version)

	// Launch UTM if not running
	if err := LaunchUTM(); err != nil {
		return fmt.Errorf("failed to launch UTM: %w", err)
	}

	// Determine backend and architecture
	backend := GetBackendForOS(vm.OS, vm.Arch)
	arch := GetArchCode(vm.Arch)

	vmName := vm.Name

	// Check if VM already exists
	if VMExistsInUTM(vmName) {
		if !opts.Force {
			return fmt.Errorf("VM '%s' already exists. Use --force to recreate", vmName)
		}
		cli.Info("Removing existing VM '%s'...", vmName)
		if err := DeleteVMFromUTM(vmName); err != nil {
			cli.Warn("failed to delete existing VM: %v", err)
		}
	}

	cli.Info("Creating VM '%s'...", vmName)

	// Step 1: Create VM
	createCmd := []string{
		"create_vm.applescript",
		"--name", vmName,
		"--backend", string(backend),
		"--arch", string(arch),
	}

	cli.Debug("  Running: %s", strings.Join(createCmd, " "))

	output, err := ExecuteOsaScript(createCmd...)
	if err != nil {
		return fmt.Errorf("failed to create VM: %w", err)
	}

	vmID, err := ExtractUUID(output)
	if err != nil {
		return fmt.Errorf("failed to get VM ID: %w (output: %s)", err, output)
	}

	cli.Debug("  VM ID: %s", vmID)

	// Step 2: Customize VM (CPU, RAM, UEFI)
	cli.Info("Configuring hardware (CPU=%d, RAM=%dMB)...", vm.Template.CPU, vm.Template.RAM)

	customizeCmd := []string{
		"customize_vm.applescript", vmID,
		"--cpus", strconv.Itoa(vm.Template.CPU),
		"--memory", strconv.Itoa(vm.Template.RAM),
		"--name", vmName,
	}
	// QEMU-only options
	if backend == BackendQEMU {
		uefiBoot := vm.OS == "linux" || vm.OS == "windows"
		customizeCmd = append(customizeCmd, "--uefi-boot", strconv.FormatBool(uefiBoot))
		// Enable hypervisor (HVF) for ARM64 guests on Apple Silicon — near-native speed
		useHypervisor := vm.Arch == "arm64" || vm.Arch == "aarch64"
		customizeCmd = append(customizeCmd, "--use-hypervisor", strconv.FormatBool(useHypervisor))
	}

	cli.Debug("  Running: %s", strings.Join(customizeCmd, " "))

	if _, err := ExecuteOsaScript(customizeCmd...); err != nil {
		// Cleanup on failure
		DeleteVMFromUTM(vmName)
		return fmt.Errorf("failed to customize VM: %w", err)
	}

	// Step 3: Add disk drive
	cli.Info("Adding disk (%d GB)...", diskSizeMB/1024)

	// Windows needs nvme — no virtio-blk driver in the Windows installer
	diskInterface := "virtio"
	if vm.OS == "windows" {
		diskInterface = "nvme"
	}
	controllerCode, _ := GetControllerEnumCode(diskInterface)
	addDriveCmd := []string{
		"add_drive.applescript", vmID,
		"--interface", controllerCode,
		"--size", strconv.Itoa(diskSizeMB),
	}

	cli.Debug("  Running: %s", strings.Join(addDriveCmd, " "))

	if _, err := ExecuteOsaScript(addDriveCmd...); err != nil {
		DeleteVMFromUTM(vmName)
		return fmt.Errorf("failed to add disk: %w", err)
	}

	// Step 4: Attach ISO
	cli.Info("Attaching ISO...")

	// Use USB interface for ISO (widely compatible)
	isoControllerCode, _ := GetControllerEnumCode("usb")
	attachISOCmd := []string{
		"attach_iso.applescript", vmID,
		"--interface", isoControllerCode,
		"--source", isoPath,
	}

	cli.Debug("  Running: %s", strings.Join(attachISOCmd, " "))

	if _, err := ExecuteOsaScript(attachISOCmd...); err != nil {
		DeleteVMFromUTM(vmName)
		return fmt.Errorf("failed to attach ISO: %w", err)
	}

	// Step 5: Add network interface (shared network for internet access)
	cli.Info("Configuring network...")

	networkCode, _ := GetNetworkModeEnumCode("shared")
	addNetworkCmd := []string{
		"add_network_interface.applescript", vmID, networkCode,
	}

	cli.Debug("  Running: %s", strings.Join(addNetworkCmd, " "))

	if _, err := ExecuteOsaScript(addNetworkCmd...); err != nil {
		// Network is optional, just warn
		cli.Warn("failed to add network interface: %v", err)
	}

	cli.Success("VM '%s' created successfully!", vmName)
	cli.Info("\nNext steps:")
	cli.Info("  1. Start the VM:  utm-dev utm start \"%s\"", vmName)
	cli.Info("  2. Complete OS installation in the VM window")
	cli.Info("  3. After installation, eject the ISO from UTM settings")
	cli.Info("\nShared folder: %s", shareDir)
	if vm.OS == "windows" {
		cli.Info("  Access in guest: \\\\mac\\share (auto-mounted via SPICE WebDAV)")
	} else {
		cli.Info("  Mount in guest: sudo mount -t virtiofs share /mnt/share")
	}

	return nil
}

// showManualInstructions displays manual VM creation instructions (fallback)
func showManualInstructions(vmKey string, vm *VMEntry, isoPath, shareDir string, diskSizeMB int) error {
	paths := GetPaths()

	cli.Info("VM Setup for '%s'", vmKey)
	cli.Info("══════════════════════════════════════════════════════════")

	cli.Info("\nSpecs from gallery:")
	cli.Info("  Name: %s", vm.Name)
	cli.Info("  RAM:  %d MB", vm.Template.RAM)
	cli.Info("  CPU:  %d cores", vm.Template.CPU)
	cli.Info("  Disk: %d GB", diskSizeMB/1024)
	fmt.Println()

	cli.Info("Files prepared:")
	cli.Info("  ISO:   %s", isoPath)
	cli.Info("  Share: %s", shareDir)
	fmt.Println()

	// Open UTM
	cli.Info("Opening UTM...")
	LaunchUTM()

	fmt.Println()
	cli.Info("Create VM in UTM:")
	cli.Info("══════════════════════════════════════════════════════════")
	cli.Info("1. Click + (Create New Virtual Machine)")
	if vm.OS == "windows" {
		cli.Info("2. Select: Virtualize → Windows")
		cli.Info("3. Boot ISO: Browse to:")
		cli.Info("   %s", isoPath)
		cli.Info("4. Hardware: RAM=%d MB, CPU=%d cores", vm.Template.RAM, vm.Template.CPU)
		cli.Info("5. Storage: %d GB", diskSizeMB/1024)
		cli.Info("6. Shared Directory: Enable and set to:")
		cli.Info("   %s", shareDir)
		cli.Info("7. Name: %s", vm.Name)
		cli.Info("8. Save and Start")
		fmt.Println()
		cli.Info("After OS installation, access shared files at:")
		cli.Info("  Host:  %s", shareDir)
		cli.Info("  Guest: \\\\mac\\share (auto-mounted via SPICE WebDAV)")
	} else {
		cli.Info("2. Select: Virtualize → Linux")
		cli.Info("3. Boot ISO: Browse to:")
		cli.Info("   %s", isoPath)
		cli.Info("4. Hardware: RAM=%d MB, CPU=%d cores", vm.Template.RAM, vm.Template.CPU)
		cli.Info("5. Storage: %d GB", diskSizeMB/1024)
		cli.Info("6. Shared Directory: Enable and set to:")
		cli.Info("   %s", shareDir)
		cli.Info("7. Name: %s", vm.Name)
		cli.Info("8. Save and Start")
		fmt.Println()
		cli.Info("After OS installation, access shared files at:")
		cli.Info("  Host:  %s", shareDir)
		cli.Info("  Guest: /mnt/share (mount with: sudo mount -t virtiofs share /mnt/share)")
	}

	_ = paths // unused but kept for consistency
	return nil
}

// VMExists checks if a VM exists (file system check)
func VMExists(vmKey string) bool {
	paths := GetPaths()
	vmPath := filepath.Join(paths.VMs, vmKey+".utm")
	if _, err := os.Stat(vmPath); err == nil {
		return true
	}
	return false
}

// VMExistsInUTM checks if a VM exists in UTM by name
func VMExistsInUTM(vmName string) bool {
	// Use utmctl to list VMs and check if name exists
	output, err := RunUTMCtl("list")
	if err != nil {
		return false
	}
	// UTM list format: UUID    Status    Name
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, vmName) {
			return true
		}
	}
	return false
}

// DeleteVMFromUTM deletes a VM from UTM by name
func DeleteVMFromUTM(vmName string) error {
	_, err := RunUTMCtl("delete", vmName)
	return err
}
