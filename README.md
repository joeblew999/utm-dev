# utm-dev

Help devs not go crazy.

Building [Tauri](https://tauri.app/) apps for macOS + iOS + Android + Windows from a single Mac is a nightmare — Rust, Android SDK, NDK, CocoaPods, Xcode, and a Windows machine. utm-dev handles all of it.

**Your Mac does 3 out of 4 platforms natively.** utm-dev sets up a Windows 11 ARM VM via [UTM](https://mac.getutm.app/) for the 4th.

## Add to your project

Add this to your `mise.toml`:

```toml
[task_config]
includes = ["git::https://github.com/joeblew999/utm-dev.git//.mise/tasks?ref=main"]
```

Then:

```bash
mise run init      # Adds tools + env to your mise.toml (one time)
mise install       # Install tools
mise run setup     # Install SDKs + targets (idempotent)
mise run vm:up     # Windows VM (idempotent)
```

That's it. `init` configures your project, `setup` installs everything, `vm:up` gives you Windows.

## What you can build

| Platform | How |
|---|---|
| macOS | `cargo tauri build` — .app + .dmg |
| iOS simulator | `cargo tauri ios build --target aarch64-sim` — .app |
| iOS device | `cargo tauri ios build` — needs Apple Developer signing |
| Android | `cargo tauri android build` — .apk + .aab |
| Windows | RDP into VM (`localhost:3389`, vagrant/vagrant), build there |

macOS, iOS, and Android build natively on your Mac after `mise run setup`.

Windows builds inside the UTM VM after `mise run vm:up`. Automated Windows builds (SSH bootstrap, code sync) are coming — for now, RDP in and build manually.

## Teardown

```bash
mise run vm:down          # Stop the VM
mise run vm:delete vm     # Delete the VM (keeps cached 6 GB download)
mise run vm:delete all    # Nuclear option (still keeps cached download)
```

## Prerequisites

- macOS on Apple Silicon
- [mise](https://mise.jdx.dev) — `curl https://mise.run | sh`
- [Homebrew](https://brew.sh)
- Xcode (from App Store)

## Examples

See [`examples/tauri-basic/`](examples/tauri-basic/) — a working Tauri app with per-platform build tasks.
