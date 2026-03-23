package utm

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// VM represents a UTM virtual machine
type VM struct {
	UUID   string
	Name   string
	Status string
}

// RunUTMCtl executes a utmctl command and returns output
func RunUTMCtl(args ...string) (string, error) {
	utmctl := GetUTMCtlPath()

	cmd := exec.Command(utmctl, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("utmctl %s failed: %w\n%s", strings.Join(args, " "), err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// RunUTMCtlInteractive executes utmctl with stdin/stdout connected
func RunUTMCtlInteractive(args ...string) error {
	utmctl := GetUTMCtlPath()

	cmd := exec.Command(utmctl, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// ListVMs returns all registered VMs
func ListVMs() ([]VM, error) {
	output, err := RunUTMCtl("list")
	if err != nil {
		return nil, err
	}

	var vms []VM
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// utmctl list output format: "UUID    Name    Status"
		// We'll parse this line
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			vm := VM{
				UUID: parts[0],
				Name: strings.Join(parts[1:len(parts)-1], " "),
			}
			if len(parts) >= 3 {
				vm.Status = parts[len(parts)-1]
			}
			vms = append(vms, vm)
		}
	}

	return vms, nil
}

// GetVMStatus returns the status of a VM
func GetVMStatus(vmName string) (string, error) {
	return RunUTMCtl("status", vmName)
}

// StartVM starts a VM
func StartVM(vmName string) error {
	return RunUTMCtlInteractive("start", vmName)
}

// StopVM stops a VM
func StopVM(vmName string) error {
	return RunUTMCtlInteractive("stop", vmName)
}

// GetVMIP returns the IP address of a VM
func GetVMIP(vmName string) (string, error) {
	return RunUTMCtl("ip-address", vmName)
}

// ExecInVM executes a command in a VM.
// Tries WinRM first (reliable), falls back to utmctl exec (needs QEMU Guest Agent).
func ExecInVM(vmName string, command string) error {
	if IsWinRMAvailable() {
		return ExecViaWinRM(command)
	}
	return RunUTMCtlInteractive("exec", vmName, "--cmd", command)
}

// CloneVM creates a clone of a VM
func CloneVM(vmName, newName string) error {
	return RunUTMCtlInteractive("clone", vmName, "--name", newName)
}

// DeleteVM deletes a VM (no confirmation!)
func DeleteVM(vmName string) error {
	return RunUTMCtlInteractive("delete", vmName)
}

// PushFile pushes a file to a VM.
// Tries WinRM first, falls back to utmctl file push.
func PushFile(vmName, localPath, remotePath string) error {
	if IsWinRMAvailable() {
		return PushFileViaWinRM(localPath, remotePath)
	}

	utmctl := GetUTMCtlPath()
	file, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer file.Close()

	cmd := exec.Command(utmctl, "file", "push", vmName, remotePath)
	cmd.Stdin = file
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// PullFile pulls a file from a VM
func PullFile(vmName, remotePath, localPath string) error {
	utmctl := GetUTMCtlPath()

	// Create local file
	file, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("failed to create local file: %w", err)
	}
	defer file.Close()

	cmd := exec.Command(utmctl, "file", "pull", vmName, remotePath)
	cmd.Stdout = file
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
