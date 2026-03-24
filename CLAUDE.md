# CLAUDE.md — utm-dev

Installs from nothing. Sets up UTM with a Windows VM so we can build Tauri apps for all platforms (macOS, iOS, Android, Windows) from a single Mac.

## Stack

- **mise** for task running, tool management, orchestration (deps, env, sources/outputs)
- **Bun** (TypeScript) for all task scripts — cross-platform (Mac + Windows)
- **Tauri** (Rust) for the actual apps

## Task architecture

```
mise.toml                   # tools + env (what init adds to consumer projects)
package.json                # bun project manifest (bun-types)
tsconfig.json               # TypeScript config for IDE support
.mise/tasks/
├── _lib.ts                 # shared: constants, SSH helpers, logging
├── _winrm.ts               # WinRM SOAP client over fetch (replaces pywinrm)
├── init.ts                 # adds [tools]+[env] to project's mise.toml
├── setup.ts                # installs Tauri dev prereqs (Rust, Android SDK, iOS)
├── doctor.ts               # health check for all tools and VM status
└── vm/
    ├── up.ts               # install UTM + download VM + start + bootstrap
    ├── bootstrap.ts        # WinRM bootstrap (SSH, VS Build Tools, mise)
    ├── sync.ts             # tar+scp project to VM (sources/outputs for skip)
    ├── exec.ts             # run command in VM via SSH
    ├── build.ts            # depends on vm:sync, build in VM, pull artifacts
    ├── down.ts             # stop VM
    ├── delete.ts           # delete VM/UTM/data
    └── package.ts          # export VM as reusable Vagrant box
examples/
└── tauri-basic/            # minimal Tauri app demonstrating all platforms
    ├── mise.toml           # remote include + app tasks (dev, build, ios, android)
    ├── ui/index.html       # static frontend
    └── src-tauri/          # Rust backend (Tauri standard layout)
```

All task files are TypeScript (`.ts`) with `#!/usr/bin/env bun` shebangs. mise strips the extension automatically — `vm/sync.ts` becomes `mise run vm:sync`.

## Tasks

```bash
mise run init               # Add tools + env to project's mise.toml
mise run setup              # Install Tauri dev prereqs (Rust, Android SDK, iOS deps)
mise run doctor             # Check what's installed and what's missing
mise run vm:up              # Install UTM + Windows VM + bootstrap SSH + Rust
mise run vm:build           # Sync code, build in VM, pull artifacts back
mise run vm:sync            # Sync project files to VM
mise run vm:exec <cmd>      # Run a command in the VM via SSH
mise run vm:bootstrap       # Bootstrap SSH + Rust in VM (called by vm:up)
mise run vm:down            # Stop the VM
mise run vm:package         # Export VM as reusable Vagrant box
mise run vm:delete vm       # Delete VM (keeps UTM + box cache)
mise run vm:delete utm      # Delete VM + uninstall UTM (keeps box cache)
mise run vm:delete all      # Delete VM + UTM + app data (keeps box cache)
```

## Key mise features used

- **`depends`** — vm:build depends on vm:sync (no copy-paste)
- **`sources`/`outputs`** — vm:sync skips if source files unchanged
- **`[tools]`** — manages cargo-tauri, bun, xcodegen, ruby, java
- **Remote includes** — other projects pull `.mise/tasks/` via git URL
- **`_lib.ts`** — shared module at tasks root, imported by all tasks
- **`_winrm.ts`** — reusable WinRM SOAP client class at tasks root (used by bootstrap)

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

1. `vm:sync` — tar project, scp to VM, extract (skipped if sources unchanged)
2. `vm:build` — depends on vm:sync, then `cargo tauri build` in VM, pull .msi/.exe back to `.build/windows/`
3. `vm:exec` — run ad-hoc commands via sshpass over SSH

## Remote task include (how other devs use this)

Other projects pull in tasks via mise remote includes in their `mise.toml`:

```toml
[task_config]
includes = ["git::https://github.com/joeblew999/utm-dev.git//.mise/tasks?ref=main"]
```

Scripts use `PROJECT_DIR` (pwd) for logs/state. `_lib.ts` is imported via relative path so tasks work both locally and as remote includes.

`mise run init` adds the `[tools]` and `[env]` blocks needed for Tauri builds.

## Dependencies on dev's machine

- **bun** — managed by mise `[tools]`, runs all task scripts + WinRM bootstrap
- **sshpass** — auto-installed by vm:* tasks via brew on first use

## Examples

- `examples/tauri-basic/` — minimal Tauri app with mise tasks for every platform

## CI

`jdx/mise-action@v2` everywhere. No Taskfile.
