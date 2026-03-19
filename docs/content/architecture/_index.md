---
title: "Architecture"
date: 2025-12-21
draft: false
weight: 3
---

# Architecture

How utm-dev works and the technical decisions behind it.

## How utm-dev Builds Apps

utm-dev wraps [gogio](https://pkg.go.dev/gioui.org/cmd/gogio) (the official Gio build tool) with additional capabilities:

1. **Build** -- Compiles Go source to platform binaries via gogio, with idempotent caching
2. **Bundle** -- Creates signed app bundles with Info.plist, entitlements, code signing (pure Go, no bash)
3. **Package** -- Archives bundles into tar.gz/zip for distribution

The build cache (`pkg/buildcache/`) hashes source files (SHA256) and skips rebuilds when nothing changed.

## Two Separate Systems

utm-dev has two completely separate concerns:

### Self System (`pkg/self/`)
Manages utm-dev itself -- building, installing, upgrading the tool binary. Self-contained, no imports from other `pkg/` directories. All commands output JSON for remote automation.

### App Build System (everything else)
Manages the Gio applications that users create -- building, bundling, packaging, SDK management, icon generation.

These never cross-reference each other.

## Technical Guides

- **[Webview Analysis](/architecture/webview/)** -- Cross-platform webview deep dive: which engines are used, API comparison, architecture options
- **[Roadmap](/architecture/roadmap/)** -- Planned improvements and implementation phases

## Key Packages

| Package | Purpose |
|---------|---------|
| `pkg/buildcache` | SHA256-based build caching for idempotent builds |
| `pkg/config` | SDK definitions (JSON files), directory management |
| `pkg/icons` | Icon generation from source PNG to platform formats |
| `pkg/installer` | SDK download, extraction, checksum verification |
| `pkg/packaging` | macOS bundle creation, code signing, archive creation |
| `pkg/project` | Project structure detection and path management |
| `pkg/self` | utm-dev self-management (build, install, upgrade) |
| `pkg/self/output` | JSON output types for self commands |
| `pkg/utm` | UTM virtual machine control for Windows testing |

## Dependencies

### Gio Ecosystem
- `gioui.org` -- Core Gio UI framework (v0.9.1 compatible commit)
- `github.com/gioui-plugins/gio-plugins` -- Native plugins: webviewer, hyperlink, auth, explorer (v0.9.1)

### CLI
- `github.com/spf13/cobra` -- Command framework
- `github.com/schollz/progressbar/v3` -- Progress display

### Build Tools
- `github.com/JackMordaunt/icns` -- macOS icon format generation
- `mvdan.cc/garble` -- Optional code obfuscation for release builds
