# utm-dev

Cross-platform build tooling that actually works.

![Status](https://img.shields.io/badge/status-alpha-orange)
![Go Version](https://img.shields.io/badge/go-1.25%2B-blue)

## Why

Building cross-platform apps is a nightmare. Every dev hits the same wall:

- **Android**: download Android Studio, install 5 SDKs, set 8 environment variables, pray Gradle finds the NDK
- **iOS**: Xcode works but only on Mac, simulator setup is manual, signing is confusing
- **Windows**: if you're on Mac, you can't build natively. Cross-compilation breaks with CGO. VMs are manual and painful
- **Linux**: same VM problem as Windows

Every project re-invents this setup. Every new team member spends a day fighting toolchains. CI configs are 200 lines of environment setup.

**utm-dev fixes this.** One command installs everything. One command builds for any platform. SDKs go in an isolated directory (not polluting your system). Windows/Linux builds run in automated UTM VMs on Apple Silicon. Everything is idempotent — run it twice, nothing breaks.

```bash
utm-dev tauri setup                    # installs Rust, Android SDK, NDK — done
utm-dev tauri build android myapp      # just works
utm-dev utm up windows-11-arm          # Windows VM, fully automated
utm-dev utm exec "Windows 11 ARM" utm-dev tauri build windows myapp
```

## Install

```bash
# Via mise (recommended — used by plat-trunk)
# In your mise.toml:
"github:joeblew999/utm-dev" = "latest"

# From source
go build -o utm-dev .
```

## Quick start

```bash
# Build a Tauri app for macOS
utm-dev tauri build macos examples/tauri-basic

# Build a Gio app for Android
utm-dev gio build android examples/hybrid-dashboard

# Build for Windows via UTM VM (from Mac)
utm-dev utm up windows-11-arm                              # install + start + wait
utm-dev utm exec "Windows 11 ARM" whoami                   # run any command
utm-dev utm build "Windows 11 ARM" windows examples/tauri-basic
utm-dev utm down "Windows 11 ARM"                          # stop VM
```

## Tauri apps (desktop + mobile)

```bash
# Setup (installs Rust, cargo-tauri, Android SDK/NDK — idempotent)
utm-dev tauri setup

# Desktop builds
utm-dev tauri build macos examples/tauri-basic        # .app + .dmg
utm-dev tauri build windows examples/tauri-basic      # via UTM VM

# Mobile builds (on host Mac)
utm-dev tauri init android examples/tauri-basic       # one-time scaffolding
utm-dev tauri build android examples/tauri-basic      # APK/AAB
utm-dev tauri init ios examples/tauri-basic
utm-dev tauri build ios examples/tauri-basic           # sim fallback if no cert
utm-dev tauri run ios examples/tauri-basic             # launch in simulator

# Dev mode
utm-dev tauri dev examples/tauri-basic

# Icons
utm-dev tauri icons examples/tauri-basic
```

## Gio apps (desktop, mobile, web)

```bash
utm-dev gio build android examples/hybrid-dashboard
utm-dev gio build ios examples/hybrid-dashboard
utm-dev gio build macos examples/hybrid-dashboard
utm-dev gio run android examples/hybrid-dashboard      # build + install + launch
utm-dev gio run ios-simulator examples/hybrid-dashboard
utm-dev gio bundle macos examples/hybrid-dashboard     # signed .app bundle
```

## UTM virtual machines

Control Windows 11 ARM VMs on Apple Silicon. Uses WinRM (pre-installed in vagrant boxes) for reliable command execution.

```bash
# Full lifecycle (idempotent)
utm-dev utm up windows-11-arm         # install UTM + download VM + start + wait for WinRM
utm-dev utm exec "Windows 11 ARM" whoami
utm-dev utm exec "Windows 11 ARM" powershell -Command "Get-Date"
utm-dev utm down "Windows 11 ARM"

# Individual commands
utm-dev utm install                   # install UTM app
utm-dev utm install windows-11-arm    # download + import pre-built VM
utm-dev utm gallery                   # list available VMs
utm-dev utm start "Windows 11 ARM"
utm-dev utm status "Windows 11 ARM"
utm-dev utm stop "Windows 11 ARM"

# File transfer
utm-dev utm push "Windows 11 ARM" ./local.txt "C:\Users\vagrant\local.txt"
utm-dev utm pull "Windows 11 ARM" "C:\Users\vagrant\out.txt" ./out.txt

# Build inside VM
utm-dev utm build "Windows 11 ARM" windows examples/tauri-basic

# Network
utm-dev utm fix-network "Windows 11 ARM"   # setup RDP + WinRM port forwards
utm-dev utm doctor                          # check UTM installation
```

**VM credentials:** vagrant / vagrant
**Ports:** RDP localhost:3389, WinRM localhost:5985

## SDK management

```bash
utm-dev install android-sdk           # Android SDK + build-tools
utm-dev install ndk                   # Android NDK 27
utm-dev list                          # show available SDKs
```

## Device management

```bash
# Android
utm-dev android devices               # list connected devices
utm-dev android emulator              # launch emulator

# iOS
utm-dev ios devices                   # list simulators + devices
utm-dev ios boot                      # boot default simulator
```

## Self-management

```bash
utm-dev self version | jq .           # version info (JSON)
utm-dev self build                    # cross-compile utm-dev
utm-dev self upgrade                  # update to latest release
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
| iOS | Sim only* | Tested | Host Mac (Xcode) |
| Android | Tested | Tested | Host Mac (NDK) |
| Windows | Tested | Cross-compile | UTM VM (WinRM) |
| Linux | Planned | Cross-compile | UTM VM |

*iOS device builds require a signing cert. Sim builds work without one. Tauri iOS sim blocked on upstream cargo-mobile2 Xcode 26.x fix.

## Development

```bash
mise run build    # go build -o .bin/utm-dev .
mise run test     # go test ./...
mise run ci       # test + self build --obfuscate
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
