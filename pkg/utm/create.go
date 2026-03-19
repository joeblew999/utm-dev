// Package utm provides UTM VM management functionality.
// AppleScript automation adapted from github.com/naveenrajm7/packer-plugin-utm
package utm

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

	if opts.Verbose {
		fmt.Printf("UTM version: %s\n", version)
	}

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
		fmt.Printf("Removing existing VM '%s'...\n", vmName)
		if err := DeleteVMFromUTM(vmName); err != nil {
			fmt.Printf("Warning: failed to delete existing VM: %v\n", err)
		}
	}

	fmt.Printf("Creating VM '%s'...\n", vmName)

	// Step 1: Create VM
	createCmd := []string{
		"create_vm.applescript",
		"--name", vmName,
		"--backend", string(backend),
		"--arch", string(arch),
	}

	if opts.Verbose {
		fmt.Printf("  Running: %s\n", strings.Join(createCmd, " "))
	}

	output, err := ExecuteOsaScript(createCmd...)
	if err != nil {
		return fmt.Errorf("failed to create VM: %w", err)
	}

	vmID, err := ExtractUUID(output)
	if err != nil {
		return fmt.Errorf("failed to get VM ID: %w (output: %s)", err, output)
	}

	if opts.Verbose {
		fmt.Printf("  VM ID: %s\n", vmID)
	}

	// Step 2: Customize VM (CPU, RAM, UEFI)
	fmt.Printf("Configuring hardware (CPU=%d, RAM=%dMB)...\n", vm.Template.CPU, vm.Template.RAM)

	// Determine if UEFI boot is needed (Linux typically needs it)
	uefiBoot := vm.OS == "linux" || vm.OS == "windows"

	customizeCmd := []string{
		"customize_vm.applescript", vmID,
		"--cpus", strconv.Itoa(vm.Template.CPU),
		"--memory", strconv.Itoa(vm.Template.RAM),
		"--name", vmName,
		"--uefi-boot", strconv.FormatBool(uefiBoot),
		"--use-hypervisor", strconv.FormatBool(backend == BackendApple),
	}

	if opts.Verbose {
		fmt.Printf("  Running: %s\n", strings.Join(customizeCmd, " "))
	}

	if _, err := ExecuteOsaScript(customizeCmd...); err != nil {
		// Cleanup on failure
		DeleteVMFromUTM(vmName)
		return fmt.Errorf("failed to customize VM: %w", err)
	}

	// Step 3: Add disk drive
	fmt.Printf("Adding disk (%d GB)...\n", diskSizeMB/1024)

	controllerCode, _ := GetControllerEnumCode("virtio")
	addDriveCmd := []string{
		"add_drive.applescript", vmID,
		"--interface", controllerCode,
		"--size", strconv.Itoa(diskSizeMB),
	}

	if opts.Verbose {
		fmt.Printf("  Running: %s\n", strings.Join(addDriveCmd, " "))
	}

	if _, err := ExecuteOsaScript(addDriveCmd...); err != nil {
		DeleteVMFromUTM(vmName)
		return fmt.Errorf("failed to add disk: %w", err)
	}

	// Step 4: Attach ISO
	fmt.Printf("Attaching ISO...\n")

	// Use USB interface for ISO (widely compatible)
	isoControllerCode, _ := GetControllerEnumCode("usb")
	attachISOCmd := []string{
		"attach_iso.applescript", vmID,
		"--interface", isoControllerCode,
		"--source", isoPath,
	}

	if opts.Verbose {
		fmt.Printf("  Running: %s\n", strings.Join(attachISOCmd, " "))
	}

	if _, err := ExecuteOsaScript(attachISOCmd...); err != nil {
		DeleteVMFromUTM(vmName)
		return fmt.Errorf("failed to attach ISO: %w", err)
	}

	// Step 5: Add network interface (shared network for internet access)
	fmt.Printf("Configuring network...\n")

	networkCode, _ := GetNetworkModeEnumCode("shared")
	addNetworkCmd := []string{
		"add_network_interface.applescript", vmID, networkCode,
	}

	if opts.Verbose {
		fmt.Printf("  Running: %s\n", strings.Join(addNetworkCmd, " "))
	}

	if _, err := ExecuteOsaScript(addNetworkCmd...); err != nil {
		// Network is optional, just warn
		fmt.Printf("Warning: failed to add network interface: %v\n", err)
	}

	fmt.Printf("\n✅ VM '%s' created successfully!\n", vmName)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Start the VM:  utm-dev utm start \"%s\"\n", vmName)
	fmt.Printf("  2. Complete OS installation in the VM window\n")
	fmt.Printf("  3. After installation, eject the ISO from UTM settings\n")
	fmt.Printf("\nShared folder: %s\n", shareDir)
	fmt.Printf("  Mount in guest: sudo mount -t virtiofs share /mnt/share\n")

	return nil
}

// showManualInstructions displays manual VM creation instructions (fallback)
func showManualInstructions(vmKey string, vm *VMEntry, isoPath, shareDir string, diskSizeMB int) error {
	paths := GetPaths()

	fmt.Printf("VM Setup for '%s'\n", vmKey)
	fmt.Printf("══════════════════════════════════════════════════════════\n\n")

	fmt.Printf("Specs from gallery:\n")
	fmt.Printf("  Name: %s\n", vm.Name)
	fmt.Printf("  RAM:  %d MB\n", vm.Template.RAM)
	fmt.Printf("  CPU:  %d cores\n", vm.Template.CPU)
	fmt.Printf("  Disk: %d GB\n", diskSizeMB/1024)
	fmt.Println()

	fmt.Printf("Files prepared:\n")
	fmt.Printf("  ISO:   %s\n", isoPath)
	fmt.Printf("  Share: %s\n", shareDir)
	fmt.Println()

	// Open UTM
	fmt.Println("Opening UTM...")
	LaunchUTM()

	fmt.Println()
	fmt.Printf("Create VM in UTM:\n")
	fmt.Printf("══════════════════════════════════════════════════════════\n")
	fmt.Printf("1. Click + (Create New Virtual Machine)\n")
	fmt.Printf("2. Select: Virtualize → Linux\n")
	fmt.Printf("3. Boot ISO: Browse to:\n")
	fmt.Printf("   %s\n", isoPath)
	fmt.Printf("4. Hardware: RAM=%d MB, CPU=%d cores\n", vm.Template.RAM, vm.Template.CPU)
	fmt.Printf("5. Storage: %d GB\n", diskSizeMB/1024)
	fmt.Printf("6. Shared Directory: Enable and set to:\n")
	fmt.Printf("   %s\n", shareDir)
	fmt.Printf("7. Name: %s\n", vm.Name)
	fmt.Printf("8. Save and Start\n")
	fmt.Println()

	fmt.Printf("After OS installation, access shared files at:\n")
	fmt.Printf("  Host:  %s\n", shareDir)
	fmt.Printf("  Guest: /mnt/share (mount with: sudo mount -t virtiofs share /mnt/share)\n")

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
