package icons

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/JackMordaunt/icns"
	"github.com/joeblew999/utm-dev/pkg/cli"
	"github.com/nfnt/resize"
	ico "github.com/vldrus/golang/image/ico"
)

// generateAndroidIcons creates Android drawable icons
func generateAndroidIcons(inputPath, outputDir string) error {
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	densities := map[string]int{
		"drawable-mdpi":    48,
		"drawable-hdpi":    72,
		"drawable-xhdpi":   96,
		"drawable-xxhdpi":  144,
		"drawable-xxxhdpi": 192,
	}

	for dir, size := range densities {
		dirPath := filepath.Join(outputDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}

		resizedImg := resize.Resize(uint(size), uint(size), img, resize.Lanczos3)
		filePath := filepath.Join(dirPath, "icon.png")
		outFile, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", filePath, err)
		}
		defer outFile.Close()

		if err := png.Encode(outFile, resizedImg); err != nil {
			return fmt.Errorf("failed to encode %s: %w", filePath, err)
		}
		cli.Debug("Generated %s", filePath)
	}

	return nil
}

// generateIOSIcons creates iOS app icon set
func generateIOSIcons(inputPath, outputDir string) error {
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	iconSetDir := filepath.Join(outputDir, "AppIcon.appiconset")
	if err := os.MkdirAll(iconSetDir, 0755); err != nil {
		return fmt.Errorf("failed to create iconset directory: %w", err)
	}

	sizes := map[string]int{
		"icon-20@1x.png":   20,
		"icon-20@2x.png":   40,
		"icon-20@3x.png":   60,
		"icon-29@1x.png":   29,
		"icon-29@2x.png":   58,
		"icon-29@3x.png":   87,
		"icon-40@1x.png":   40,
		"icon-40@2x.png":   80,
		"icon-40@3x.png":   120,
		"icon-60@2x.png":   120,
		"icon-60@3x.png":   180,
		"icon-76@1x.png":   76,
		"icon-76@2x.png":   152,
		"icon-83.5@2x.png": 167,
		"icon-1024.png":    1024,
	}

	for name, size := range sizes {
		resizedImg := resize.Resize(uint(size), uint(size), img, resize.Lanczos3)
		filePath := filepath.Join(iconSetDir, name)
		outFile, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", name, err)
		}
		defer outFile.Close()

		if err := png.Encode(outFile, resizedImg); err != nil {
			return fmt.Errorf("failed to encode %s: %w", name, err)
		}
		cli.Debug("Generated %s", filePath)
	}

	// Create Contents.json
	contentsJSON := `{
  "images": [
    {"idiom": "iphone", "scale": "2x", "size": "20x20", "filename": "icon-20@2x.png"},
    {"idiom": "iphone", "scale": "3x", "size": "20x20", "filename": "icon-20@3x.png"},
    {"idiom": "iphone", "scale": "1x", "size": "29x29", "filename": "icon-29@1x.png"},
    {"idiom": "iphone", "scale": "2x", "size": "29x29", "filename": "icon-29@2x.png"},
    {"idiom": "iphone", "scale": "3x", "size": "29x29", "filename": "icon-29@3x.png"},
    {"idiom": "iphone", "scale": "2x", "size": "40x40", "filename": "icon-40@2x.png"},
    {"idiom": "iphone", "scale": "3x", "size": "40x40", "filename": "icon-40@3x.png"},
    {"idiom": "iphone", "scale": "2x", "size": "60x60", "filename": "icon-60@2x.png"},
    {"idiom": "iphone", "scale": "3x", "size": "60x60", "filename": "icon-60@3x.png"},
    {"idiom": "ipad", "scale": "1x", "size": "20x20", "filename": "icon-20@1x.png"},
    {"idiom": "ipad", "scale": "2x", "size": "20x20", "filename": "icon-20@2x.png"},
    {"idiom": "ipad", "scale": "1x", "size": "29x29", "filename": "icon-29@1x.png"},
    {"idiom": "ipad", "scale": "2x", "size": "29x29", "filename": "icon-29@2x.png"},
    {"idiom": "ipad", "scale": "1x", "size": "40x40", "filename": "icon-40@1x.png"},
    {"idiom": "ipad", "scale": "2x", "size": "40x40", "filename": "icon-40@2x.png"},
    {"idiom": "ipad", "scale": "1x", "size": "76x76", "filename": "icon-76@1x.png"},
    {"idiom": "ipad", "scale": "2x", "size": "76x76", "filename": "icon-76@2x.png"},
    {"idiom": "ipad", "scale": "2x", "size": "83.5x83.5", "filename": "icon-83.5@2x.png"},
    {"idiom": "ios-marketing", "scale": "1x", "size": "1024x1024", "filename": "icon-1024.png"}
  ],
  "info": {"version": 1, "author": "utm-dev"}
}`

	contentsPath := filepath.Join(iconSetDir, "Contents.json")
	if err := os.WriteFile(contentsPath, []byte(contentsJSON), 0644); err != nil {
		return fmt.Errorf("failed to write Contents.json: %w", err)
	}

	cli.Success("Generated iOS icons in %s", iconSetDir)
	return nil
}

// generateWindowsIcons creates Windows MSIX icons
func generateWindowsIcons(inputPath, outputDir string) error {
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	assetsDir := filepath.Join(outputDir, "assets")
	if err := os.MkdirAll(assetsDir, 0755); err != nil {
		return fmt.Errorf("failed to create assets directory: %w", err)
	}

	sizes := map[string]int{
		"Square150x150Logo.png": 150,
		"Square44x44Logo.png":   44,
	}

	for name, size := range sizes {
		resizedImg := resize.Resize(uint(size), uint(size), img, resize.Lanczos3)
		filePath := filepath.Join(assetsDir, name)
		outFile, err := os.Create(filePath)
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", name, err)
		}
		defer outFile.Close()

		if err := png.Encode(outFile, resizedImg); err != nil {
			return fmt.Errorf("failed to encode %s: %w", name, err)
		}
		cli.Debug("Generated %s", filePath)
	}

	return nil
}

// generateICNS creates macOS .icns file
func generateICNS(inputPath, outputPath string) error {
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	icnsPath := filepath.Join(outputPath, "icon.icns")
	outFile, err := os.Create(icnsPath)
	if err != nil {
		return fmt.Errorf("failed to create .icns file: %w", err)
	}
	defer outFile.Close()

	if err := icns.Encode(outFile, img); err != nil {
		return fmt.Errorf("failed to encode .icns file: %w", err)
	}

	cli.Success("Generated %s", icnsPath)
	return nil
}

// generateICO creates Windows .ico file
func generateICO(inputPath, outputPath string) error {
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open input file: %w", err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputPath, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	icoPath := filepath.Join(outputPath, "icon.ico")
	outFile, err := os.Create(icoPath)
	if err != nil {
		return fmt.Errorf("failed to create .ico file: %w", err)
	}
	defer outFile.Close()

	if err := ico.Encode(outFile, img); err != nil {
		return fmt.Errorf("failed to encode .ico file: %w", err)
	}

	cli.Success("Generated %s", icoPath)
	return nil
}
