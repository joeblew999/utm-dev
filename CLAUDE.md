# CLAUDE.md — utm-dev

CLI for cross-platform build tooling: Tauri desktop apps (tested via UTM VMs on Apple Silicon) + Gio mobile apps (iOS/Android on host Mac). Manages SDK installation, device/simulator control, and packaging so devs don't pollute their OS.
Installed by plat-trunk via `"github:joeblew999/utm-dev" = "latest"` in mise.toml.

## Task runner: mise (Taskfile deleted)

```bash
mise run build    # go build -o .bin/utm-dev .
mise run test     # go test ./...
mise run ci       # test + self build --obfuscate
mise run release  # go run . self release minor
```

## Three systems — don't mix

- `utm-dev tauri` — Tauri apps (Rust, desktop + mobile)
- `utm-dev gio` — Gio apps (Go, desktop + mobile + web)
- `utm-dev self` — builds/upgrades utm-dev itself (JSON output only)

## Key commands

```bash
# Tauri (desktop + mobile)
utm-dev tauri setup                              # install everything
utm-dev tauri build macos|windows|ios|android <dir>
utm-dev tauri verify ios <dir>                   # build + launch + screenshot

# Gio (mobile)
utm-dev gio build android|ios|macos <dir>
utm-dev gio run android|ios-simulator <dir>
utm-dev gio bundle macos <dir>

# UTM VMs
utm-dev utm install|start|exec|stop "Windows 11"

# Utilities
utm-dev android devices|emulator|screenshot
utm-dev ios devices|boot|screenshot
utm-dev config

# Self
utm-dev self version | jq .
```

## CI

`jdx/mise-action@v2` everywhere. No setup-task. No Taskfile.

## Gio versions (never @latest — causes panics)

```bash
go get gioui.org@7bcb315ee174
go get github.com/gioui-plugins/gio-plugins@v0.9.1
```
