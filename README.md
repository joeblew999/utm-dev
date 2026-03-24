# utm-dev

Help devs not go crazy.

Building [Tauri](https://tauri.app/) apps for macOS + iOS + Android + Windows from a single Mac is a nightmare — Rust, Android SDK, NDK, CocoaPods, Xcode, and a Windows machine. utm-dev handles all of it.

**Your Mac does 3 out of 4 platforms natively.** utm-dev sets up a Windows 11 ARM VM via [UTM](https://mac.getutm.app/) for the 4th.

## Stack

- **[mise](https://mise.jdx.dev)** — task runner, tool management, orchestration
- **[Bun](https://bun.sh)** — all task scripts are TypeScript (cross-platform)
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
mise run setup     # Install SDKs + targets (idempotent)
mise run vm:up     # Windows VM + SSH + Rust (idempotent)
```

Everything is idempotent. Run any command as many times as you want.

## Build for every platform

```bash
cargo tauri build                              # macOS — .app + .dmg
cargo tauri ios build --target aarch64-sim     # iOS simulator — .app
cargo tauri android build                      # Android — .apk + .aab
mise run vm:build                              # Windows — .msi + .exe
```

macOS, iOS, and Android build natively on your Mac. Windows builds inside the VM — code is synced automatically, artifacts are pulled back to `.build/windows/`.

## All commands

| Command | What it does |
|---|---|
| `mise run init` | Add tools + env to your mise.toml |
| `mise run setup` | Install Rust, Android SDK/NDK, CocoaPods, targets |
| `mise run doctor` | Check what's installed and what's missing |
| `mise run vm:up` | Install UTM + Windows VM + bootstrap SSH + Rust |
| `mise run vm:bootstrap` | Bootstrap SSH + dev tools in VM (called by vm:up) |
| `mise run vm:build` | Sync code to VM, build, pull .msi/.exe back |
| `mise run vm:exec '<cmd>'` | Run any command inside Windows |
| `mise run vm:sync` | Sync project files to VM |
| `mise run vm:down` | Stop the VM |
| `mise run vm:package` | Export VM as reusable Vagrant box |
| `mise run vm:delete vm` | Delete VM (keeps UTM + cached box) |
| `mise run vm:delete all` | Nuclear option (still keeps cached 6 GB download) |

## Prerequisites

- macOS on Apple Silicon
- [mise](https://mise.jdx.dev) — `curl https://mise.run | sh`
- [Homebrew](https://brew.sh)
- Xcode (from App Store)

## Examples

See [`examples/tauri-basic/`](examples/tauri-basic/) — a working Tauri app you can build for all 4 platforms.
