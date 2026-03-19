# CLAUDE.md — utm-dev

CLI for building Gio + Tauri apps and automating UTM VMs on Apple Silicon.
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
utm-dev build macos|ios|android|windows <dir>
utm-dev utm install|create|start|exec|stop "Windows 11"
utm-dev self version | jq .
```

## CI

`jdx/mise-action@v2` everywhere. No setup-task. No Taskfile.

## Gio versions (never @latest — causes panics)

```bash
go get gioui.org@7bcb315ee174
go get github.com/gioui-plugins/gio-plugins@v0.9.1
```
