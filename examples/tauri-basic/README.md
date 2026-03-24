# tauri-basic

Minimal Tauri app showing how to build for all 5 platforms from a single Mac using [utm-dev](https://github.com/joeblew999/utm-dev).

This is a reference example — copy this structure into your own project. The key pieces are:

- **`mise.toml`** — platform tasks, remote task include, tools, env
- **`src-tauri/`** — standard Tauri app (Rust backend)
- **`ui/`** — frontend (plain HTML here, swap for React/Vue/etc.)

## Getting started

```bash
mise install       # Install tools (Rust, Bun, cargo-tauri, etc.)
mise run setup     # Install Mac + mobile dev tools (one time)
```

## Dev & build

```bash
mise run mac:dev              # macOS desktop (hot reload)
mise run mac:build            # macOS .app/.dmg

mise run ios:sim              # iOS simulator
mise run ios:xcode            # Open in Xcode (device/debugging)
mise run ios:build            # iOS release IPA (requires signing)

mise run android:sim          # Android emulator
mise run android:studio       # Open in Android Studio (device/debugging)
mise run android:build        # Android .apk/.aab

mise run windows:build        # Windows .msi/.exe (VM auto-starts)
mise run linux:build          # Linux .deb/.AppImage (VM auto-starts)

mise run all:build            # Build every platform at once

mise run icon                 # Generate all platform icons from app-icon.png
mise run clean:project        # Wipe build artifacts and start fresh
```

## How VM builds work

`windows:build` and `linux:build` call utm-dev's `vm:build` under the hood. It:

1. Starts the VM (downloads + bootstraps on first run)
2. Syncs your project into the VM
3. Runs `mise run build` inside the VM (the `build` task in this `mise.toml`)
4. Pulls artifacts back to `.build/windows/` or `.build/linux/`

Your `mise.toml` **must** define a `build` task — that's what the VM executes.

## Adapting for your project

1. Copy this `mise.toml` to your project root
2. Swap the `includes` line from local to remote:
   ```toml
   includes = ["git::https://github.com/joeblew999/utm-dev.git//.mise/tasks?ref=main"]
   ```
3. Run `mise run init` to add `[tools]` and `[env]` (or keep them from this example)
4. Replace `ui/` with your frontend framework
5. Customize `src-tauri/tauri.conf.json` with your app name, identifier, icons
