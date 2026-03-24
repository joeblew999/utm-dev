# utm-dev

Help devs not go crazy.

Building [Tauri](https://tauri.app/) apps for macOS + iOS + Android + Windows from a single Mac is a nightmare — Rust, Android SDK, NDK, CocoaPods, Xcode, and a Windows machine. utm-dev handles all of it.

**Your Mac does 3 out of 4 platforms natively.** utm-dev sets up a Windows 11 ARM VM via [UTM](https://mac.getutm.app/) for the 4th.

## Stack

- **[mise](https://mise.jdx.dev)** — task runner, tool management, orchestration
- **[Bun](https://bun.sh)** — all task scripts are TypeScript
- **[Tauri](https://tauri.app/)** — the apps you're building

## Add to your project

Add this to your `mise.toml`:

```toml
[task_config]
includes = ["git::https://github.com/joeblew999/utm-dev.git//.mise/tasks?ref=main"]
```

Then:

```bash
mise run init      # Adds tools + env to your mise.toml (one time)
mise install       # Install tools (Rust, Bun, cargo-tauri, etc.)
mise run setup     # Install EVERYTHING: SDKs, UTM, both VMs (idempotent)
```

One command. `setup` installs Mac tools, Android SDK, iOS deps, UTM, and both Windows VMs. Run it as many times as you want.

## Two VMs

| VM | Purpose | Ports |
|---|---|---|
| **build** | VS Build Tools, Rust, mise — for compiling | SSH:2222 RDP:3389 WinRM:5985 |
| **test** | Clean Windows + SSH — for testing installers | SSH:2322 RDP:3489 WinRM:6985 |

Both set up by `mise run setup`. Start them with `mise run vm:up`.

## Build for every platform

```bash
cargo tauri build                              # macOS — .app + .dmg
cargo tauri ios build --target aarch64-sim     # iOS simulator — .app
cargo tauri android build                      # Android — .apk + .aab
mise run vm:build                              # Windows — .msi + .exe
```

## All commands

| Command | What it does |
|---|---|
| `mise run init` | Add tools + env to your mise.toml |
| `mise run setup` | Install everything: Mac tools, SDKs, UTM, both VMs |
| `mise run doctor` | Check what's installed/missing |
| `mise run vm:up [build\|test]` | Start a VM (default: build) |
| `mise run vm:down [build\|test]` | Stop a VM |
| `mise run vm:build` | Sync + build in build VM, pull .msi/.exe back |
| `mise run vm:exec [build\|test] '<cmd>'` | Run a command inside a VM |
| `mise run vm:package [build\|test]` | Export a VM as reusable Vagrant box |
| `mise run vm:delete build\|test\|utm\|all` | Delete VMs/UTM (keeps cached box) |

## Prerequisites

- macOS on Apple Silicon
- [mise](https://mise.jdx.dev) — `curl https://mise.run | sh`
- [Homebrew](https://brew.sh)
- Xcode (from App Store)

## Examples

See [`examples/tauri-basic/`](examples/tauri-basic/) for a working Tauri app with per-platform build tasks.
