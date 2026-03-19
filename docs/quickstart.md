# Quick Start Guide

Get up and running with utm-dev in under 5 minutes.

## Prerequisites

- [Go 1.21+](https://golang.org/)
- [Task](https://taskfile.dev/) (optional, but recommended)
- macOS, Linux, or Windows

## Installation

```bash
# Clone the repository
git clone https://github.com/joeblew999/utm-dev
cd utm-dev

# Build utm-dev
go build .

# Or use task
task build:self
```

## Run Your First Example

```bash
# Build and run the hybrid dashboard example (macOS)
task run:hybrid

# Or manually
go run . build macos examples/hybrid-dashboard
open examples/hybrid-dashboard/.bin/hybrid-dashboard.app
```

## Install Platform SDKs

For Android and iOS development, you'll need platform SDKs:

```bash
# Android SDK and NDK
go run . install android-sdk
go run . install android-ndk

# iOS/macOS (requires manual Xcode installation from App Store)
# Then install command-line tools:
xcode-select --install
```

## Build for Different Platforms

```bash
# macOS app
go run . build macos examples/hybrid-dashboard

# iOS app (requires macOS)
go run . build ios examples/hybrid-dashboard

# Android APK
go run . build android examples/hybrid-dashboard

# Windows app (cross-compile may not work, use Windows machine)
go run . build windows examples/hybrid-dashboard
```

## Generate Icons

```bash
# Generate all platform icons from source image
go run . icons examples/hybrid-dashboard
```

## Common Tasks

```bash
# List all available tasks
task --list

# Build all examples for macOS
task build:examples:macos

# Run screenshot capture
task screenshot-hybrid

# Check utm-dev installation
go run . self doctor
```

## Next Steps

- Read [Webviewer Shell](WEBVIEWER-SHELL.md) for the zero-compile option (no Go required)
- Read [Platform Support](platforms.md) for platform-specific details
- See [PACKAGING.md](PACKAGING.md) for distribution packaging
- Explore [WEBVIEW-ANALYSIS.md](WEBVIEW-ANALYSIS.md) for hybrid app architecture
- Check [agents/](agents/) for AI collaboration patterns

## Troubleshooting

**Build fails with "SDK not found"**
- Run `go run . install <sdk-name>` to install required SDKs

**Icons not generating**
- Ensure `icon-source.png` exists in your project
- Use a square PNG (512x512 or larger recommended)

**macOS screenshot permission denied**
- Go to System Settings → Privacy & Security → Screen Recording
- Grant permission to Terminal or your IDE

## Getting Help

- Run `go run . --help` for command reference
- Check [TODO.md](../TODO.md) for known issues and roadmap
- File issues at https://github.com/joeblew999/utm-dev/issues
