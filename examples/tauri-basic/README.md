# tauri-basic

Minimal Tauri v2 example. Opens a webview pointing at `http://localhost:3000`
and exposes a `greet` command from Rust to the frontend.

## Usage

```sh
# Dev mode (needs a frontend running on :3000)
mise run dev

# Build desktop bundle
mise run build          # → .app/.dmg on macOS, .msi/.exe on Windows

# iOS (macOS only)
mise run ios:init       # first time only
mise run ios            # run on simulator

# Windows via UTM
mise run windows        # build + run in Windows 11 VM
```

## Prerequisites

Installed automatically by the parent repo's `mise install`:
- `cargo:tauri-cli@2` — macOS + Windows
- `xcodegen` — macOS only (iOS target)
- `ruby` + CocoaPods — macOS only (iOS target)

## Icons

Drop a square PNG at `src-tauri/icons/icon-source.png` then:

```sh
mise run icons
```
