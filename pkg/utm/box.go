package utm

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/joeblew999/utm-dev/pkg/cli"
)

const hcpVagrantBase = "https://api.cloud.hashicorp.com/vagrant/2022-09-30/registry"

// hcpVersionResponse is the relevant part of the HCP versions API response
type hcpVersionResponse struct {
	Versions []struct {
		Version string `json:"version"`
		State   string `json:"state"`
	} `json:"versions"`
}

// hcpDownloadResponse is the HCP download API response
type hcpDownloadResponse struct {
	URL           string `json:"url"`
	Checksum      string `json:"checksum"`
	ChecksumType  string `json:"checksum_type"`
}

// InstallBox downloads and imports a pre-built .utm box using naveenrajm7's
// getbox.sh script (https://naveenrajm7.github.io/utm-gallery/getbox.sh).
// Using the upstream shell script ensures correct extraction and import behaviour.
func InstallBox(vmKey string, force bool) error {
	gallery, err := LoadGallery()
	if err != nil {
		return fmt.Errorf("failed to load gallery: %w", err)
	}

	vm, ok := gallery.GetVM(vmKey)
	if !ok {
		return fmt.Errorf("VM '%s' not found in gallery", vmKey)
	}
	if !vm.IsBoxBased() {
		return fmt.Errorf("VM '%s' is not a box-based VM — use 'utm install %s' for ISO", vmKey, vmKey)
	}

	// Check cache
	if !force && IsISOCached(vmKey) {
		cli.Info("Box '%s' already imported", vm.Box.Name)
		return nil
	}

	// Extract box name (e.g. "utm/windows-11" → "windows-11")
	parts := strings.SplitN(vm.Box.Name, "/", 2)
	boxName := parts[len(parts)-1]

	if err := LaunchUTM(); err != nil {
		return fmt.Errorf("failed to launch UTM: %w", err)
	}

	cli.Info("Downloading and importing %s via getbox.sh (~%.1f GB)...",
		vm.Box.Name, float64(vm.Box.Size)/1024/1024/1024)

	// Use getbox.sh — the upstream script that is known to work correctly
	cmd := exec.Command("sh", "-c",
		fmt.Sprintf("curl -sSf https://naveenrajm7.github.io/utm-gallery/getbox.sh | sh -s %s", boxName))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("getbox.sh failed: %w", err)
	}

	// Set up port forwards for Windows
	if vm.OS == "windows" {
		cli.Info("Configuring network and port forwards...")
		if err := SetupWindowsPortForwards(vm.Name); err != nil {
			cli.Warn("failed to configure port forwards: %v", err)
			cli.Info("Fix manually: utm-dev utm fix-network \"%s\"", vm.Name)
		}
	}

	if err := AddISOToCache(vmKey); err != nil {
		cli.Warn("failed to update cache: %v", err)
	}

	cli.Success("'%s' imported into UTM successfully!", vm.Name)
	cli.Info("Start it with: utm-dev utm up %s", vmKey)
	cli.Info("Connect via RDP: localhost:3389  |  WinRM: localhost:5985")
	cli.Info("Credentials: vagrant / vagrant")
	return nil
}

// resolveBoxVersion returns the latest active version for a box on HCP Vagrant registry
func resolveBoxVersion(namespace, boxName string) (string, error) {
	url := fmt.Sprintf("%s/%s/box/%s/versions", hcpVagrantBase, namespace, boxName)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HCP API returned HTTP %d", resp.StatusCode)
	}

	var result hcpVersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse version response: %w", err)
	}

	for _, v := range result.Versions {
		if v.State == "ACTIVE" {
			return v.Version, nil
		}
	}
	return "", fmt.Errorf("no active version found for %s/%s", namespace, boxName)
}

// getBoxDownloadURL fetches a signed download URL from the HCP Vagrant API
func getBoxDownloadURL(namespace, boxName, version, arch string) (*hcpDownloadResponse, error) {
	url := fmt.Sprintf("%s/%s/box/%s/version/%s/provider/utm/architecture/%s/download",
		hcpVagrantBase, namespace, boxName, version, arch)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HCP API returned HTTP %d for download URL", resp.StatusCode)
	}

	var result hcpDownloadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse download response: %w", err)
	}
	if result.URL == "" {
		return nil, fmt.Errorf("HCP API returned empty download URL")
	}
	return &result, nil
}

// extractUTMFromBox extracts the .utm directory from a Vagrant .box file (tar).
// Returns the path to the extracted .utm directory.
func extractUTMFromBox(boxPath string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "utm-box-*")
	if err != nil {
		return "", err
	}

	f, err := os.Open(boxPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("box is not gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			os.RemoveAll(tmpDir)
			return "", fmt.Errorf("tar read error: %w", err)
		}

		target := filepath.Join(tmpDir, hdr.Name)
		// prevent path traversal
		if !strings.HasPrefix(target, tmpDir) {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				os.RemoveAll(tmpDir)
				return "", err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				os.RemoveAll(tmpDir)
				return "", err
			}
			out, err := os.Create(target)
			if err != nil {
				os.RemoveAll(tmpDir)
				return "", err
			}
			if _, err := io.Copy(out, tr); err != nil {
				out.Close()
				os.RemoveAll(tmpDir)
				return "", err
			}
			out.Close()
		}
	}

	// Find the .utm directory
	var utmPath string
	err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && strings.HasSuffix(path, ".utm") {
			utmPath = path
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		os.RemoveAll(tmpDir)
		return "", err
	}
	if utmPath == "" {
		os.RemoveAll(tmpDir)
		return "", fmt.Errorf("no .utm directory found in box")
	}
	return utmPath, nil
}

// WaitForWindows polls WinRM (http://host:5985/wsman) until Windows actually responds.
// QEMU's port forward listens immediately on start — we need a real HTTP response
// (401 Unauthorized) which only comes once Windows has booted and WinRM is running.
func WaitForWindows(host string, timeout time.Duration) error {
	url := fmt.Sprintf("http://%s/wsman", net.JoinHostPort(host, "5985"))
	client := &http.Client{Timeout: 3 * time.Second}
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			resp.Body.Close()
			// 401 = WinRM is up and demanding auth — Windows is ready
			// Any HTTP response means Windows is running
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timed out waiting for Windows WinRM at %s after %s", url, timeout)
}
