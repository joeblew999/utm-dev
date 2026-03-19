# utm-dev Documentation

Welcome to the utm-dev documentation.

## What is utm-dev?

**utm-dev** is a build tool for creating **cross-platform hybrid applications** using Go and Gio UI.

Build pure Go apps that run everywhere:
- 🖥️ **Desktop**: macOS, Windows, Linux
- 📱 **Mobile**: iOS, Android
- 🌐 **Web**: Browser (WASM)
- 🔀 **Hybrid**: Native Gio UI + native webviews

## Quick Links

- **[Quick Start Guide](quickstart.md)** - Get up and running in 5 minutes
- **[Platform Support](platforms.md)** - Platform-specific features and requirements
- **[Packaging Guide](PACKAGING.md)** - Create distribution-ready packages
- **[Webview Analysis](WEBVIEW-ANALYSIS.md)** - Deep dive into hybrid app architecture
- **[AI Collaboration](agents/)** - Guides for AI assistants working on this project

## Core Capabilities

### Build for All Platforms

```bash
# macOS app
go run . build macos examples/hybrid-dashboard

# iOS app
go run . build ios examples/hybrid-dashboard

# Android APK
go run . build android examples/hybrid-dashboard

# Windows app
go run . build windows examples/hybrid-dashboard
```

### SDK Management

Automated installation and management of platform SDKs:

```bash
# Install Android SDK and NDK
go run . install android-sdk
go run . install android-ndk

# List available SDKs
go run . list
```

### Asset Generation

Automatic icon generation for all platforms:

```bash
# Generate icons from icon-source.png
go run . icons examples/hybrid-dashboard
```

### Screenshot Capture

Built-in screenshot capabilities for documentation:

```bash
# Capture app screenshots
task screenshot-hybrid

# Generate App Store screenshots
task screenshot-appstore-all
```

## The Hybrid App Vision

Build apps that combine **native Gio UI** (controls, navigation) with **native webviews** (rich content):

```
┌─────────────────────────────────────┐
│     Your App (Pure Go)              │
│                                     │
│  ┌─────────────────────────────┐   │
│  │  Gio UI (Native Controls)   │   │
│  │  - Tabs, buttons, layout    │   │
│  └─────────────────────────────┘   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │  Native WebView             │   │
│  │  - HTML/CSS/JavaScript      │   │
│  │  - Go ↔ JS bridge           │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

**One codebase → Runs everywhere**

## Documentation Structure

### Getting Started
- [quickstart.md](quickstart.md) - Installation and first app
- [platforms.md](platforms.md) - Platform-specific requirements

### Core Features
- [PACKAGING.md](PACKAGING.md) - Distribution packaging
- [cicd.md](cicd.md) - CI/CD integration

### Architecture
- [WEBVIEW-ANALYSIS.md](WEBVIEW-ANALYSIS.md) - Hybrid app deep dive
- [IMPROVEMENTS.md](IMPROVEMENTS.md) - Architectural overview and roadmap

### AI Collaboration
- [agents/README.md](agents/README.md) - AI assistant collaboration guide
- [agents/gio-plugins.md](agents/gio-plugins.md) - Gio plugins reference
- [agents/robotgo.md](agents/robotgo.md) - robotgo screenshot reference

## Key Features

- ✅ **Pure Go Development** - One language for all platforms
- ✅ **Hybrid Architecture** - Native UI + webview content
- ✅ **SDK Management** - Automated install and caching
- ✅ **Asset Generation** - Icons for all platforms
- ✅ **Idempotent Builds** - Safe to run multiple times
- ✅ **Screenshot Capture** - Built-in App Store screenshot generation

## Examples

The `examples/` directory contains working demonstrations:

- **hybrid-dashboard** - Gio UI + webview hybrid app (recommended starting point)
- **gio-basic** - Simple Gio UI demo
- **gio-plugin-hyperlink** - Hyperlink plugin integration
- **gio-plugin-webviewer** - Multi-tab webview browser

## Getting Help

- Check [TODO.md](../TODO.md) for known issues and roadmap
- Read [CLAUDE.md](../CLAUDE.md) for development guidelines
- File issues at https://github.com/joeblew999/utm-dev/issues

## Philosophy

**KISS (Keep It Simple, Stupid)** - utm-dev aims to provide just enough functionality to build and package cross-platform hybrid apps without unnecessary complexity.

**Pure Go** - Everything is written in Go. No polyglot toolchains required.

**Developer-Focused** - Clean CLI interface, clear error messages, helpful documentation.

---

**Ready to build hybrid apps in pure Go?** Start with the [Quick Start Guide](quickstart.md).
