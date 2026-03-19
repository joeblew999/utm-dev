// +build screenshot

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/joeblew999/utm-dev/pkg/logging"
	"github.com/joeblew999/utm-dev/pkg/screenshot"
	"github.com/spf13/cobra"
)

// generateTitleVariations creates different possible window title variations from a directory name
// Example: "hybrid-dashboard" -> ["hybrid", "dashboard", "Hybrid", "Dashboard", "Hybrid Dashboard"]
func generateTitleVariations(dirName string) []string {
	variations := []string{}

	// Split by hyphen and underscore
	parts := strings.FieldsFunc(dirName, func(r rune) bool {
		return r == '-' || r == '_'
	})

	// Add individual parts (lowercase)
	for _, part := range parts {
		if len(part) > 2 { // Skip very short parts
			variations = append(variations, part)
		}
	}

	// Add title-cased version (e.g., "Hybrid Dashboard")
	titleParts := make([]string, len(parts))
	for i, part := range parts {
		if len(part) > 0 {
			titleParts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	titleCase := strings.Join(titleParts, " ")
	if titleCase != "" {
		variations = append(variations, titleCase)
	}

	// Add the original directory name as last resort
	variations = append(variations, dirName)

	return variations
}

var runAndCaptureCmd = &cobra.Command{
	Use:   "run-and-capture <app-dir> <output-file>",
	Short: "Run Gio app and capture screenshot",
	Long: `Run a Gio application, wait for its window to appear, and capture a screenshot.

This automates the workflow of:
1. Launch the app
2. Wait for window to appear
3. Optionally resize window (if --preset or --width/--height specified)
4. Capture screenshot of the window
5. Stop the app

Examples:
  # Run app and capture screenshot
  utm-dev run-and-capture examples/hybrid-dashboard screenshot.png

  # Load a specific URL (writes temporary app.json)
  utm-dev run-and-capture --url https://example.com examples/gio-plugin-webviewer screenshot.png

  # Use App Store preset size
  utm-dev run-and-capture --preset macos-retina examples/hybrid-dashboard screenshot.png`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		appDir := args[0]
		output := args[1]

		// Get flags
		presetName, _ := cmd.Flags().GetString("preset")
		width, _ := cmd.Flags().GetInt("width")
		height, _ := cmd.Flags().GetInt("height")
		quality, _ := cmd.Flags().GetInt("quality")
		waitTime, _ := cmd.Flags().GetInt("wait")
		urlOverride, _ := cmd.Flags().GetString("url")

		// Initialize structured logger (DEV role — this is utm-dev itself)
		log, err := logging.New(logging.Config{
			AppName: "run-and-capture",
			Role:    logging.RoleDev,
			Console: false, // we still use fmt.Printf for user-facing output
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: logging init failed: %v\n", err)
		}
		defer log.Close()

		// Resolve absolute paths
		absAppDir, err := filepath.Abs(appDir)
		if err != nil {
			return fmt.Errorf("failed to resolve app directory: %w", err)
		}

		absOutput, err := filepath.Abs(output)
		if err != nil {
			return fmt.Errorf("failed to resolve output path: %w", err)
		}

		// Ensure output directory exists
		outputDir := filepath.Dir(absOutput)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}

		// Handle preset
		if presetName != "" {
			preset, ok := screenshot.GetPreset(presetName)
			if !ok {
				return fmt.Errorf("unknown preset: %s (use --list-presets to see available)", presetName)
			}
			fmt.Printf("Using preset: %s (%dx%d)\n", preset.Name, preset.Width, preset.Height)
			width = preset.Width
			height = preset.Height
		}

		// If --url is set, write a temporary app.json so the app auto-loads the URL
		var tmpAppJSON string
		if urlOverride != "" {
			appJSON := filepath.Join(absAppDir, "app.json")
			// Read existing app.json to preserve other fields
			cfg := map[string]interface{}{
				"url":    urlOverride,
				"name":   "Screenshot",
				"width":  1200,
				"height": 800,
			}
			if data, err := os.ReadFile(appJSON); err == nil {
				json.Unmarshal(data, &cfg)
				cfg["url"] = urlOverride // Override URL
			}
			data, _ := json.MarshalIndent(cfg, "", "    ")

			// If app.json exists, back it up; if not, mark for removal
			if _, err := os.Stat(appJSON); err == nil {
				tmpAppJSON = appJSON + ".screenshot-backup"
				os.Rename(appJSON, tmpAppJSON)
			} else {
				tmpAppJSON = "remove" // marker to remove after
			}
			os.WriteFile(appJSON, data, 0644)
			fmt.Printf("URL override: %s\n", urlOverride)
			log.Event("url_override", "url", urlOverride)
		}

		// Build the app first to get a direct binary
		fmt.Printf("Building app in %s...\n", absAppDir)
		log.Event("build_start", "app_dir", absAppDir)
		binaryPath := filepath.Join(absAppDir, "app-temp")
		cmdBuild := exec.Command("go", "build", "-o", binaryPath, ".")
		cmdBuild.Dir = absAppDir
		cmdBuild.Env = append(os.Environ(), "GOWORK=off") // Avoid workspace interference
		cmdBuild.Stdout = os.Stdout
		cmdBuild.Stderr = os.Stderr

		if err := cmdBuild.Run(); err != nil {
			return fmt.Errorf("failed to build app: %w", err)
		}

		// Launch the binary directly
		fmt.Printf("Launching %s...\n", binaryPath)
		cmdRun := exec.Command(binaryPath)
		cmdRun.Dir = absAppDir
		cmdRun.Stdout = os.Stdout
		cmdRun.Stderr = os.Stderr

		if err := cmdRun.Start(); err != nil {
			os.Remove(binaryPath)
			return fmt.Errorf("failed to launch app: %w", err)
		}

		pid := cmdRun.Process.Pid
		fmt.Printf("✓ Launched app (PID %d)\n", pid)
		log.Event("app_launched", "pid", pid, "binary", binaryPath)

		// Ensure we kill the app and clean up when done
		defer func() {
			if cmdRun.Process != nil {
				cmdRun.Process.Kill()
				fmt.Printf("✓ Stopped app\n")
			}
			os.Remove(binaryPath)
			// Restore or remove temporary app.json
			if tmpAppJSON != "" {
				appJSON := filepath.Join(absAppDir, "app.json")
				if tmpAppJSON == "remove" {
					os.Remove(appJSON)
				} else {
					os.Remove(appJSON)
					os.Rename(tmpAppJSON, appJSON)
				}
			}
		}()

		// Wait for window to appear - try multiple detection methods
		fmt.Printf("Waiting for app to initialize...\n")
		timeout := time.Duration(waitTime) * time.Millisecond

		// Method 1: Try PID-based detection with robotgo (works for some apps)
		err = screenshot.WaitForWindow(pid, timeout)
		windowDetected := err == nil
		useCoreGraphics := false

		// Method 2: Try CoreGraphics-based detection on macOS (works for Gio apps!)
		if !windowDetected {
			fmt.Printf("⚠ PID-based window detection failed, trying CoreGraphics...\n")
			err = screenshot.WaitForCGWindow(pid, timeout)
			if err == nil {
				windowDetected = true
				useCoreGraphics = true
				fmt.Printf("✓ Found window via CoreGraphics\n")
				log.Event("window_detected", "method", "coregraphics", "pid", pid)
			}
		}

		// Method 3: Try title-based detection if CoreGraphics also failed
		// Extract app name from directory for title search
		appName := filepath.Base(absAppDir)
		var windowID int
		if !windowDetected {
			fmt.Printf("⚠ CoreGraphics detection failed, trying title search...\n")
			// Try variations of the app name for title matching
			titleVariations := generateTitleVariations(appName)
			for _, titleGuess := range titleVariations {
				windowID, err = screenshot.WaitForWindowByTitle(titleGuess, 2*time.Second)
				if err == nil {
					windowDetected = true
					fmt.Printf("✓ Found window by title search '%s' (ID: %d)\n", titleGuess, windowID)
					break
				}
			}
		}
		_ = windowID // May be used later for title-based capture

		// If window detection fails, fall back to full screen capture
		if !windowDetected {
			fmt.Printf("⚠ Window detection failed (robotgo may not support Gio windows on this platform)\n")
			fmt.Printf("⚠ Falling back to full screen capture\n")
			fmt.Printf("⚠ Please manually position the app window before screenshot\n")

			// Give user time to position window
			fmt.Printf("Waiting 3 seconds for window positioning...\n")
			time.Sleep(3 * time.Second)

			// Capture full screen instead
			fmt.Printf("Capturing full screen...\n")
			if err := screenshot.CaptureDesktop(absOutput, quality); err != nil {
				return fmt.Errorf("failed to capture screenshot: %w", err)
			}
		} else {
			fmt.Printf("✓ Window appeared\n")

			// Give app time to render and webviews to load content.
			// The webview auto-navigates on frame 3, then the page
			// needs time to fully load (DNS, fetch, render).
			time.Sleep(10 * time.Second)

			// Note: robotgo doesn't have window positioning functions
			// So we just capture the window as-is
			if width > 0 && height > 0 {
				fmt.Printf("Note: Window sizing requested (%dx%d) but robotgo doesn't support resizing\n", width, height)
				fmt.Printf("      Will capture window at its current size\n")
			}

			// Capture screenshot using the method that worked
			fmt.Printf("Capturing window screenshot...\n")
			var captureErr error
			if useCoreGraphics {
				captureErr = screenshot.CaptureWindowByCGBounds(pid, absOutput, quality)
			} else {
				captureErr = screenshot.CaptureWindowByPID(pid, absOutput, quality)
			}
			if captureErr != nil {
				return fmt.Errorf("failed to capture screenshot: %w", captureErr)
			}
		}

		fmt.Printf("✓ Screenshot saved: %s\n", absOutput)
		log.Event("screenshot_saved", "output", absOutput, "method", func() string {
			if useCoreGraphics {
				return "coregraphics_region"
			}
			return "robotgo"
		}())
		return nil
	},
}

func init() {
	runAndCaptureCmd.Flags().String("preset", "", "App Store preset size (e.g., macos-retina, iphone-6.9)")
	runAndCaptureCmd.Flags().Int("width", 0, "Window width (note: robotgo doesn't support resizing)")
	runAndCaptureCmd.Flags().Int("height", 0, "Window height (note: robotgo doesn't support resizing)")
	runAndCaptureCmd.Flags().IntP("quality", "q", 90, "JPEG quality (1-100)")
	runAndCaptureCmd.Flags().IntP("wait", "w", 5000, "Max wait time for window in milliseconds")
	runAndCaptureCmd.Flags().String("url", "", "URL to load in webview apps (writes temporary app.json)")

	rootCmd.AddCommand(runAndCaptureCmd)
}
