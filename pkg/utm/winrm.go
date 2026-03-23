package utm

import (
	"encoding/base64"
	"fmt"
	"os"

	"github.com/masterzen/winrm"
)

// WinRM defaults for vagrant boxes
const (
	winrmHost     = "127.0.0.1"
	winrmPort     = 5985
	winrmUser     = "vagrant"
	winrmPassword = "vagrant"
)

func newWinRMClient() (*winrm.Client, error) {
	endpoint := winrm.NewEndpoint(winrmHost, winrmPort, false, true, nil, nil, nil, 0)
	return winrm.NewClient(endpoint, winrmUser, winrmPassword)
}

// ExecViaWinRM runs a command in the VM over WinRM.
func ExecViaWinRM(command string) error {
	client, err := newWinRMClient()
	if err != nil {
		return fmt.Errorf("winrm connect failed: %w", err)
	}
	exitCode, err := client.Run(command, os.Stdout, os.Stderr)
	if err != nil {
		return fmt.Errorf("winrm exec failed: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("command exited with code %d", exitCode)
	}
	return nil
}

// ExecViaWinRMOutput runs a command and returns stdout as a string.
func ExecViaWinRMOutput(command string) (string, error) {
	client, err := newWinRMClient()
	if err != nil {
		return "", fmt.Errorf("winrm connect failed: %w", err)
	}
	stdout, stderr, exitCode, err := client.RunWithString(command, "")
	if err != nil {
		return "", fmt.Errorf("winrm exec failed: %w", err)
	}
	if exitCode != 0 {
		return "", fmt.Errorf("command exited with code %d: %s", exitCode, stderr)
	}
	return stdout, nil
}

// PushFileViaWinRM copies a local file to the VM using WinRM + PowerShell.
func PushFileViaWinRM(localPath, remotePath string) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	client, err := newWinRMClient()
	if err != nil {
		return fmt.Errorf("winrm connect failed: %w", err)
	}

	// Create file first
	psCreate := fmt.Sprintf(`powershell -Command "New-Item -ItemType File -Path '%s' -Force | Out-Null"`, remotePath)
	if _, _, _, err := client.RunWithString(psCreate, ""); err != nil {
		return fmt.Errorf("failed to create remote file: %w", err)
	}

	// Write in chunks via PowerShell to avoid WinRM message size limits
	const chunkSize = 6000 // Safe size for base64 in a WinRM command
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		b64 := base64.StdEncoding.EncodeToString(data[i:end])

		psAppend := fmt.Sprintf(
			`powershell -Command "$bytes = [Convert]::FromBase64String('%s'); `+
				`[System.IO.File]::WriteAllBytes('%s', `+
				`([byte[]][System.IO.File]::ReadAllBytes('%s') + $bytes))"`,
			b64, remotePath, remotePath,
		)
		_, stderr, exitCode, err := client.RunWithString(psAppend, "")
		if err != nil {
			return fmt.Errorf("failed to write chunk: %w", err)
		}
		if exitCode != 0 {
			return fmt.Errorf("failed to write chunk: %s", stderr)
		}
	}

	return nil
}

// IsWinRMAvailable checks if WinRM is responding.
func IsWinRMAvailable() bool {
	client, err := newWinRMClient()
	if err != nil {
		return false
	}
	_, _, _, err = client.RunWithString("echo ok", "")
	return err == nil
}
