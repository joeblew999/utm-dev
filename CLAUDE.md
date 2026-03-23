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

## Two systems — don't mix

- `pkg/self/` + `cmd/self.go` — builds/upgrades utm-dev itself (JSON output only)
- Everything else — builds user Gio/Tauri apps and manages UTM VMs

## Key commands

```bash
# Tauri (desktop + mobile)
utm-dev tauri dev|build|run|init <platform> <dir>
utm-dev tauri build windows <dir>   # builds inside UTM VM

# Gio (mobile)
utm-dev build macos|ios|android|windows <dir>

# UTM VMs
utm-dev utm install|start|exec|stop "Windows 11"

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
