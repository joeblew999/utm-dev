---
title: "utm-dev Documentation"
date: 2025-12-21
draft: false
---

# utm-dev

A build tool for cross-platform hybrid applications using Go and Gio UI.

Write your app in Go, deploy it to macOS, iOS, Android, and Windows from a single codebase. Combine native Gio UI controls with native webviews for hybrid apps.

## What utm-dev Does

- **Builds** Gio applications for macOS, iOS, Android, and Windows
- **Bundles** signed app packages for distribution (macOS code signing, Android APK signing)
- **Manages SDKs** (Android SDK, NDK) with automated install and caching
- **Generates icons** for all platforms from a single source image
- **Packages** apps into distribution archives (tar.gz, zip)

```bash
# Build a hybrid app for macOS
utm-dev build macos examples/hybrid-dashboard

# Build for Android
utm-dev build android examples/hybrid-dashboard

# Create a signed macOS bundle
utm-dev bundle macos examples/hybrid-dashboard

# Generate platform icons
utm-dev icons examples/hybrid-dashboard
```

## The Hybrid App Architecture

utm-dev enables apps that combine **native Gio UI** (Go-based controls, navigation, layout) with **native webviews** (WKWebView on macOS/iOS, Chromium WebView on Android, WebView2 on Windows):

```
+-------------------------------------+
|     Your App (Pure Go)              |
|                                     |
|  +-----------------------------+   |
|  |  Gio UI (Native Controls)   |   |
|  |  Tabs, buttons, layout      |   |
|  +-----------------------------+   |
|                                     |
|  +-----------------------------+   |
|  |  Native WebView             |   |
|  |  HTML/CSS/JavaScript        |   |
|  |  Platform web engine        |   |
|  +-----------------------------+   |
+-------------------------------------+

One codebase -> macOS, iOS, Android, Windows
```

This is for apps where you want native performance and native platform integration, but also want to render rich web content. Think: a Go desktop app that embeds a dashboard, a mobile app with web-based settings pages, or a native shell around your existing web app.

## Guides

### For Users

- **[Quick Start](/users/quickstart/)** -- Install utm-dev, build your first app
- **[Platform Support](/users/platforms/)** -- What works on each platform, requirements, known limitations
- **[Packaging](/users/packaging/)** -- Build, bundle, and package apps for distribution
- **[Webviewer Shell](/users/webviewer-shell/)** -- Run any website as a desktop app with zero coding

### For Developers

- **[Architecture](/architecture/)** -- How utm-dev works, webview analysis, project structure
- **[CI/CD](/dev/cicd/)** -- GitHub Actions integration for automated builds
- **[AI Collaboration](/dev/agents/)** -- Guides for AI assistants working on this project

## Examples

The `examples/` directory contains working apps:

| Example | Description | Key Feature |
|---------|-------------|-------------|
| `hybrid-dashboard` | Gio UI + webview hybrid app | Recommended starting point |
| `gio-plugin-webviewer` | Multi-tab webview browser | Full webview API demo / [Webviewer Shell](/users/webviewer-shell/) |
| `gio-basic` | Simple Gio UI | Gio without webview |
| `gio-plugin-hyperlink` | System browser links | Open URLs in default browser |

## Screenshots

| macOS | iOS Simulator |
|-------|---------------|
| ![macOS](/screenshots/hybrid-dashboard-macos.png) | ![iOS](/screenshots/appstore/ios/iphone-6.7.png) |

| macOS App Store | macOS Retina |
|-----------------|--------------|
| ![macOS Standard](/screenshots/appstore/macos/standard.png) | ![macOS Retina](/screenshots/appstore/macos/retina.png) |

## Requirements

- [Go 1.24+](https://golang.org/)
- macOS, Linux, or Windows host
- Platform SDKs for cross-platform builds (Android SDK/NDK, Xcode)

## Links

- [GitHub Repository](https://github.com/joeblew999/utm-dev)
- [Issues](https://github.com/joeblew999/utm-dev/issues)
- [Releases](https://github.com/joeblew999/utm-dev/releases)
