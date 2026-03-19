package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joeblew999/utm-dev/pkg/utm"
	"github.com/spf13/cobra"
)

var utmCmd = &cobra.Command{
	Use:   "utm",
	Short: "Control UTM virtual machines",
	Long: `Control UTM virtual machines using utmctl.

This command is a wrapper around utmctl for convenient VM automation.
Requires UTM to be installed and QEMU guest agent running in the VM.

Examples:
  # List all VMs
  utm-dev utm list

  # List available VMs from gallery
  utm-dev utm gallery

  # Check VM status
  utm-dev utm status "Windows 11"

  # Execute command in VM
  utm-dev utm exec "Windows 11" -- build windows examples/hybrid-dashboard

  # Execute Task in VM
  utm-dev utm task "Windows 11" build:hybrid:windows

  # Pull file from VM
  utm-dev utm pull "Windows 11" "/path/in/vm/file.txt" ./local/

  # Push file to VM
  utm-dev utm push "Windows 11" ./local/file.txt "/path/in/vm/"`,
}

var utmListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all UTM virtual machines",
	RunE: func(cmd *cobra.Command, args []string) error {
		return utm.RunUTMCtlInteractive("list")
	},
}

var utmGalleryCmd = &cobra.Command{
	Use:   "gallery",
	Short: "List available VMs from the gallery",
	Long: `List VMs available in the gallery for installation.

The gallery contains pre-configured VM definitions for common operating systems
including Windows 11 ARM, Ubuntu, Debian, and Fedora.

Examples:
  utm-dev utm gallery
  utm-dev utm gallery --os windows
  utm-dev utm gallery --arch arm64`,
	RunE: func(cmd *cobra.Command, args []string) error {
		gallery, err := utm.LoadGallery()
		if err != nil {
			return fmt.Errorf("failed to load gallery: %w", err)
		}

		osFilter, _ := cmd.Flags().GetString("os")
		archFilter, _ := cmd.Flags().GetString("arch")

		vms := gallery.VMs

		// Apply filters
		if osFilter != "" {
			vms = gallery.FilterByOS(osFilter)
		}
		if archFilter != "" {
			filtered := make(map[string]utm.VMEntry)
			for k, v := range vms {
				if v.Arch == archFilter {
					filtered[k] = v
				}
			}
			vms = filtered
		}

		if len(vms) == 0 {
			fmt.Println("No VMs match the filter criteria")
			return nil
		}

		fmt.Println("Available VMs in gallery:")
		fmt.Println()
		for key, vm := range vms {
			fmt.Printf("  %s\n", key)
			fmt.Printf("    Name: %s\n", vm.Name)
			fmt.Printf("    OS:   %s (%s)\n", vm.OS, vm.Arch)
			if vm.Description != "" {
				fmt.Printf("    Desc: %s\n", vm.Description)
			}
			fmt.Printf("    RAM:  %d MB, Disk: %d MB, CPU: %d\n",
				vm.Template.RAM, vm.Template.Disk, vm.Template.CPU)
			if vm.ISO.URL != "" {
				sizeGB := float64(vm.ISO.Size) / 1024 / 1024 / 1024
				fmt.Printf("    ISO:  %.1f GB\n", sizeGB)
			}
			fmt.Println()
		}

		return nil
	},
}

var utmPathsCmd = &cobra.Command{
	Use:   "paths",
	Short: "Show UTM paths configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		paths := utm.GetPaths()
		fmt.Println("UTM Paths:")
		fmt.Printf("  App:   %s\n", paths.App)
		fmt.Printf("  VMs:   %s\n", paths.VMs)
		fmt.Printf("  ISO:   %s\n", paths.ISO)
		fmt.Printf("  Share: %s\n", paths.Share)
		fmt.Println()
		fmt.Printf("utmctl: %s\n", utm.GetUTMCtlPath())
		fmt.Printf("Installed: %v\n", utm.IsUTMInstalled())
		return nil
	},
}

var utmInstallCmd = &cobra.Command{
	Use:   "install [vm-key]",
	Short: "Install UTM app or download VM ISO",
	Long: `Install the UTM application or download a VM ISO from the gallery.

Without arguments, installs the UTM application.
With a VM key, downloads the ISO for that VM.

Examples:
  # Install UTM app
  utm-dev utm install

  # Download Windows 11 ISO
  utm-dev utm install windows-11-arm

  # Force reinstall UTM
  utm-dev utm install --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		if len(args) == 0 {
			// Install UTM app
			return utm.InstallUTM(force)
		}

		// Download ISO for specified VM
		return utm.DownloadISO(args[0], force)
	},
}

var utmUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall UTM app",
	RunE: func(cmd *cobra.Command, args []string) error {
		return utm.UninstallUTM()
	},
}

var utmDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check UTM installation status",
	RunE: func(cmd *cobra.Command, args []string) error {
		status, err := utm.GetInstallStatus()
		if err != nil {
			return err
		}

		fmt.Println("UTM Installation Status:")
		fmt.Println()

		if status.Installed {
			fmt.Printf("  ✓ UTM installed at %s\n", status.InstalledPath)
			if status.InstalledVersion != "" {
				fmt.Printf("    Version: %s\n", status.InstalledVersion)
			}
		} else {
			fmt.Printf("  ✗ UTM not installed\n")
			fmt.Printf("    Run: utm-dev utm install\n")
		}

		fmt.Printf("  Gallery version: %s\n", status.GalleryVersion)

		if status.UpdateAvailable {
			fmt.Printf("  ⚠ Update available: %s\n", status.GalleryVersion)
			fmt.Printf("    Run: utm-dev utm install --force\n")
		}

		// Show driver capabilities
		if status.Installed {
			driver, err := utm.NewDriver()
			if err == nil {
				fmt.Println()
				fmt.Println("Capabilities:")
				capStr := func(supported bool) string {
					if supported {
						return "✓"
					}
					return "✗"
				}
				fmt.Printf("  %s Export/Import (UTM 4.6+)\n", capStr(driver.SupportsExport()))
				fmt.Printf("  %s Guest Tools (UTM 4.6+)\n", capStr(driver.SupportsGuestTools()))
			}
		}

		// Check directories
		paths := utm.GetPaths()
		fmt.Println()
		fmt.Println("Directories:")
		checkDir("VMs", paths.VMs)
		checkDir("ISO", paths.ISO)
		checkDir("Share", paths.Share)

		return nil
	},
}

func checkDir(name, path string) {
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("  ✓ %s: %s\n", name, path)
	} else {
		fmt.Printf("  ✗ %s: %s (missing)\n", name, path)
	}
}

var utmStatusCmd = &cobra.Command{
	Use:     "status <vm-name>",
	Aliases: []string{"st"},
	Short:   "Get status of a VM",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		status, err := utm.GetVMStatus(args[0])
		if err != nil {
			return err
		}
		fmt.Println(status)
		return nil
	},
}

var utmStartCmd = &cobra.Command{
	Use:   "start <vm-name>",
	Short: "Start a VM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return utm.StartVM(args[0])
	},
}

var utmStopCmd = &cobra.Command{
	Use:   "stop <vm-name>",
	Short: "Stop a VM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return utm.StopVM(args[0])
	},
}

var utmIPCmd = &cobra.Command{
	Use:   "ip <vm-name>",
	Short: "Get IP address of a VM",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ip, err := utm.GetVMIP(args[0])
		if err != nil {
			return err
		}
		fmt.Println(ip)
		return nil
	},
}

var utmExecCmd = &cobra.Command{
	Use:   "exec <vm-name> -- <command> [args...]",
	Short: "Execute a utm-dev command in the VM",
	Long: `Execute a utm-dev command in the VM.

The command after '--' will be prefixed with 'utm-dev' automatically.

Examples:
  # Build for Windows
  utm-dev utm exec "Windows 11" -- build windows examples/hybrid-dashboard

  # Generate icons
  utm-dev utm exec "Windows 11" -- icons examples/hybrid-dashboard

  # Check config
  utm-dev utm exec "Windows 11" -- config`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 2 {
			return fmt.Errorf("usage: utm-dev utm exec <vm-name> -- <command> [args...]")
		}

		vmName := args[0]

		// Find the -- separator
		dashIndex := -1
		for i, arg := range args {
			if arg == "--" {
				dashIndex = i
				break
			}
		}

		if dashIndex == -1 || dashIndex == len(args)-1 {
			return fmt.Errorf("missing command after '--'")
		}

		// Build utm-dev command
		goupCommand := append([]string{"utm-dev"}, args[dashIndex+1:]...)
		cmdStr := strings.Join(goupCommand, " ")

		fmt.Printf("Executing in VM '%s': %s\n\n", vmName, cmdStr)

		return utm.ExecInVM(vmName, cmdStr)
	},
}

var utmTaskCmd = &cobra.Command{
	Use:   "task <vm-name> <task-name>",
	Short: "Execute a Taskfile task in the VM",
	Long: `Execute a Taskfile task in the VM.

This is a convenience wrapper around 'task <taskname>'.

Examples:
  utm-dev utm task "Windows 11" build:hybrid:windows
  utm-dev utm task "Windows 11" test:all`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		taskName := args[1]

		fmt.Printf("Executing task '%s' in VM '%s'\n\n", taskName, vmName)

		cmdStr := fmt.Sprintf("task %s", taskName)
		return utm.ExecInVM(vmName, cmdStr)
	},
}

var utmPullCmd = &cobra.Command{
	Use:   "pull <vm-name> <remote-path> <local-path>",
	Short: "Pull a file from the VM to local machine",
	Long: `Pull a file from the VM to local machine.

Examples:
  # Pull MSIX from VM
  utm-dev utm pull "Windows 11" "C:\\Users\\User\\utm-dev\\examples\\hybrid-dashboard\\.bin\\hybrid-dashboard.msix" ./artifacts/

  # Pull build log
  utm-dev utm pull "Windows 11" "/tmp/build.log" ./logs/`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		remotePath := args[1]
		localPath := args[2]

		fmt.Printf("Pulling from VM '%s':\n", vmName)
		fmt.Printf("  Remote: %s\n", remotePath)
		fmt.Printf("  Local:  %s\n\n", localPath)

		if err := utm.PullFile(vmName, remotePath, localPath); err != nil {
			return err
		}

		fmt.Printf("✓ File pulled successfully\n")
		return nil
	},
}

var utmPushCmd = &cobra.Command{
	Use:   "push <vm-name> <local-path> <remote-path>",
	Short: "Push a file from local machine to the VM",
	Long: `Push a file from local machine to the VM.

Examples:
  # Push config file
  utm-dev utm push "Windows 11" ./config.json "C:\\Users\\User\\config.json"

  # Push test data
  utm-dev utm push "Windows 11" ./test-data.zip "/tmp/test-data.zip"`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		localPath := args[1]
		remotePath := args[2]

		fmt.Printf("Pushing to VM '%s':\n", vmName)
		fmt.Printf("  Local:  %s\n", localPath)
		fmt.Printf("  Remote: %s\n\n", remotePath)

		if err := utm.PushFile(vmName, localPath, remotePath); err != nil {
			return err
		}

		fmt.Printf("✓ File pushed successfully\n")
		return nil
	},
}

var utmCreateCmd = &cobra.Command{
	Use:   "create <vm-key>",
	Short: "Create a VM from gallery template (automated)",
	Long: `Create a new UTM virtual machine from the gallery template.

This uses UTM's AppleScript API to fully automate VM creation including:
  - Creating the VM with correct backend (QEMU/Apple)
  - Configuring CPU, RAM, and UEFI settings
  - Adding disk drive with appropriate size
  - Attaching the boot ISO
  - Configuring network interface

AppleScript automation adapted from github.com/naveenrajm7/packer-plugin-utm

Prerequisites:
  1. UTM must be installed: utm-dev utm install
  2. ISO must be downloaded: utm-dev utm install <vm-key>
  3. UTM Automation permission granted in System Settings

Examples:
  # Create Debian ARM64 VM (automated)
  utm-dev utm create debian-13-arm

  # Create with verbose output
  utm-dev utm create debian-13-arm --verbose

  # Force recreate existing VM
  utm-dev utm create debian-13-arm --force

  # Use manual mode (shows instructions instead of automating)
  utm-dev utm create debian-13-arm --manual`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")
		manual, _ := cmd.Flags().GetBool("manual")
		verbose, _ := cmd.Flags().GetBool("verbose")

		opts := utm.CreateVMOptions{
			Force:   force,
			Manual:  manual,
			Verbose: verbose,
		}
		return utm.CreateVM(args[0], opts)
	},
}

var utmMigrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate UTM files from local to global location",
	Long: `Migrate UTM.app and ISOs from local repo paths to global SDK location.

This moves:
  .bin/UTM.app     -> ~/utm-dev-sdks/utm/UTM.app
  .data/utm/iso/*  -> ~/utm-dev-sdks/utm/iso/

VMs and share directories remain local (project-specific).

The migration is idempotent - running it multiple times is safe.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return utm.MigrateAll()
	},
}

var utmExportCmd = &cobra.Command{
	Use:   "export <vm-name> <output-path>",
	Short: "Export a VM to a .utm file (UTM 4.6+)",
	Long: `Export a virtual machine to a .utm file for sharing or backup.

The exported file can be imported on another machine or used as a template.
Requires UTM 4.6 or later.

Examples:
  # Export to current directory
  utm-dev utm export "Debian 13 Trixie" ./debian-template.utm

  # Export to specific path
  utm-dev utm export "Windows 11" ~/vm-templates/windows-dev.utm`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		outputPath := args[1]

		fmt.Printf("Exporting VM '%s' to %s...\n", vmName, outputPath)

		if err := utm.ExportVM(vmName, outputPath); err != nil {
			return err
		}

		fmt.Printf("✓ VM exported successfully\n")
		return nil
	},
}

var utmImportCmd = &cobra.Command{
	Use:   "import <utm-file>",
	Short: "Import a VM from a .utm file (UTM 4.6+)",
	Long: `Import a virtual machine from a .utm file.

This creates a new VM from the exported template.
Requires UTM 4.6 or later.

Examples:
  # Import a VM template
  utm-dev utm import ./debian-template.utm

  # Import from absolute path
  utm-dev utm import ~/vm-templates/windows-dev.utm`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		utmPath := args[0]

		fmt.Printf("Importing VM from %s...\n", utmPath)

		vmID, err := utm.ImportVM(utmPath)
		if err != nil {
			return err
		}

		fmt.Printf("✓ VM imported successfully (ID: %s)\n", vmID)
		return nil
	},
}

var utmScreenshotCmd = &cobra.Command{
	Use:   "screenshot <vm-name> [output-file]",
	Short: "Capture a screenshot from the VM",
	Long: `Capture a screenshot from a running VM.

Uses PowerShell to capture the screen inside the VM, then pulls the
image back to the host. Does NOT require utm-dev to be installed in the VM.

The VM must have the QEMU guest agent running.

Examples:
  # Capture screenshot from Windows VM
  utm-dev utm screenshot "Windows 11" windows-screenshot.png

  # Capture with default filename
  utm-dev utm screenshot "Windows 11"`,
	Args: cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		output := "utm-screenshot.png"
		if len(args) > 1 {
			output = args[1]
		}

		// Remote temp path in the VM
		remotePath := "C:\\Users\\User\\utm-dev-screenshot.png"

		// Use PowerShell to capture screenshot (no utm-dev needed in VM)
		// This uses .NET's System.Drawing to capture the primary screen
		psScript := fmt.Sprintf(`powershell -Command "Add-Type -AssemblyName System.Windows.Forms; Add-Type -AssemblyName System.Drawing; $bounds = [System.Windows.Forms.Screen]::PrimaryScreen.Bounds; $bmp = New-Object System.Drawing.Bitmap($bounds.Width, $bounds.Height); $graphics = [System.Drawing.Graphics]::FromImage($bmp); $graphics.CopyFromScreen($bounds.Location, [System.Drawing.Point]::Empty, $bounds.Size); $bmp.Save('%s'); $graphics.Dispose(); $bmp.Dispose()"`, remotePath)

		fmt.Printf("Capturing screenshot in VM '%s'...\n", vmName)
		if err := utm.ExecInVM(vmName, psScript); err != nil {
			// Fallback: try utm-dev if PowerShell fails
			fmt.Println("PowerShell screenshot failed, trying utm-dev in VM...")
			goupCmd := fmt.Sprintf("utm-dev screenshot --force %s", remotePath)
			if err2 := utm.ExecInVM(vmName, goupCmd); err2 != nil {
				return fmt.Errorf("screenshot failed in VM: %w (PowerShell: %v)", err2, err)
			}
		}

		// Pull the screenshot back to host
		fmt.Printf("Pulling screenshot to %s...\n", output)
		if err := utm.PullFile(vmName, remotePath, output); err != nil {
			return fmt.Errorf("failed to pull screenshot: %w", err)
		}

		// Clean up remote file
		cleanupCmd := fmt.Sprintf("del %s", remotePath)
		_ = utm.ExecInVM(vmName, cleanupCmd) // Best-effort cleanup

		fmt.Printf("✓ Screenshot saved to %s\n", output)
		return nil
	},
}

var utmRunCmd = &cobra.Command{
	Use:   "run <vm-name> <app-directory>",
	Short: "Build app for Windows and run it in the VM",
	Long: `Cross-compile a Gio application for Windows, push it to the VM, and run it.

This automates the full workflow:
1. Build the app for Windows (cross-compile on macOS)
2. Push the binary to the VM
3. Launch it inside the VM

The VM must have the QEMU guest agent running.

Examples:
  # Build and run hybrid-dashboard in Windows VM
  utm-dev utm run "Windows 11" examples/hybrid-dashboard

  # Build and run webviewer example
  utm-dev utm run "Windows 11" examples/gio-plugin-webviewer`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		appDir := args[1]

		// Build for Windows
		fmt.Printf("Building %s for Windows...\n", appDir)
		buildCmdStr := fmt.Sprintf("utm-dev build windows %s", appDir)
		buildExec := exec.Command("sh", "-c", buildCmdStr)
		buildExec.Stdout = os.Stdout
		buildExec.Stderr = os.Stderr
		if err := buildExec.Run(); err != nil {
			return fmt.Errorf("Windows build failed: %w", err)
		}

		// Determine the output binary path
		appName := filepath.Base(appDir)
		localBinary := filepath.Join(appDir, ".bin", "windows", appName+".exe")

		// Check binary exists
		if _, err := os.Stat(localBinary); os.IsNotExist(err) {
			return fmt.Errorf("built binary not found at %s", localBinary)
		}

		// Push binary to VM
		remoteBinary := fmt.Sprintf("C:\\Users\\User\\%s.exe", appName)
		fmt.Printf("Pushing %s to VM '%s'...\n", localBinary, vmName)
		if err := utm.PushFile(vmName, localBinary, remoteBinary); err != nil {
			return fmt.Errorf("failed to push binary: %w", err)
		}

		// Run in VM
		fmt.Printf("Launching %s in VM...\n", appName)
		if err := utm.ExecInVM(vmName, remoteBinary); err != nil {
			return fmt.Errorf("failed to run in VM: %w", err)
		}

		fmt.Printf("✓ App running in VM '%s'\n", vmName)
		return nil
	},
}

var utmBuildCmd = &cobra.Command{
	Use:   "build <vm-name> [platform] <app-directory>",
	Short: "Build an app inside the VM natively",
	Long: `Build a Gio application inside the VM using the VM's native toolchain.

This is different from 'utm run' which cross-compiles on macOS. This command
executes the build inside the VM, producing a native binary. Useful when
cross-compilation isn't sufficient (e.g., CGO dependencies, native libraries).

Requires utm-dev and Go to be installed in the VM.
Use 'utm-dev utm exec <vm> -- self setup' to install the toolchain.

Examples:
  # Build hybrid-dashboard for Windows inside the VM
  utm-dev utm build "Windows 11" windows examples/hybrid-dashboard

  # Build with platform auto-detected from VM OS
  utm-dev utm build "Windows 11" examples/hybrid-dashboard`,
	Args: cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		var platform, appDir string

		if len(args) == 3 {
			platform = args[1]
			appDir = args[2]
		} else {
			// Auto-detect platform: assume Windows for now
			platform = "windows"
			appDir = args[1]
		}

		// Build the utm-dev command to run inside the VM
		buildCmd := fmt.Sprintf("utm-dev build %s %s", platform, appDir)

		fmt.Printf("Building %s for %s in VM '%s'...\n", appDir, platform, vmName)
		if err := utm.ExecInVM(vmName, buildCmd); err != nil {
			return fmt.Errorf("build in VM failed: %w", err)
		}

		fmt.Printf("✓ Built %s for %s in VM '%s'\n", appDir, platform, vmName)

		// Optionally pull the built binary back to host
		pull, _ := cmd.Flags().GetBool("pull")
		if pull {
			appName := filepath.Base(appDir)
			var remoteBinary, localBinary string

			switch platform {
			case "windows":
				remoteBinary = fmt.Sprintf("%s\\.bin\\windows\\%s.exe", appDir, appName)
				localBinary = filepath.Join(appDir, ".bin", "windows", appName+".exe")
			default:
				remoteBinary = fmt.Sprintf("%s/.bin/%s/%s", appDir, platform, appName)
				localBinary = filepath.Join(appDir, ".bin", platform, appName)
			}

			fmt.Printf("Pulling %s to %s...\n", remoteBinary, localBinary)
			if err := utm.PullFile(vmName, remoteBinary, localBinary); err != nil {
				return fmt.Errorf("failed to pull binary: %w", err)
			}
			fmt.Printf("✓ Binary pulled to %s\n", localBinary)
		}

		return nil
	},
}

var utmPortForwardCmd = &cobra.Command{
	Use:   "port-forward <vm-name> <guest-port> <host-port>",
	Short: "Set up port forwarding for a VM",
	Long: `Set up port forwarding from host to guest VM.

This allows you to access services running in the VM from your host machine.
The VM must have an emulated VLAN network interface configured.

Note: Port forwarding only works with "Emulated VLAN" network mode, not "Shared Network".
Use --setup-network to automatically configure the required network interfaces.

Examples:
  # Forward SSH (guest:22 -> host:2222)
  utm-dev utm port-forward "Debian 13 Trixie" 22 2222

  # Forward with network setup (adds emulated VLAN if needed)
  utm-dev utm port-forward "Debian 13 Trixie" 22 2222 --setup-network

  # Forward HTTP
  utm-dev utm port-forward "Windows 11" 80 8080`,
	Args: cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]

		guestPort, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid guest port: %s", args[1])
		}

		hostPort, err := strconv.Atoi(args[2])
		if err != nil {
			return fmt.Errorf("invalid host port: %s", args[2])
		}

		setupNetwork, _ := cmd.Flags().GetBool("setup-network")

		// Setup network if requested
		if setupNetwork {
			fmt.Printf("Setting up emulated network for '%s'...\n", vmName)
			if err := utm.SetupEmulatedNetwork(vmName); err != nil {
				return fmt.Errorf("failed to setup network: %w", err)
			}
		}

		protocol, _ := cmd.Flags().GetString("protocol")

		rule := utm.PortForward{
			Protocol:     protocol,
			GuestAddress: "",
			GuestPort:    guestPort,
			HostAddress:  "127.0.0.1",
			HostPort:     hostPort,
		}

		// Network index 1 is typically the emulated VLAN interface
		networkIndex, _ := cmd.Flags().GetInt("network-index")

		fmt.Printf("Adding port forward: localhost:%d -> %s:%d (%s)\n",
			hostPort, vmName, guestPort, protocol)

		if err := utm.AddPortForward(vmName, networkIndex, rule); err != nil {
			return err
		}

		fmt.Printf("✓ Port forwarding configured\n")
		fmt.Printf("  Access via: %s://localhost:%d\n", protocol, hostPort)
		return nil
	},
}

func init() {
	// Command group for help organization
	utmCmd.GroupID = "vm"

	rootCmd.AddCommand(utmCmd)
	utmCmd.AddCommand(utmListCmd)
	utmCmd.AddCommand(utmGalleryCmd)
	utmCmd.AddCommand(utmPathsCmd)
	utmCmd.AddCommand(utmInstallCmd)
	utmCmd.AddCommand(utmUninstallCmd)
	utmCmd.AddCommand(utmDoctorCmd)
	utmCmd.AddCommand(utmStatusCmd)
	utmCmd.AddCommand(utmStartCmd)
	utmCmd.AddCommand(utmStopCmd)
	utmCmd.AddCommand(utmIPCmd)
	utmCmd.AddCommand(utmExecCmd)
	utmCmd.AddCommand(utmTaskCmd)
	utmCmd.AddCommand(utmPullCmd)
	utmCmd.AddCommand(utmPushCmd)
	utmCmd.AddCommand(utmMigrateCmd)
	utmCmd.AddCommand(utmCreateCmd)
	utmCmd.AddCommand(utmExportCmd)
	utmCmd.AddCommand(utmImportCmd)
	utmCmd.AddCommand(utmPortForwardCmd)
	utmCmd.AddCommand(utmScreenshotCmd)
	utmCmd.AddCommand(utmRunCmd)
	utmCmd.AddCommand(utmBuildCmd)

	// Build flags
	utmBuildCmd.Flags().Bool("pull", false, "Pull built binary back to host after building")

	// Create flags
	utmCreateCmd.Flags().Bool("force", false, "Force recreate VM if exists")
	utmCreateCmd.Flags().Bool("manual", false, "Show manual instructions instead of automating")
	utmCreateCmd.Flags().BoolP("verbose", "v", false, "Verbose output")

	// Gallery filters
	utmGalleryCmd.Flags().String("os", "", "Filter by OS (windows, linux)")
	utmGalleryCmd.Flags().String("arch", "", "Filter by architecture (arm64, amd64)")

	// Install flags
	utmInstallCmd.Flags().Bool("force", false, "Force reinstall/redownload")

	// Port forward flags
	utmPortForwardCmd.Flags().String("protocol", "tcp", "Protocol (tcp or udp)")
	utmPortForwardCmd.Flags().Int("network-index", 1, "Network interface index (1 = emulated VLAN)")
	utmPortForwardCmd.Flags().Bool("setup-network", false, "Setup emulated network if not configured")
}
