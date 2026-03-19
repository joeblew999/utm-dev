// Package adb provides a Go wrapper around Android Debug Bridge (adb) and emulator commands.
// It resolves tool paths from utm-dev's managed SDK directory, so callers don't need
// to worry about PATH configuration.
package adb

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/joeblew999/utm-dev/pkg/config"
)

// Client wraps adb and emulator operations.
type Client struct {
	sdkDir string
}

// New creates a new ADB client using utm-dev's SDK directory.
func New() *Client {
	return &Client{sdkDir: config.GetSDKDir()}
}

// ADBPath returns the absolute path to the adb binary.
func (c *Client) ADBPath() string {
	name := "adb"
	if runtime.GOOS == "windows" {
		name = "adb.exe"
	}
	return filepath.Join(c.sdkDir, "platform-tools", name)
}

// EmulatorPath returns the absolute path to the emulator binary.
func (c *Client) EmulatorPath() string {
	name := "emulator"
	if runtime.GOOS == "windows" {
		name = "emulator.exe"
	}
	return filepath.Join(c.sdkDir, "emulator", name)
}

// Available returns true if adb is installed in the SDK directory.
func (c *Client) Available() bool {
	_, err := os.Stat(c.ADBPath())
	return err == nil
}

// EmulatorAvailable returns true if the emulator is installed.
func (c *Client) EmulatorAvailable() bool {
	_, err := os.Stat(c.EmulatorPath())
	return err == nil
}

// run executes an adb command and returns combined output.
func (c *Client) run(args ...string) (string, error) {
	cmd := exec.Command(c.ADBPath(), args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return out.String(), fmt.Errorf("adb %s: %w\n%s", strings.Join(args, " "), err, out.String())
	}
	return strings.TrimSpace(out.String()), nil
}

// runPassthrough executes an adb command with stdout/stderr connected to the terminal.
func (c *Client) runPassthrough(args ...string) error {
	cmd := exec.Command(c.ADBPath(), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Device represents a connected Android device or emulator.
type Device struct {
	Serial string
	State  string // "device", "offline", "unauthorized"
	Model  string
}

// Devices lists connected Android devices.
func (c *Client) Devices() ([]Device, error) {
	out, err := c.run("devices", "-l")
	if err != nil {
		return nil, err
	}

	var devices []Device
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "List of") {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		d := Device{Serial: parts[0], State: parts[1]}
		for _, p := range parts[2:] {
			if strings.HasPrefix(p, "model:") {
				d.Model = strings.TrimPrefix(p, "model:")
			}
		}
		devices = append(devices, d)
	}
	return devices, nil
}

// HasDevice returns true if at least one device is connected and online.
func (c *Client) HasDevice() bool {
	devices, err := c.Devices()
	if err != nil {
		return false
	}
	for _, d := range devices {
		if d.State == "device" {
			return true
		}
	}
	return false
}

// WaitForDevice blocks until a device is online.
func (c *Client) WaitForDevice() error {
	_, err := c.run("wait-for-device")
	return err
}

// Install installs an APK on the connected device. Replaces existing install.
func (c *Client) Install(apkPath string) error {
	return c.runPassthrough("install", "-r", apkPath)
}

// Uninstall removes an app by package name.
func (c *Client) Uninstall(pkg string) error {
	return c.runPassthrough("uninstall", pkg)
}

// Launch starts an app's main launcher activity by package name.
func (c *Client) Launch(pkg string) error {
	return c.runPassthrough("shell", "monkey", "-p", pkg, "-c", "android.intent.category.LAUNCHER", "1")
}

// ForceStop stops an app by package name.
func (c *Client) ForceStop(pkg string) error {
	_, err := c.run("shell", "am", "force-stop", pkg)
	return err
}

// Screenshot captures the device screen and saves it to a local file.
func (c *Client) Screenshot(outputPath string) error {
	cmd := exec.Command(c.ADBPath(), "exec-out", "screencap", "-p")
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer f.Close()
	cmd.Stdout = f
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		os.Remove(outputPath)
		return fmt.Errorf("screencap: %w", err)
	}
	return nil
}

// Logcat streams filtered logcat output to stdout. Blocks until interrupted.
// tags can be used to filter (e.g., "GoLog:V", "GioView:V").
func (c *Client) Logcat(tags ...string) error {
	args := []string{"logcat", "-v", "time"}
	if len(tags) > 0 {
		args = append(args, "*:S")
		args = append(args, tags...)
	}
	cmd := exec.Command(c.ADBPath(), args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// WebViewVersion returns the Chrome/WebView version on the device.
func (c *Client) WebViewVersion() (string, error) {
	out, err := c.run("shell", "dumpsys", "webviewupdate")
	if err != nil {
		return "", err
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Current WebView package") {
			return line, nil
		}
		if strings.Contains(line, "versionName") {
			return line, nil
		}
	}
	return "unknown", nil
}

// EmulatorList returns the list of available AVD names.
func (c *Client) EmulatorList() ([]string, error) {
	cmd := exec.Command(c.EmulatorPath(), "-list-avds")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("emulator -list-avds: %w", err)
	}
	var avds []string
	for _, line := range strings.Split(out.String(), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			avds = append(avds, line)
		}
	}
	return avds, nil
}

// EmulatorStart launches an emulator AVD in the background.
// Returns the PID of the emulator process.
func (c *Client) EmulatorStart(avdName string) (int, error) {
	cmd := exec.Command(c.EmulatorPath(), "-avd", avdName, "-no-snapshot-load")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start emulator: %w", err)
	}
	return cmd.Process.Pid, nil
}
