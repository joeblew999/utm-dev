# tauri-basic

Minimal Tauri v2 app with embedded frontend. Builds for iOS Simulator without a signing cert, macOS desktop, and Windows via UTM VM.

## Running

From the project root (with utm-dev built):

```bash
# iOS simulator (no signing cert needed)
.bin/utm-dev tauri build ios examples/tauri-basic
.bin/utm-dev ios install examples/tauri-basic/src-tauri/gen/apple/build/arm64-sim/tauri-basic.app
.bin/utm-dev ios launch dev.example.tauri-basic

# macOS desktop
.bin/utm-dev tauri build macos examples/tauri-basic

# Windows via UTM VM
.bin/utm-dev tauri build windows examples/tauri-basic
```

From this directory (with utm-dev installed via plat-trunk):

```bash
utm-dev tauri build ios .
utm-dev tauri build macos .
utm-dev tauri build windows .
```

## How iOS sim build works

When no `APPLE_DEVELOPMENT_TEAM` env var is set and no `developmentTeam` is in `tauri.conf.json`, `utm-dev tauri build ios` automatically adds `--target aarch64-sim` to build for the iOS Simulator instead of a physical device.

## Project structure

```
ui/index.html              Frontend (embedded into binary at build time)
src-tauri/tauri.conf.json   Tauri config (frontendDist, windows, bundle)
src-tauri/src/lib.rs        Rust backend with greet command
src-tauri/src/main.rs       Desktop entry point
```

## Screenshot

![iOS Simulator](.screenshots/ios-simulator.png)
