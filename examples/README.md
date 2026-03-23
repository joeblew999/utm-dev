# Examples

Five examples covering Gio mobile apps and Tauri desktop/mobile apps.

## Examples

| Example | Framework | What it shows |
|---|---|---|
| **gio-basic** | Gio | Grid rendering with `gioui.org/x/component` |
| **gio-plugin-hyperlink** | Gio | Opening URLs via `gio-plugins/hyperlink` |
| **gio-plugin-webviewer** | Gio | Full browser UI with tabs via `gio-plugins/webviewer` |
| **hybrid-dashboard** | Gio | Embedded HTTP server + WebView + deep linking |
| **tauri-basic** | Tauri v2 | Webview app — iOS sim, macOS, Windows via UTM |

## Running Gio examples

From the project root (with utm-dev built):

```bash
# macOS
.bin/utm-dev gio run macos examples/gio-basic

# Android (device or emulator must be connected)
.bin/utm-dev gio run android examples/gio-basic

# iOS simulator
.bin/utm-dev gio run ios-simulator examples/gio-basic

# Build without launching
.bin/utm-dev gio build macos examples/gio-basic
.bin/utm-dev gio build android examples/gio-basic
.bin/utm-dev gio build ios examples/gio-basic
```

From an example directory (with utm-dev installed via plat-trunk):

```bash
cd examples/gio-basic
mise run dev       # macOS
mise run android   # Android
mise run ios       # iOS simulator
```

## Running Tauri examples

```bash
# iOS simulator (no signing cert needed)
.bin/utm-dev tauri build ios examples/tauri-basic

# macOS desktop
.bin/utm-dev tauri build macos examples/tauri-basic

# Windows via UTM VM
.bin/utm-dev tauri build windows examples/tauri-basic
```

## Gio versions

Pinned to avoid panics (never use `@latest`):

```
gioui.org                        v0.9.1-0.20251215212054-7bcb315ee174
gioui.org/x                      v0.9.0
github.com/gioui-plugins/gio-plugins  v0.9.2
```

All Gio examples use `go 1.25.0`.
