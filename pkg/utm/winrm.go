package utm

import (
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

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
// For files > 100KB, spins up a temp HTTP server and tells PowerShell to download.
// For small files, uses base64 chunks over WinRM commands.
func PushFileViaWinRM(localPath, remotePath string) error {
	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("failed to stat local file: %w", err)
	}

	// Large files: serve via HTTP, download in VM
	if info.Size() > 100*1024 {
		return pushLargeFileViaHTTP(localPath, remotePath)
	}
	return pushSmallFileViaChunks(localPath, remotePath)
}

// pushLargeFileViaHTTP starts a temp HTTP server and tells the VM to download the file.
func pushLargeFileViaHTTP(localPath, remotePath string) error {
	// Find a free port
	listener, err := net.Listen("tcp", "0.0.0.0:0")
	if err != nil {
		return fmt.Errorf("failed to find free port: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Serve the file
	filename := filepath.Base(localPath)
	mux := http.NewServeMux()
	mux.HandleFunc("/"+filename, func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, localPath)
	})
	server := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: mux}
	go server.ListenAndServe()
	defer server.Close()

	// Small delay for server to start
	time.Sleep(100 * time.Millisecond)

	// The VM reaches the host via the gateway IP (10.0.2.2 for QEMU VLAN)
	// But we're using port-forwarded localhost, so the VM should use the host's IP.
	// For emulated VLAN, the host is reachable at 10.0.2.2.
	url := fmt.Sprintf("http://10.0.2.2:%d/%s", port, filename)

	client, err := newWinRMClient()
	if err != nil {
		return fmt.Errorf("winrm connect failed: %w", err)
	}

	// Use PowerShell to download
	psDownload := fmt.Sprintf(
		`powershell -Command "Invoke-WebRequest -Uri '%s' -OutFile '%s' -UseBasicParsing"`,
		url, remotePath)

	fmt.Printf("  Serving file via HTTP (port %d)...\n", port)
	_, stderr, exitCode, err := client.RunWithString(psDownload, "")
	if err != nil {
		return fmt.Errorf("download in VM failed: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("download in VM failed: %s", stderr)
	}

	return nil
}

// pushSmallFileViaChunks sends small files as base64 chunks over WinRM.
func pushSmallFileViaChunks(localPath, remotePath string) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	client, err := newWinRMClient()
	if err != nil {
		return fmt.Errorf("winrm connect failed: %w", err)
	}

	// Delete target if exists, then create empty file
	psCreate := fmt.Sprintf(
		`powershell -Command "Remove-Item -Path '%s' -Force -ErrorAction SilentlyContinue; New-Item -ItemType File -Path '%s' -Force | Out-Null"`,
		remotePath, remotePath)
	if _, _, _, err := client.RunWithString(psCreate, ""); err != nil {
		return fmt.Errorf("failed to create remote file: %w", err)
	}

	// Write in chunks using append mode
	const chunkSize = 3000
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}
		b64 := base64.StdEncoding.EncodeToString(data[i:end])

		psAppend := fmt.Sprintf(
			`powershell -Command "$b=[Convert]::FromBase64String('%s');$f=[System.IO.File]::Open('%s','Append');$f.Write($b,0,$b.Length);$f.Close()"`,
			b64, remotePath,
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
