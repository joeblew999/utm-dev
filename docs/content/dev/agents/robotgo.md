---
title: "robotgo Reference"
date: 2025-12-21
draft: false
---

# RobotGo Integration Guide

**Source**: `.src/robotgo/` (local clone for reference)
**Repository**: https://github.com/go-vgo/robotgo
**Purpose**: Desktop automation and screenshot capabilities
**Integration**: Optional via build tags (to avoid CGO in main build)

---

## Overview

RobotGo is a Go library for desktop automation including:
- **Screenshots** - Capture screen regions, full screen, multiple displays
- **Mouse control** - Move, click, scroll
- **Keyboard control** - Type text, key combinations
- **Window management** - Find, activate, manipulate windows
- **Bitmap operations** - Find images on screen
- **Event hooks** - Listen for global keyboard/mouse events

**For utm-dev**: We use **screenshots only** (via build tags)

---

## Platform Support

| Platform | Screenshot | Mouse/Keyboard | Requirements |
|----------|-----------|----------------|--------------|
| **macOS** | ✅ | ✅ | Xcode Command Line Tools, Privacy permissions |
| **Windows** | ✅ | ✅ | MinGW-w64 (GCC) |
| **Linux** | ✅ | ✅ | X11, Xtst, libpng |

**CGO Required**: Yes (C bindings to platform APIs)
**Cross-compilation**: Difficult (CGO + platform-specific libraries)

---

## Key Files to Study

### Screenshot Implementation

```
.src/robotgo/
├── screen.go              # Main screenshot API
├── screen/
│   ├── goScreen.h         # C header for screen capture
│   ├── goScreen_c.h       # Platform-agnostic interface
│   ├── screen_c.h         # Platform-specific implementations
│   └── c_screen_*.c       # Platform code (darwin, windows, linux)
└── examples/screen/main.go  # Screenshot examples
```

### Core APIs (screenshot-related)

```go
// screen.go
func CaptureScreen(x, y, w, h int) C.MMBitmapRef
func CaptureImg(x, y, w, h int) (image.Image, error)
func SaveCapture(path string, x, y, w, h int) error
func Save(img image.Image, path string) error
func SaveJpeg(img image.Image, path string, quality int) error
func FreeBitmap(bitmap C.MMBitmapRef)
func GetScreenSize() (int, int)
func GetDisplayBounds(displayID int) (int, int, int, int)
func DisplaysNum() int
```

---

## Screenshot Usage Patterns

### Simple Screenshot

```go
package main

import "github.com/go-vgo/robotgo"

func main() {
    // Full screen
    img, _ := robotgo.CaptureImg()
    robotgo.Save(img, "fullscreen.png")

    // Region
    img2, _ := robotgo.CaptureImg(10, 10, 500, 300)
    robotgo.Save(img2, "region.png")

    // Direct save
    robotgo.SaveCapture("direct.png", 0, 0, 800, 600)
}
```

### Multi-Display Support

```go
package main

import (
    "fmt"
    "strconv"
    "github.com/go-vgo/robotgo"
)

func main() {
    num := robotgo.DisplaysNum()
    fmt.Printf("Found %d displays\n", num)

    for i := 0; i < num; i++ {
        robotgo.DisplayID = i

        // Get display bounds
        x, y, w, h := robotgo.GetDisplayBounds(i)
        fmt.Printf("Display %d: x=%d y=%d w=%d h=%d\n", i, x, y, w, h)

        // Capture entire display
        img, _ := robotgo.CaptureImg(x, y, w, h)
        robotgo.Save(img, fmt.Sprintf("display_%d.png", i))
    }
}
```

### Memory Management

```go
// When using C bitmaps directly
bit := robotgo.CaptureScreen(10, 10, 100, 100)
defer robotgo.FreeBitmap(bit) // IMPORTANT: Free C memory

img := robotgo.ToImage(bit)
robotgo.Save(img, "output.png")
```

---

## Integration in utm-dev

### Build Tag Approach

**File**: `pkg/screenshot/robotgo.go`

```go
//go:build screenshot
// +build screenshot

package screenshot

import "github.com/go-vgo/robotgo"

type RobotgoCapturer struct{}

func (c *RobotgoCapturer) CaptureDesktop(output string) error {
    img, err := robotgo.CaptureImg()
    if err != nil {
        return err
    }
    return robotgo.Save(img, output)
}

func (c *RobotgoCapturer) CaptureRegion(x, y, w, h int, output string) error {
    img, err := robotgo.CaptureImg(x, y, w, h)
    if err != nil {
        return err
    }
    return robotgo.Save(img, output)
}

func (c *RobotgoCapturer) CaptureAllDisplays(prefix string) error {
    num := robotgo.DisplaysNum()
    for i := 0; i < num; i++ {
        robotgo.DisplayID = i
        x, y, w, h := robotgo.GetDisplayBounds(i)
        img, err := robotgo.CaptureImg(x, y, w, h)
        if err != nil {
            return err
        }
        path := fmt.Sprintf("%s_display_%d.png", prefix, i)
        if err := robotgo.Save(img, path); err != nil {
            return err
        }
    }
    return nil
}
```

**File**: `pkg/screenshot/screenshot.go` (no build tag)

```go
package screenshot

import (
    "fmt"
    "os/exec"
    "runtime"
)

// Fallback to platform CLI tools when robotgo not available
type CLICapturer struct{}

func (c *CLICapturer) CaptureDesktop(output string) error {
    switch runtime.GOOS {
    case "darwin":
        return exec.Command("screencapture", "-x", "-t", "png", output).Run()
    case "linux":
        return exec.Command("scrot", output).Run()
    case "windows":
        // PowerShell screenshot command
        cmd := fmt.Sprintf(`Add-Type -AssemblyName System.Windows.Forms; [System.Windows.Forms.Screen]::PrimaryScreen.Bounds | Out-Null`)
        return exec.Command("powershell", "-Command", cmd).Run()
    default:
        return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
    }
}
```

**Usage**:

```bash
# Default build (no CGO, uses platform CLI tools)
go build .

# Build with robotgo support (requires CGO)
go build -tags screenshot .

# For users
utm-dev screenshot --output screenshot.png  # Uses CLI tools

# Or with robotgo (if built with -tags screenshot)
utm-dev screenshot --output screenshot.png  # Uses robotgo
```

---

## Build Requirements

### macOS

```bash
# Already installed with Xcode Command Line Tools
xcode-select --install

# Privacy: Grant Terminal/VSCode "Screen Recording" permission
# System Settings → Privacy & Security → Screen Recording
```

### Windows

```bash
# Install MinGW-w64
winget install MartinStorsjo.LLVM-MinGW.UCRT

# Add to PATH
C:\mingw64\bin
```

### Linux (Ubuntu/Debian)

```bash
# GCC
sudo apt install gcc libc6-dev

# X11 dependencies
sudo apt install libx11-dev xorg-dev libxtst-dev

# Optional: libpng for bitmap operations
sudo apt install libpng++-dev
```

---

## Platform-Specific Notes

### macOS Privacy

RobotGo requires "Screen Recording" permission on macOS 10.15+:

1. First run will prompt for permission
2. Or manually: System Settings → Privacy & Security → Screen Recording
3. Add Terminal, VSCode, or your IDE to allowed apps
4. May need to restart app after granting permission

### Windows MinGW

- Use MinGW-w64 (not legacy MinGW)
- LLVM-MinGW recommended for modern Windows
- Ensure `gcc` is in PATH

### Linux X11

- Wayland not supported (use X11 session)
- May need `DISPLAY=:0` environment variable
- Test with `echo $DISPLAY` to verify X11 running

---

## Common Patterns

### Capture Window by PID

```go
// Find window
fpid, _ := robotgo.FindIds("Chrome")
if len(fpid) > 0 {
    // Activate window
    robotgo.ActivePid(fpid[0])

    // Wait for window to come to front
    robotgo.MilliSleep(100)

    // Get window bounds (platform-specific, see robotgo docs)
    // Then capture that region
    img, _ := robotgo.CaptureImg(x, y, w, h)
    robotgo.Save(img, "window.png")
}
```

### Find Simulator/Emulator Window

```go
// iOS Simulator
pids, _ := robotgo.FindIds("Simulator")

// Android Emulator
pids, _ := robotgo.FindIds("qemu-system")

// Activate and screenshot
if len(pids) > 0 {
    robotgo.ActivePid(pids[0])
    robotgo.MilliSleep(200)
    // Capture screen
}
```

---

## When to Use robotgo vs CLI Tools

### Use CLI Tools (Default)

- ✅ Simple screenshots
- ✅ CI/CD environments
- ✅ No CGO acceptable
- ✅ Platform-specific OK
- ✅ Lighter binary size

### Use robotgo (Optional)

- ✅ Multi-display support needed
- ✅ Precise region capture
- ✅ Window detection/activation
- ✅ Cross-platform consistency
- ✅ Advanced features (find image on screen, pixel detection)
- ⚠️ Accept CGO dependency
- ⚠️ Accept larger binary size

---

## Debugging

### Test robotgo Installation

```bash
# Clone robotgo examples
cd .src/robotgo/examples/screen

# Run screenshot example
go run main.go

# Check output files
ls -lh *.png
```

### Common Errors

**Error**: `png.h: No such file or directory`
**Fix**: Install libpng development files
```bash
# macOS
brew install libpng

# Linux
sudo apt install libpng++-dev
```

**Error**: `undefined: C.MMBitmapRef`
**Fix**: CGO not enabled
```bash
export CGO_ENABLED=1
go build
```

**Error**: `screen capture permission denied` (macOS)
**Fix**: Grant Screen Recording permission in System Settings

---

## References

- **Source code**: `.src/robotgo/`
- **Examples**: `.src/robotgo/examples/`
- **Documentation**: https://pkg.go.dev/github.com/go-vgo/robotgo
- **Platform requirements**: `.src/robotgo/README.md`
- **Screenshot strategy**: `docs/SCREENSHOT-STRATEGY.md`

---

## Summary for AI Assistants

**When implementing screenshot command**:

1. Start with CLI tools (no CGO, fast implementation)
2. Add robotgo as **optional** via build tags
3. Document both approaches in `utm-dev screenshot --help`
4. Test on actual macOS, Windows, Linux before releasing
5. Check `.src/robotgo/examples/screen/main.go` for reference implementation
6. Remember: robotgo needs Screen Recording permission on macOS
7. For CI/CD: Use CLI tools (simpler, no CGO issues)
8. For power users: Offer `go build -tags screenshot` option

**Key API**:
```go
robotgo.CaptureImg(x, y, w, h int) (image.Image, error)
robotgo.Save(img, "output.png") error
robotgo.DisplaysNum() int
```

**Build tag pattern**:
```go
//go:build screenshot
```
