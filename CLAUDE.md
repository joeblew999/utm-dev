# CLAUDE.md — utm-dev

CLI for cross-platform build tooling: Tauri desktop apps (via UTM VMs on Apple Silicon) + Gio mobile apps (iOS/Android on host Mac). Manages SDK installation, device/simulator control, and packaging so devs don't pollute their OS.
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
utm-dev tauri setup                              # install everything (idempotent)
utm-dev tauri build macos|android <dir>
utm-dev tauri build ios <dir>                    # sim fallback if no signing cert
utm-dev tauri build windows <dir>                # via UTM VM from Mac

# Gio (mobile)
utm-dev gio build android|ios|macos <dir>
utm-dev gio run android|ios-simulator <dir>
utm-dev gio bundle macos <dir>

# UTM VMs (WinRM for Windows, QEMU GA fallback)
utm-dev utm up windows-11-arm                    # idempotent: install + start + wait
utm-dev utm exec "Windows 11 ARM" whoami         # arbitrary commands
utm-dev utm down "Windows 11 ARM"

# Utilities
utm-dev android devices|emulator
utm-dev ios devices|boot
utm-dev config

# Self
utm-dev self version | jq .
```

## UTM VM details

- Gallery key: `windows-11-arm` → VM name: `Windows 11 ARM`
- Credentials: vagrant / vagrant
- Ports: RDP localhost:3389, WinRM localhost:5985
- `utm exec` runs arbitrary commands (no auto-prefix)
- `utm build` runs `utm-dev build` inside the VM (requires utm-dev bootstrapped)

## CI

`jdx/mise-action@v2` everywhere. No setup-task. No Taskfile.

## Gio versions (never @latest — causes panics)

```bash
go get gioui.org@7bcb315ee174
go get github.com/gioui-plugins/gio-plugins@v0.9.2
```
