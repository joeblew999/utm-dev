# [utm-dev](https://github.com/joeblew999/utm-dev)

Help devs not go crazy.

Build [Tauri](https://tauri.app/) apps for **all 5 platforms from a single Mac**. utm-dev handles Rust, Android SDK, NDK, CocoaPods, Xcode, Windows VMs, and Linux VMs — so you don't have to.

Your Mac does macOS + iOS + Android natively. utm-dev adds Windows 11 and Ubuntu ARM VMs via [UTM](https://mac.getutm.app/) for the rest.

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
mise run setup     # install Mac + mobile dev tools
mise run doctor    # check what's installed
```

That's it. You're ready to build.

## Build for every platform

```bash
# macOS
mise run mac:dev              # desktop dev mode (hot reload)
mise run mac:build            # .app + .dmg

# iOS
mise run ios:sim              # simulator (no signing required)
mise run ios:xcode            # open in Xcode
mise run ios:build            # release IPA (requires signing)

# Android
mise run android:sim          # emulator
mise run android:studio       # open in Android Studio
mise run android:build        # .apk + .aab

# Windows & Linux (VMs auto-start on first run)
mise run windows:build        # .msi + .exe
mise run linux:build          # .deb + .AppImage

# Everything at once
mise run all:build
```

Build artifacts land in `.build/windows/` and `.build/linux/`.

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

- macOS on Apple Silicon
- [mise](https://mise.jdx.dev) — `curl https://mise.run | sh`
- [Homebrew](https://brew.sh)
- Xcode (from App Store)
- **8 GB+ RAM** recommended (VMs need headroom)
- **~6 GB disk** for the box cache (`~/.cache/utm-dev/`)

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

| Tool | Installed by | Location |
|---|---|---|
| Bun, cargo-tauri, xcodegen, ruby, Java | **mise** | `~/.local/share/mise/installs/` |
| Rust | **rustup** | `~/.rustup/` + `~/.cargo/` |
| Android SDK, NDK, emulator | **sdkmanager** | `~/.android-sdk/` |
| CocoaPods | **gem** | system Ruby gems |
| Xcode | **App Store** | `/Applications/Xcode.app` |
| UTM | **Homebrew cask** | `/Applications/UTM.app` |

`mise run setup` installs the non-mise tools. `mise install` installs the mise-managed tools.

### Four VMs

| VM | Purpose | Ports |
|---|---|---|
| **windows-build** | VS Build Tools, Rust, mise — for compiling | SSH:2222 RDP:3389 WinRM:5985 |
| **windows-test** | Clean Windows + SSH — for testing installers | SSH:2322 RDP:3489 WinRM:6985 |
| **linux-build** | build-essential, Rust, mise, Tauri deps — for compiling | SSH:2422 |
| **linux-test** | Clean Ubuntu + SSH — for testing packages | SSH:2522 |

VMs are created on-demand — `windows:build` and `linux:build` handle everything automatically.

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

Profiles: `windows-build`, `windows-test`, `linux-build`, `linux-test`

### Deleting VMs

```bash
mise run vm:delete windows-build  # removes VM + state, keeps box cache
mise run vm:delete utm            # all 4 VMs + uninstall UTM
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
sshpass -p vagrant ssh -p 2422 vagrant@127.0.0.1   # linux-build
sshpass -p vagrant ssh -p 2522 vagrant@127.0.0.1   # linux-test
```

## CI

Use [`jdx/mise-action@v2`](https://github.com/jdx/mise-action) in GitHub Actions.

## License

[MIT](LICENSE)
