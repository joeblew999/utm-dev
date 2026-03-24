# utm-dev

Help devs not go crazy.

Building [Tauri](https://tauri.app/) apps for macOS + iOS + Android + Windows + Linux from a single Mac is a nightmare — Rust, Android SDK, NDK, CocoaPods, Xcode, a Windows machine, and a Linux box. utm-dev handles all of it.

**Your Mac does 3 out of 5 platforms natively.** utm-dev sets up Windows 11 and Ubuntu ARM VMs via [UTM](https://mac.getutm.app/) for the rest.

## Prerequisites

- macOS on Apple Silicon
- [mise](https://mise.jdx.dev) — `curl https://mise.run | sh`
- [Homebrew](https://brew.sh)
- Xcode (from App Store)
- **8 GB+ RAM** recommended (VMs need headroom)
- **~6 GB disk** for the box cache (`~/.cache/utm-dev/`)

## Stack

- **[mise](https://mise.jdx.dev)** — task runner, tool management, orchestration
- **[Bun](https://bun.sh)** — all task scripts are TypeScript
- **[Tauri](https://tauri.app/)** — the apps you're building

## mise basics

If you're new to [mise](https://mise.jdx.dev), here's everything you need:

**Install:**

```bash
curl https://mise.run | sh       # install mise
mise activate zsh >> ~/.zshrc    # add to your shell (restart terminal after)
```

**Daily usage:**

```bash
mise install          # install all tools defined in mise.toml [tools] section
mise run <task>       # run a task (e.g., mise run setup, mise run mac:dev)
mise run <task> -- <args>  # pass arguments to a task
mise tasks            # list available tasks
mise tasks --hidden   # also show hidden tasks (vm:up, vm:down, etc.)
```

**How it works:**

- `mise.toml` in your project root defines tools (with versions) and tasks
- `mise install` downloads and manages tool versions in `~/.local/share/mise/installs/`
- Tools are activated automatically when you `cd` into the project — no global installs
- Tasks are scripts (TypeScript files in `.mise/tasks/`) that mise discovers and runs with the right tool versions on PATH

**Key concepts:**

| Concept | What it means |
|---|---|
| `[tools]` | Tools + versions to install (Bun, Rust, Java, etc.) |
| `[env]` | Environment variables set when in this project |
| `[tasks]` | Named commands you can `mise run` |
| `[task_config].includes` | Pull in tasks from other repos (how utm-dev works) |

**Useful commands:**

```bash
mise ls                # show installed tool versions
mise doctor            # check mise health
mise self-update       # update mise itself
mise trust             # trust a mise.toml (required on first use)
```

> **mise replaces:** nvm, pyenv, rbenv, asdf, make, npm scripts, and Makefiles — all in one tool.

## What installs what

mise manages version-isolated tools in `~/.local/share/mise/installs/`. Everything else uses native installers:

| Tool | Installed by | Location |
|---|---|---|
| Bun, cargo-tauri, xcodegen, ruby, Java | **mise** (`[tools]`) | `~/.local/share/mise/installs/` |
| Rust | **rustup** | `~/.rustup/` + `~/.cargo/` |
| Android SDK, NDK, emulator | **sdkmanager** | `~/.android-sdk/` |
| CocoaPods | **gem** | system Ruby gems |
| Xcode | **App Store** | `/Applications/Xcode.app` |
| UTM | **Homebrew cask** | `/Applications/UTM.app` |

`mise run setup` installs the non-mise tools. `mise install` installs the mise-managed tools from `[tools]`.

## Add to your project

Run all commands from your **project root** (where `mise.toml` lives).

Add this to your `mise.toml`:

```toml
[task_config]
includes = ["git::https://github.com/joeblew999/utm-dev.git//.mise/tasks?ref=main"]
```

> Pin to a specific commit or tag for stability: `?ref=v1.0.0` or `?ref=abc1234`.

Then:

```bash
mise run init      # Adds tools + env to your mise.toml (one time)
mise install       # Install tools (Rust, Bun, cargo-tauri, etc.)
mise run setup     # Install Mac + mobile dev tools (fast, no VMs)
```

That's it. You're ready to build for macOS, iOS, and Android. VMs for Windows and Linux are set up automatically when you first need them.

> **Everything is idempotent.** Run any command as many times as you want — `mise install`, `setup`, `vm:up`, `vm:build` all skip work that's already done. Safe to re-run after errors, interruptions, or just to make sure things are current.

> **First run takes a while.** Box downloads are ~6 GB each, plus VM import and bootstrap. Subsequent runs are fast.

Add these to your `.gitignore`:

```
.build/
.mise/state/
.mise/logs/
```

## Build for every platform

```bash
mise run mac:dev              # macOS desktop dev mode (hot reload)
mise run mac:build            # macOS — .app + .dmg

mise run ios:sim              # iOS simulator (no signing required)
mise run ios:xcode            # Open in Xcode (physical device, debugging)
mise run ios:build            # iOS release IPA (requires signing)

mise run android:sim          # Android emulator
mise run android:studio       # Open in Android Studio (physical device, debugging)
mise run android:build        # Android — .apk + .aab

mise run windows:build        # Windows — .msi + .exe (VM auto-starts)
mise run linux:build          # Linux — .deb + .AppImage (VM auto-starts)

mise run all:build            # Build every platform at once

mise run icon                 # Generate all platform icons from app-icon.png
mise run screenshot           # Take screenshots via WebDriver
mise run clean:project        # Wipe this project's build artifacts
mise run clean:disk           # Free system-wide disk space (Rust targets, caches, simulators)
mise run clean:disk -- --dry-run  # Preview what would be cleaned
mise run clean:disk -- --deep # Also: Homebrew, Xcode archives, Docker
```

VM builds are fully automatic — if the VM doesn't exist, it downloads, imports, and bootstraps it. If the VM exists but is stopped, it starts it. Then it syncs your code, runs `mise run build` inside the VM, and pulls artifacts back to:

- `.build/windows/` — `.msi`, `.exe`
- `.build/linux/` — `.deb`, `.AppImage`, `.rpm`

Your project's `mise.toml` must define a `build` task (e.g., `run = "cargo tauri build"`) and the `platform:target` wrapper tasks. See [examples/tauri-basic/](examples/tauri-basic/).

## Four VMs

### Windows (ARM64 — utm/windows-11)

| VM | Purpose | Ports |
|---|---|---|
| **windows-build** | VS Build Tools, Rust, mise — for compiling | SSH:2222 RDP:3389 WinRM:5985 |
| **windows-test** | Clean Windows + SSH — for testing installers | SSH:2322 RDP:3489 WinRM:6985 |

### Linux (ARM64 — utm/ubuntu-24.04)

| VM | Purpose | Ports |
|---|---|---|
| **linux-build** | build-essential, Rust, mise, Tauri deps — for compiling | SSH:2422 |
| **linux-test** | Clean Ubuntu + SSH — for testing packages | SSH:2522 |

VMs are created on-demand by `vm:up` (called automatically by build tasks). Start them manually with `mise run vm:up`.

**Credentials:** `vagrant` / `vagrant` for all VMs (SSH, RDP, WinRM).

**RDP into Windows:** Use any Remote Desktop client (e.g., Microsoft Remote Desktop) to connect to `127.0.0.1:3389` (windows-build) or `127.0.0.1:3489` (windows-test) for a full GUI desktop.

## Infrastructure commands

These are hidden from `mise tasks` (use `mise tasks --hidden` to see them). Most devs never need these directly — the `platform:target` tasks call them automatically.

| Command | What it does |
|---|---|
| `mise run init` | Add tools + env to your mise.toml |
| `mise run setup` | Install Mac + mobile dev tools |
| `mise run doctor` | Check what's installed/missing |
| `mise run screenshot` | Take screenshots via WebDriver (runs `screenshots/take.ts` if present) |
| `mise run clean:disk` | Free system-wide disk space (`--dry-run`, `--deep`) |
| `mise run vm:up [profile]` | Start a VM (imports + bootstraps on first run) |
| `mise run vm:down [profile]` | Stop a VM |
| `mise run vm:build [profile]` | Sync + build in VM, pull artifacts back |
| `mise run vm:exec [profile] '<cmd>'` | Run a command inside a VM |
| `mise run vm:package [profile]` | Export a VM as a Vagrant box for sharing |
| `mise run vm:delete <profile>` | Delete a single VM, `utm` (all VMs + UTM app), or `all` |

Profiles: `windows-build`, `windows-test`, `linux-build`, `linux-test`

## Deleting and packaging

**Delete a single VM:**

```bash
mise run vm:delete windows-build  # removes VM + state file, keeps box cache
```

**Nuclear options:**

| Target | What it removes |
|---|---|
| `windows-build`, `windows-test`, `linux-build`, `linux-test` | That VM + its state file |
| `utm` | All 4 VMs + uninstalls UTM app |
| `all` | All VMs + UTM app + UTM data |

The box cache (`~/.cache/utm-dev/`) is always preserved — re-downloading ~6 GB is painful.

**Export a VM as a Vagrant box** (for sharing pre-configured VMs with your team):

```bash
mise run vm:package windows-build  # → .build/boxes/windows-11-windows-build_arm64.box
```

## Troubleshooting

**Logs:** Check `.mise/logs/` for detailed output from builds, SSH, and bootstrap.

**Manual SSH into a VM:**

```bash
sshpass -p vagrant ssh -p 2222 vagrant@127.0.0.1   # windows-build
sshpass -p vagrant ssh -p 2322 vagrant@127.0.0.1   # windows-test
sshpass -p vagrant ssh -p 2422 vagrant@127.0.0.1   # linux-build
sshpass -p vagrant ssh -p 2522 vagrant@127.0.0.1   # linux-test
```

**VM state is corrupted:** Delete the state file at `.mise/state/vm-{name}.env` and re-run `mise run vm:up` — it will re-import and bootstrap.

**Doctor says something is missing:** Run `mise run doctor` to see exactly what's installed and what's not. Fix what it flags, then re-run.

**Bootstrap is slow:** Windows VS Build Tools install takes 5-10 minutes. This is normal on first run.

## MCP (AI-assisted development)

utm-dev configures two MCP servers that give Claude Code (and other MCP-compatible tools) structured access to your dev environment and documentation.

**Set up MCP for your project:**

```bash
mise run mcp        # Writes .mcp.json + auto-allows permissions
```

This creates a `.mcp.json` in your project root. The task resolves full binary paths (required for sandboxed environments where PATH may not include mise-managed tools).

Add `.mcp.json` to your `.gitignore` — it contains machine-specific paths.

**Servers:**

| Server | What it does |
|---|---|
| **[context7](https://github.com/upstash/context7)** | Up-to-date docs and code examples for any library (Tauri v2, etc.) |
| **[mise](https://mise.jdx.dev)** | Query tools, tasks, env vars, config — and run tasks directly |

**Mise MCP capabilities:**

| Type | Capability | Description |
|---|---|---|
| Tool | `run_task` | Run any mise task (e.g., `doctor`, `vm:up windows-build`) |
| Tool | `install_tool` | Install a mise-managed tool (e.g., `node@20`) |
| Resource | `mise://tools` | Installed tools with versions and install paths |
| Resource | `mise://tasks` | All tasks (public + hidden) with descriptions and aliases |
| Resource | `mise://env` | Environment variables set by mise (ANDROID_HOME, etc.) |
| Resource | `mise://config` | Active config file paths and project root |

AI agents can query your environment natively — no need to shell out to `mise ls` or `mise tasks`.

## CI

Use [`jdx/mise-action@v2`](https://github.com/jdx/mise-action) in GitHub Actions to run tasks in CI.

## Examples

See [`examples/tauri-basic/`](examples/tauri-basic/) for a working Tauri app with per-platform build tasks.

## License

[MIT](LICENSE)
