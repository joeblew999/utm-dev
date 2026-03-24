# [utm-dev](https://github.com/joeblew999/utm-dev)

Help devs not go crazy.

Build [Tauri](https://tauri.app/) apps for **all 5 platforms** — macOS, iOS, Android, Windows, Linux — without thinking about where code runs.

**You just say what you want. utm-dev figures out the rest.**

```bash
mise run mac:dev          # runs natively on your Mac
mise run ios:sim          # runs natively on your Mac
mise run android:sim      # runs natively on your Mac
mise run windows:build    # runs in a Windows VM (auto-created)
mise run linux:build      # runs in a Linux VM (auto-created)
mise run linux:dev        # opens a Linux desktop VM for dev/testing
```

You never manage VMs, install cross-compilers, or SSH into anything. If the target platform needs a VM, it starts one. If it doesn't, it runs locally. Same commands whether you're on macOS or inside the Linux VM.

## Quick start

```bash
# 1. Install mise (if you don't have it)
curl https://mise.run | sh
mise activate zsh >> ~/.zshrc   # restart terminal after

# 2. Add utm-dev to your Tauri project
# Add to your mise.toml:
#   [task_config]
#   includes = ["git::https://github.com/joeblew999/utm-dev.git//.mise/tasks?ref=main"]

# 3. Set up everything
mise run init      # add tools + env to your mise.toml
mise install       # install tools (Rust, Bun, cargo-tauri, etc.)
mise run setup     # install platform deps (macOS: Xcode/Android, Linux: system libs)
mise run doctor    # check what's installed
```

That's it. You're ready to build.

## Build for every platform

```bash
# macOS (runs natively)
mise run mac:dev              # desktop dev mode (hot reload)
mise run mac:build            # .app + .dmg

# iOS (runs natively)
mise run ios:sim              # simulator (no signing required)
mise run ios:xcode            # open in Xcode
mise run ios:build            # release IPA (requires signing)

# Android (runs natively)
mise run android:sim          # emulator
mise run android:studio       # open in Android Studio
mise run android:build        # .apk + .aab

# Windows (auto-starts VM on first run)
mise run windows:build        # .msi + .exe

# Linux (auto-starts VM on first run)
mise run linux:dev            # open Linux desktop VM (Debian 12 + GNOME)
mise run linux:build          # .deb + .AppImage

# Everything at once
mise run all:build
```

Build artifacts land in `.build/windows/` and `.build/linux/`.

> **Works from Linux too.** Inside the Linux VM (or on any native Linux box), `mise run setup` installs system deps, and `cargo tauri dev` / `cargo tauri build` just work. Same `mise.toml`, same tasks.

## Utilities

```bash
mise run icon                     # generate all platform icons from app-icon.png
mise run screenshot               # take screenshots via WebDriver
mise run doctor                   # check what's installed/missing
mise run clean:project            # wipe this project's build artifacts
mise run clean:disk               # free system-wide disk space
mise run clean:disk -- --dry-run  # preview what would be cleaned
mise run clean:disk -- --deep     # also: Homebrew, Xcode archives, Docker
mise run mcp                      # set up MCP servers for AI-assisted dev
```

## Prerequisites

**macOS (full platform support):**
- Apple Silicon Mac
- [mise](https://mise.jdx.dev) — `curl https://mise.run | sh`
- [Homebrew](https://brew.sh), Xcode (from App Store)
- **8 GB+ RAM** recommended (VMs need headroom)
- **~6 GB disk** for the box cache (`~/.cache/utm-dev/`)

**Linux (native desktop/server builds):**
- [mise](https://mise.jdx.dev) — `curl https://mise.run | sh`
- `mise run setup` installs everything else (apt system libs)

Add these to your `.gitignore`:

```
.build/
.mise/state/
.mise/logs/
.mcp.json
```

> **Everything is idempotent.** Safe to re-run any command after errors, interruptions, or just to make sure things are current.

## Example

See [`examples/tauri-basic/`](examples/tauri-basic/) for a working Tauri app with all platform tasks configured. Copy it as a starting point.

---

## How it works

### Stack

- **[mise](https://mise.jdx.dev)** — task runner, tool management, orchestration
- **[Bun](https://bun.sh)** — all task scripts are TypeScript
- **[Tauri](https://tauri.app/)** — the apps you're building

### What installs what

**mise handles everything it can** — Rust, Bun, cargo-tauri, Java, xcodegen, ruby. `mise run setup` only installs what mise _can't_: OS-level packages and SDKs.

| Tool | Installed by | macOS | Linux |
|---|---|---|---|
| Bun, cargo-tauri, xcodegen, ruby, Java | **mise** `[tools]` | yes | yes (skips macOS-only) |
| Rust | **mise** or **rustup** | yes | yes |
| WebKitGTK, GTK, system libs | **apt** | — | `mise run setup` |
| Android SDK, NDK, emulator | **sdkmanager** | `mise run setup` | — |
| CocoaPods | **gem** | `mise run setup` | — |
| Xcode | **App Store** | manual | — |
| UTM | **Homebrew cask** | auto on first VM | — |

### Five VMs

| VM | Box | Purpose | Ports |
|---|---|---|---|
| **windows-build** | windows-11 | VS Build Tools, Rust, mise — compiling | SSH:2222 RDP:3389 WinRM:5985 |
| **windows-test** | windows-11 | Clean Windows + SSH — testing installers | SSH:2322 RDP:3489 WinRM:6985 |
| **linux-dev** | debian-12 (GNOME) | Full desktop — dev experience validation | SSH:2622 |
| **linux-build** | ubuntu-24.04 | Headless — compiling | SSH:2422 |
| **linux-test** | ubuntu-24.04 | Clean Ubuntu + SSH — testing packages | SSH:2522 |

VMs are created on-demand — `windows:build`, `linux:build`, and `linux:dev` handle everything automatically.

**Credentials:** `vagrant` / `vagrant` for all VMs.

**RDP into Windows:** Connect to `127.0.0.1:3389` (windows-build) or `127.0.0.1:3489` (windows-test).

### VM commands (hidden)

Most devs never need these — the `platform:target` tasks call them automatically. Use `mise tasks --hidden` to see them.

| Command | What it does |
|---|---|
| `mise run vm:up [profile]` | Start a VM (imports + bootstraps on first run) |
| `mise run vm:down [profile]` | Stop a VM |
| `mise run vm:build [profile]` | Sync + build in VM, pull artifacts back |
| `mise run vm:exec [profile] '<cmd>'` | Run a command inside a VM |
| `mise run vm:package [profile]` | Export a VM as a Vagrant box |
| `mise run vm:delete <profile>` | Delete a VM, `utm` (all + app), or `all` |

Profiles: `windows-build`, `windows-test`, `linux-dev`, `linux-build`, `linux-test`

### Deleting VMs

```bash
mise run vm:delete windows-build  # removes VM + state, keeps box cache
mise run vm:delete utm            # all 5 VMs + uninstall UTM
mise run vm:delete all            # everything (box cache always preserved)
```

### MCP (AI-assisted development)

```bash
mise run mcp   # writes .mcp.json + auto-allows permissions
```

Configures two MCP servers:

| Server | What it does |
|---|---|
| **[context7](https://github.com/upstash/context7)** | Up-to-date docs and code examples for any library |
| **[mise](https://mise.jdx.dev)** | Query tools, tasks, env vars, config — and run tasks directly |

### Troubleshooting

| Problem | Fix |
|---|---|
| Something missing | `mise run doctor` — shows exactly what's installed and what's not |
| VM state corrupted | Delete `.mise/state/vm-{name}.env` and re-run `mise run vm:up` |
| Bootstrap is slow | Normal — Windows VS Build Tools takes 5-10 min on first run |
| Need detailed logs | Check `.mise/logs/` |

**Manual SSH:**

```bash
sshpass -p vagrant ssh -p 2222 vagrant@127.0.0.1   # windows-build
sshpass -p vagrant ssh -p 2322 vagrant@127.0.0.1   # windows-test
sshpass -p vagrant ssh -p 2622 vagrant@127.0.0.1   # linux-dev
sshpass -p vagrant ssh -p 2422 vagrant@127.0.0.1   # linux-build
sshpass -p vagrant ssh -p 2522 vagrant@127.0.0.1   # linux-test
```

## CI

Use [`jdx/mise-action@v2`](https://github.com/jdx/mise-action) in GitHub Actions.

## License

[MIT](LICENSE)
