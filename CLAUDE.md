# CLAUDE.md — utm-dev

Installs from nothing. Sets up UTM with Windows VMs so we can build Tauri apps for all platforms (macOS, iOS, Android, Windows) from a single Mac.

## Stack

- **mise** for task running, tool management, orchestration
- **Bun** (TypeScript) for all task scripts
- **Tauri** (Rust) for the actual apps

## Task architecture

```
.mise/tasks/
├── _lib.ts          # shared: VM profiles, SSH helpers, state, logging
├── _winrm.ts        # WinRM SOAP client over fetch
├── _utm.ts          # UTM operations: install, import box, network, start/stop
├── _bootstrap.ts    # internal: WinRM bootstrap (SSH, VS Build Tools, mise)
├── init.ts          # adds [tools]+[env] to project's mise.toml
├── setup.ts         # installs EVERYTHING: Mac tools + UTM + both VMs
├── doctor.ts        # health check
└── vm/
    ├── up.ts        # start a VM (assumes setup was run)
    ├── down.ts      # stop a VM
    ├── exec.ts      # run command in VM via SSH
    ├── build.ts     # sync code + build + pull artifacts (build VM only)
    ├── delete.ts    # delete VMs/UTM/data
    └── package.ts   # export VM as Vagrant box
```

Files prefixed with `_` are internal modules (not user-facing tasks).

## Tasks

```bash
mise run init                        # Add tools + env to project's mise.toml
mise run setup                       # Install EVERYTHING: Mac tools, SDKs, UTM, both VMs
mise run doctor                      # Check what's installed/missing
mise run vm:up [build|test]          # Start a VM (default: build)
mise run vm:down [build|test]        # Stop a VM
mise run vm:exec [build|test] <cmd>  # Run a command in a VM via SSH
mise run vm:build                    # Sync + build in build VM, pull .msi/.exe back
mise run vm:package [build|test]     # Export VM as reusable Vagrant box
mise run vm:delete build|test|utm|all
```

## Key design decisions

- **`setup` does ALL installing** — Mac tools, Android SDK, UTM, both VMs, bootstrap. One command.
- **`vm:up` just starts** — fast, no downloading/importing. Tells you to run setup if VM not found.
- **No vm:sync task** — sync is inlined in vm:build. Nobody syncs without building.
- **`_bootstrap.ts` is internal** — called by setup, not exposed as a task.
- **`_utm.ts` is a library** — UTM operations shared by setup and vm:up.

## Two VMs

| Profile | Purpose | SSH | RDP | WinRM | Bootstrap |
|---|---|---|---|---|---|
| **build** | VS Build Tools, Rust, mise | 2222 | 3389 | 5985 | full |
| **test** | Clean Windows + SSH only | 2322 | 3489 | 6985 | ssh-only |

Profiles in `_lib.ts`. State per-VM at `.mise/state/vm-{name}.env`.

## Box cache

`~/.cache/utm-dev/` (~6 GB). **Never delete this.**

## UTM VM details

- Box: `windows-11` from Vagrant Cloud (utm/windows-11)
- Credentials: vagrant / vagrant
- UTM is sandboxed — prefs must write to container plist, not defaults domain
- SSH bootstrapped via WinRM (scheduled task as SYSTEM to bypass UAC)

## Remote task include

```toml
[task_config]
includes = ["git::https://github.com/joeblew999/utm-dev.git//.mise/tasks?ref=main"]
```

`mise run init` adds `[tools]` and `[env]` blocks.

## CI

`jdx/mise-action@v2` everywhere.
