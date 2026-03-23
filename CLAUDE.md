# CLAUDE.md — utm-dev

Installs from nothing. Sets up UTM with a Windows VM so we can build Tauri apps for all platforms (macOS, iOS, Android, Windows) from a single Mac.

## Stack

- **mise** for task running and tool management (no Go, no Taskfile)
- **Bash scripts** in `.mise/tasks/` for all automation (AppleScript inlined in vm:up)
- **Tauri** (Rust) for the actual apps

## Tasks

```bash
mise run init               # Add tools + env to project's mise.toml
mise run setup              # Install Tauri dev prereqs (Rust, Android SDK, iOS deps)
mise run vm:up              # Install UTM + Windows VM + bootstrap SSH + Rust
mise run vm:build           # Sync code, build in VM, pull artifacts back
mise run vm:sync            # Sync project files to VM
mise run vm:exec <cmd>      # Run a command in the VM via SSH
mise run vm:bootstrap       # Bootstrap SSH + Rust in VM (called by vm:up)
mise run vm:down            # Stop the VM
mise run vm:delete vm       # Delete VM (keeps UTM + box cache)
mise run vm:delete utm      # Delete VM + uninstall UTM (keeps box cache)
mise run vm:delete all      # Delete VM + UTM + app data (keeps box cache)
```

## Box cache

The Windows VM box (~6 GB) is cached at `~/.cache/utm-dev/`. **Never delete this** — re-downloading takes forever.

## UTM VM details

- Box: `windows-11` from Vagrant Cloud (utm/windows-11)
- Credentials: vagrant / vagrant
- Ports: SSH localhost:2222, RDP localhost:3389, WinRM localhost:5985
- Network: shared (internet) + emulated (port forwards)
- UTM is sandboxed — prefs must write to container plist, not defaults domain
- SSH is bootstrapped via WinRM (scheduled task as SYSTEM to bypass UAC)
- Rust + cargo-tauri installed in VM by vm:bootstrap

## Windows build pipeline

1. `vm:sync` — tar project, scp to VM, extract
2. `vm:exec` — run commands via sshpass over SSH
3. `vm:build` — sync + `cargo tauri build` + pull .msi/.exe back to `.build/windows/`

## Remote task include (how other devs use this)

Other projects pull in tasks via mise remote includes in their `mise.toml`:

```toml
[task_config]
includes = ["git::https://github.com/joeblew999/utm-dev.git//.mise/tasks?ref=main"]
```

Scripts use `PROJECT_DIR` (pwd) for logs/state. All AppleScript is inlined (no external
file deps) so tasks work both locally and as remote includes.

`mise run init` adds the `[tools]` and `[env]` blocks needed for Tauri builds.

## Dependencies on dev's machine

- sshpass (auto-installed by vm:exec/vm:sync/vm:build via brew)
- pywinrm (auto-installed by vm:bootstrap via pip3)

## Examples

- `examples/tauri-basic/` — minimal Tauri app with mise tasks for every platform

## CI

`jdx/mise-action@v2` everywhere. No Taskfile.
