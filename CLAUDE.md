# CLAUDE.md — utm-dev

Installs from nothing. Sets up UTM with Windows and Linux VMs so we can build Tauri apps for all platforms (macOS, iOS, Android, Windows, Linux) from a single Mac.

## Stack

- **mise** for task running, tool management, orchestration
- **Bun** (TypeScript) for all task scripts
- **Tauri** (Rust) for the actual apps

## Task architecture

```
.mise/tasks/
├── _lib.ts            # shared: VM profiles, SSH helpers, state, logging
├── _winrm.ts          # WinRM SOAP client over fetch
├── _utm.ts            # UTM operations: install, import box, network, start/stop
├── _bootstrap.ts      # internal: WinRM bootstrap (SSH, VS Build Tools, mise)
├── _bootstrapLinux.ts  # internal: SSH bootstrap (build-essential, Rust, mise, Tauri deps)
├── init.ts            # adds [tools]+[env] to project's mise.toml
├── setup.ts           # Mac + mobile dev tools (Rust, Android SDK, iOS)
├── doctor.ts          # health check
├── _screenshot.ts     # shared: WebDriver session management for screenshots
├── screenshot.ts      # take screenshots via tauri-plugin-webdriver
├── mcp.ts             # set up MCP servers (.mcp.json) for AI-assisted dev
├── clean/
│   └── disk.ts        # free system-wide disk space
└── vm/                # hidden plumbing — called by platform:target tasks
    ├── up.ts          # start a VM (imports + bootstraps on first run)
    ├── down.ts        # stop a VM
    ├── exec.ts        # run command in VM via SSH
    ├── build.ts       # build app in the VM (auto-starts VM if needed)
    ├── delete.ts      # delete a VM or UTM
    └── package.ts     # export VM as Vagrant box
```

Files prefixed with `_` are internal modules (not user-facing tasks).
Files in `vm/` are hidden (`hide=true`) — they work but don't show in `mise tasks`.

## Consumer-facing tasks (defined in the example/consumer mise.toml)

```
mac:dev           Run macOS desktop app (hot reload)
mac:build         Build macOS .app and .dmg
ios:sim           Run on iOS simulator (no signing required)
ios:xcode         Open in Xcode (for physical device or debugging)
ios:build         Build iOS release IPA (requires signing)
android:sim       Run on Android emulator
android:studio    Open in Android Studio (for physical device or debugging)
android:build     Build Android release APK and AAB
windows:build     Build Windows .msi/.exe in the VM
linux:dev         Start Linux desktop VM for dev (Debian 12 + GNOME)
linux:build       Build Linux .deb/.AppImage in the VM
icon              Generate all platform icons from app-icon.png
```

## utm-dev infrastructure tasks

```bash
mise run init                                       # Add tools + env to project's mise.toml
mise run setup                                      # Install dev tools (platform-aware: macOS or Linux)
mise run doctor                                     # Check what's installed/missing
mise run screenshot                                 # Take screenshots via WebDriver (runs screenshots/take.ts if present)
mise run clean:disk                                 # Free system-wide disk space
mise run clean:disk -- --dry-run                    # Preview what would be cleaned
mise run clean:disk -- --deep                       # Also: Homebrew, Xcode archives, Docker, device support
mise run mcp                                        # Set up MCP servers (.mcp.json) for AI-assisted dev
```

Hidden (use `mise tasks --hidden` to see):

```bash
mise run vm:up [windows-build|windows-test|linux-build|linux-test|linux-dev]   # Start a VM
mise run vm:down [windows-build|windows-test|linux-build|linux-test|linux-dev]  # Stop a VM
mise run vm:exec [windows-build|windows-test|linux-build|linux-test|linux-dev] <cmd>  # Run via SSH
mise run vm:build [windows-build|linux-build]                                   # Build app in a VM
mise run vm:package [windows-build|windows-test|linux-build|linux-test|linux-dev]  # Export as box
mise run vm:delete windows-build|windows-test|linux-build|linux-test|linux-dev|utm|all
```

## Key design decisions

- **`setup` is platform-aware** — macOS: Xcode, Android SDK, iOS deps. Linux: apt system deps only (Rust/bun via mise).
- **`vm:up` is lazy** — first run downloads box, imports, configures network, bootstraps. After that, just starts.
- **`vm:build` auto-starts** — if SSH isn't reachable, calls `vm:up` automatically. Fully idempotent cascade.
- **`vm:*` tasks are hidden** — plumbing, not porcelain. Devs use `platform:target` tasks.
- **No vm:sync task** — sync is inlined in vm:build. Nobody syncs without building.
- **`_bootstrap.ts` is internal** — called by vm:up on first run, not exposed as a task.
- **`_utm.ts` is a library** — UTM operations shared by vm:up and other vm:* tasks.

## VMs

### Windows (ARM64 — utm/windows-11)

| Profile | Purpose | SSH | RDP | WinRM | Bootstrap |
|---|---|---|---|---|---|
| **windows-build** | VS Build Tools, Rust, mise | 2222 | 3389 | 5985 | full |
| **windows-test** | Clean Windows + SSH only | 2322 | 3489 | 6985 | ssh-only |

### Linux (ARM64)

| Profile | Box | Purpose | SSH | Bootstrap |
|---|---|---|---|---|
| **linux-dev** | debian-12 (GNOME) | Full desktop dev experience | 2622 | full |
| **linux-build** | ubuntu-24.04 | Headless builds | 2422 | full |
| **linux-test** | ubuntu-24.04 | Clean machine testing | 2522 | ssh-only |

Profiles in `_lib.ts`. State per-VM at `.mise/state/vm-{name}.env`.

## Box cache

`~/.cache/utm-dev/` (~6 GB). **Never delete this.**

## UTM VM details

- Windows box: `windows-11` from Vagrant Cloud (utm/windows-11) — ARM64
- Linux box: `ubuntu-24.04` from Vagrant Cloud (utm/ubuntu-24.04) — ARM64
- Credentials: vagrant / vagrant
- UTM is sandboxed — prefs must write to container plist, not defaults domain
- Windows SSH bootstrapped via WinRM (scheduled task as SYSTEM to bypass UAC)
- Linux SSH available out of the box (Vagrant boxes ship with it)

## Remote task include

```toml
[task_config]
includes = ["git::https://github.com/joeblew999/utm-dev.git//.mise/tasks?ref=main"]
```

`mise run init` adds `[tools]` and `[env]` blocks.

## MCP servers

`mise run mcp` configures two MCP servers in `.mcp.json`:

- **context7** — retrieves up-to-date docs and code examples for any library
- **mise** — exposes tools, tasks, env vars, and config as structured data

### Mise MCP capabilities

**Tools (actions):**
- `run_task` — run any mise task (e.g., `doctor`, `vm:up windows-build`, `clean:disk`)
- `install_tool` — install a mise-managed tool (e.g., `node@20`)

**Resources (read-only queries):**
- `mise://tools` — installed tools with versions, install paths, and which config declared them
- `mise://tasks` — all tasks (public + hidden) with descriptions, aliases, and metadata
- `mise://env` — environment variables set by mise (ANDROID_HOME, NDK_HOME, JAVA_HOME)
- `mise://config` — active config file paths and project root

Prefer MCP tools/resources over Bash for mise operations — they provide structured data and work in sandboxed environments where PATH resolution fails.

## CI

`jdx/mise-action@v2` everywhere.
