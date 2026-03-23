# utm-dev

CLI for cross-platform build tooling: Tauri desktop apps (tested via UTM VMs on Apple Silicon) + Gio mobile apps (iOS/Android on host Mac). Manages SDK installation, device/simulator control, and packaging.

![Status](https://img.shields.io/badge/status-alpha-orange)
![Go Version](https://img.shields.io/badge/go-1.25%2B-blue)

## Install

```bash
# From source
go build -o utm-dev .

# Or download from releases
# https://github.com/joeblew999/utm-dev/releases/latest
```

## Tauri apps (desktop + mobile)

Build and run Tauri v2 apps across all platforms.

```bash
# Dev mode with hot reload
utm-dev tauri dev examples/tauri-basic

# Desktop builds
utm-dev tauri build macos examples/tauri-basic      # .app + .dmg on host
utm-dev tauri build windows examples/tauri-basic     # .msi + .exe via UTM VM
utm-dev tauri build linux examples/tauri-basic       # .deb via UTM VM

# Mobile builds (on host Mac)
utm-dev tauri init ios examples/tauri-basic           # One-time setup
utm-dev tauri build ios examples/tauri-basic
utm-dev tauri run ios examples/tauri-basic            # Run on simulator
utm-dev tauri init android examples/tauri-basic
utm-dev tauri build android examples/tauri-basic
utm-dev tauri run android examples/tauri-basic        # Run on emulator

# Icons
utm-dev tauri icons examples/tauri-basic
```

## Gio apps (desktop, mobile, web)

Build Gio/webview hybrid apps for all platforms.

```bash
utm-dev gio build android examples/hybrid-dashboard
utm-dev gio build ios examples/hybrid-dashboard
utm-dev gio run android examples/hybrid-dashboard    # Build + install + launch
utm-dev gio run ios-simulator examples/hybrid-dashboard
utm-dev gio bundle macos examples/hybrid-dashboard   # Signed .app bundle
utm-dev gio package android examples/hybrid-dashboard
```

## UTM virtual machines

Automate Windows 11 ARM VMs on Apple Silicon for desktop cross-platform testing.

```bash
utm-dev utm install windows-11      # Download + import pre-built VM
utm-dev utm start "Windows 11"      # Start VM
utm-dev utm exec "Windows 11" -- whoami
utm-dev utm stop "Windows 11"
```

## SDK management

Install Android NDK, platform-tools, etc. without polluting your OS.

```bash
utm-dev install ndk-bundle          # Android NDK
utm-dev install android-sdk         # Android SDK
utm-dev list                        # Show available SDKs
```

## Self-management

```bash
utm-dev self version | jq .         # Show version (JSON)
utm-dev self build                  # Cross-compile utm-dev
utm-dev self upgrade                # Update to latest release
```

## Examples

| Example | What it shows |
|---------|--------------|
| `examples/tauri-basic` | Minimal Tauri v2 desktop + mobile app |
| `examples/hybrid-dashboard` | Gio + embedded HTTP server + HTMX + webview |
| `examples/gio-plugin-webviewer` | Multi-tab browser with webview API |
| `examples/gio-basic` | Simple pure Gio app |

## Platform support

| Platform | Tauri | Gio | How |
|----------|-------|-----|-----|
| macOS | Tested | Tested | Host Mac |
| iOS | Experimental | Tested | Host Mac (Xcode) |
| Android | Experimental | Tested | Host Mac (NDK) |
| Windows | Tested | Cross-compile | UTM VM |
| Linux | Planned | Cross-compile | UTM VM |

## Development

```bash
mise run build    # go build -o .bin/utm-dev .
mise run test     # go test ./...
```

## Gio version pinning

Never use `@latest` — causes panics:

```bash
go get gioui.org@7bcb315ee174
go get github.com/gioui-plugins/gio-plugins@v0.9.1
```

## Credits

- [Tauri](https://tauri.app) — Rust-based cross-platform app framework
- [Gio UI](https://gioui.org) — pure Go immediate-mode UI
- [gio-plugins](https://github.com/gioui-plugins/gio-plugins) — native webview, file picker, etc.
- [UTM](https://mac.getutm.app) — virtual machines for Apple Silicon

## License

[Check LICENSE file]
