package utm

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/joeblew999/utm-dev/pkg/cli"
)

// MigrationResult represents the result of a migration operation
type MigrationResult struct {
	Source      string
	Destination string
	Migrated    bool
	Skipped     bool
	Error       error
}

// MigrateUTMApp migrates UTM.app from legacy local to global location
func MigrateUTMApp() (*MigrationResult, error) {
	legacy := LegacyPaths()
	paths := GetPaths()

	result := &MigrationResult{
		Source:      legacy.App,
		Destination: paths.App,
	}

	// Check if legacy exists
	legacyCtl := filepath.Join(legacy.App, "Contents/MacOS/utmctl")
	if _, err := os.Stat(legacyCtl); os.IsNotExist(err) {
		result.Skipped = true
		return result, nil // Nothing to migrate
	}

	// Check if global already exists
	globalCtl := filepath.Join(paths.App, "Contents/MacOS/utmctl")
	if _, err := os.Stat(globalCtl); err == nil {
		cli.Info("UTM.app already exists at global location %s", paths.App)
		cli.Info("Removing legacy installation...")
		os.RemoveAll(legacy.App)
		result.Skipped = true
		return result, nil
	}

	// Ensure global parent directory exists
	if err := os.MkdirAll(filepath.Dir(paths.App), 0755); err != nil {
		result.Error = err
		return result, err
	}

	cli.Info("Migrating UTM.app from %s to %s...", legacy.App, paths.App)

	// Try rename first (fastest if same filesystem)
	if err := os.Rename(legacy.App, paths.App); err != nil {
		// Cross-device, need to copy
		cli.Info("Cross-device move, copying...")
		if err := copyDirRecursive(legacy.App, paths.App); err != nil {
			result.Error = err
			return result, err
		}
		os.RemoveAll(legacy.App)
	}

	result.Migrated = true

	// Update cache
	gallery, _ := LoadGallery()
	if gallery != nil {
		AddUTMAppToCache(gallery.Meta.UTMApp.Version, gallery.Meta.UTMApp.Checksum)
	}

	return result, nil
}

// MigrateISOs migrates ISOs from legacy local to global location
func MigrateISOs() ([]MigrationResult, error) {
	legacy := LegacyPaths()
	paths := GetPaths()

	var results []MigrationResult

	// Check if legacy ISO directory exists
	if _, err := os.Stat(legacy.ISO); os.IsNotExist(err) {
		return results, nil // Nothing to migrate
	}

	// Ensure global ISO directory exists
	if err := os.MkdirAll(paths.ISO, 0755); err != nil {
		return nil, err
	}

	// List legacy ISOs
	entries, err := os.ReadDir(legacy.ISO)
	if err != nil {
		return nil, err
	}

	gallery, _ := LoadGallery()

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".iso") {
			continue
		}

		legacyPath := filepath.Join(legacy.ISO, entry.Name())
		globalPath := filepath.Join(paths.ISO, entry.Name())

		result := MigrationResult{
			Source:      legacyPath,
			Destination: globalPath,
		}

		// Check if already exists at global location
		if _, err := os.Stat(globalPath); err == nil {
			cli.Info("ISO %s already exists at global location, removing legacy...", entry.Name())
			os.Remove(legacyPath)
			result.Skipped = true
			results = append(results, result)
			continue
		}

		cli.Info("Migrating %s...", entry.Name())

		// Try rename first
		if err := os.Rename(legacyPath, globalPath); err != nil {
			// Cross-device, need to copy
			if err := copyFile(legacyPath, globalPath); err != nil {
				result.Error = err
				results = append(results, result)
				continue
			}
			os.Remove(legacyPath)
		}

		result.Migrated = true
		results = append(results, result)

		// Try to add to cache by matching filename to gallery
		if gallery != nil {
			for key, vm := range gallery.VMs {
				if vm.ISO.Filename == entry.Name() {
					AddISOToCache(key)
					break
				}
			}
		}
	}

	// Clean up empty legacy ISO directory
	if entries, err := os.ReadDir(legacy.ISO); err == nil && len(entries) == 0 {
		os.Remove(legacy.ISO)
	}

	return results, nil
}

// MigrateAll performs full migration from legacy to global paths
func MigrateAll() error {
	cli.Info("=== UTM Migration ===")
	fmt.Println()
	cli.Info("Legacy paths:  .bin/UTM.app, .data/utm/iso/")
	paths := GetPaths()
	cli.Info("Global paths:  %s, %s", paths.App, paths.ISO)
	fmt.Println()

	// Migrate UTM app
	cli.Info("Checking UTM.app...")
	appResult, err := MigrateUTMApp()
	if err != nil {
		return fmt.Errorf("failed to migrate UTM.app: %w", err)
	}
	if appResult.Migrated {
		cli.Success("Migrated to %s", appResult.Destination)
	} else if appResult.Skipped {
		cli.Info("  Skipped (already at global location or not found locally)")
	} else if appResult.Error != nil {
		cli.Error("Migration error: %v", appResult.Error)
	}

	fmt.Println()

	// Migrate ISOs
	cli.Info("Checking ISOs...")
	isoResults, err := MigrateISOs()
	if err != nil {
		return fmt.Errorf("failed to migrate ISOs: %w", err)
	}

	migrated := 0
	for _, r := range isoResults {
		if r.Migrated {
			migrated++
			cli.Success("Migrated: %s", filepath.Base(r.Source))
		} else if r.Error != nil {
			cli.Error("Error migrating %s: %v", filepath.Base(r.Source), r.Error)
		}
	}
	if migrated == 0 && len(isoResults) == 0 {
		cli.Info("  No ISOs to migrate")
	}

	fmt.Println()
	cli.Success("Migration complete!")
	fmt.Println()
	cli.Info("Verify with: utm-dev utm paths")

	return nil
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Copy permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, sourceInfo.Mode())
}

// copyDirRecursive recursively copies a directory
func copyDirRecursive(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDirRecursive(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}
