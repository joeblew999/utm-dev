# tauri-basic

Minimal Tauri app demonstrating cross-platform builds via [utm-dev](https://github.com/joeblew999/utm-dev).

## Getting started

```bash
mise run init      # Add tools + env (one time)
mise install       # Install tools
mise run setup     # Install SDKs
```

## Build & run

```bash
mise run dev          # macOS desktop dev mode
mise run build        # macOS .app/.dmg
mise run ios          # iOS simulator
mise run android      # Android emulator
mise run vm:build     # Windows .msi/.exe (requires VM — see mise run vm:up)
```
