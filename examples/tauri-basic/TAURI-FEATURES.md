# Tauri Features Reference

What we're using, what we're not, and what's worth adding next. Keep this up to date.

---

## USING — Plugins active in this example

| Plugin | Crate | What it does | Demo tab |
|---|---|---|---|
| **Shell** | `tauri-plugin-shell` | Execute commands, open URLs in default browser | - |
| **OS** | `tauri-plugin-os` | Platform, arch, version, locale, hostname detection | System Info |
| **Dialog** | `tauri-plugin-dialog` | Native file open/save, message/ask/confirm boxes | Dialogs |
| **Store** | `tauri-plugin-store` | Persistent key-value store (survives restarts) | Key-Value Store |
| **Notification** | `tauri-plugin-notification` | Native OS notifications with permission model | Notifications |
| **Clipboard** | `tauri-plugin-clipboard-manager` | Read/write system clipboard | Clipboard |
| **Opener** | `tauri-plugin-opener` | Open URLs and files with the default app | Opener |
| **Process** | `tauri-plugin-process` | Exit, restart, process info | Tray menu |
| **Log** | `tauri-plugin-log` | Structured logging with file rotation | Logging |
| **Filesystem** | `tauri-plugin-fs` | Sandboxed file read/write with granular permissions | Filesystem |
| **Updater** | `tauri-plugin-updater` | Auto-update with signed packages (free Tauri signing keys) | Updater |
| **Window State** | `tauri-plugin-window-state` | Remembers/restores window size + position across restarts | Automatic |
| **Single Instance** | `tauri-plugin-single-instance` | Prevents multiple app instances, focuses existing window | Automatic (desktop) |
| **Global Shortcut** | `tauri-plugin-global-shortcut` | System-wide hotkeys (CmdOrCtrl+Shift+T toggles window) | Global Shortcuts |
| **Autostart** | `tauri-plugin-autostart` | Launch on login (enable/disable/check) | Autostart |
| **Deep Link** | `tauri-plugin-deep-link` | Custom URL protocol handling | Deep Link |

## USING — Core features active in this example

| Feature | Where | Notes |
|---|---|---|
| **System Tray** | `lib.rs` | Icon, menu (Show/Quit), click to focus. Desktop only. |
| **IPC Commands** | `lib.rs` → `index.html` | `greet`, `get_system_info`, `log_from_frontend` |
| **Event System** | `lib.rs` ↔ `index.html` | `backend-event` (Rust→JS), `frontend-event` (JS→Rust) |
| **Capabilities** | `capabilities/` | Fine-grained permissions split by platform (default + desktop + mobile) |
| **CSP** | `tauri.conf.json` | Content Security Policy enforced |
| **withGlobalTauri** | `tauri.conf.json` | Exposes `window.__TAURI__` for no-bundler setups |
| **WebDriver** | `lib.rs` (feature-gated) | `tauri-plugin-webdriver` for automated screenshots and E2E testing |
| **Update Signing** | `tauri.conf.json` | Free keypair for update integrity (not OS code signing) |

## USING — Screenshots

`mise run screenshot` uses `tauri-plugin-webdriver` (W3C WebDriver embedded in the app) to capture screenshots programmatically.

- The shared utm-dev task handles: build with `--features webdriver`, launch app + proxy, create session
- If `screenshots/take.ts` exists in your project, it runs that with `WEBDRIVER_URL` + `WEBDRIVER_SESSION` env vars
- Otherwise, takes a single screenshot to `screenshots/app.png`
- The example's `take.ts` discovers tabs from the HTML and captures 12 screenshots
- Uses plain `fetch()` against the standard W3C WebDriver API — no utm-dev imports needed

---

## NOT USING — Official plugins worth adding next

### High priority — add for any shipped app

| Plugin | Crate | Use case | Why we don't have it yet |
|---|---|---|---|
| **SQL** | `tauri-plugin-sql` | SQLite/MySQL/Postgres from the app | Overkill for demo; real apps add this themselves |

### Desktop features

| Plugin | Crate | Use case |
|---|---|---|
| **Positioner** | `tauri-plugin-positioner` | Position windows relative to tray, screen edges |
| **Prevent Default** | `tauri-plugin-prevent-default` | Disable right-click, text selection, etc. |

### Data & networking

| Plugin | Crate | Use case |
|---|---|---|
| **HTTP** | `tauri-plugin-http` | HTTP client from frontend (bypasses CORS) |
| **Stronghold** | `tauri-plugin-stronghold` | Encrypted secrets vault (keys, tokens) |
| **Upload** | `tauri-plugin-upload` | File upload with progress |
| **Websocket** | `tauri-plugin-websocket` | WebSocket client from frontend |

### Mobile

| Plugin | Crate | Use case |
|---|---|---|
| **Biometric** | `tauri-plugin-biometric` | Touch ID / fingerprint auth |
| **Barcode Scanner** | `tauri-plugin-barcode-scanner` | Camera barcode/QR scanning |
| **NFC** | `tauri-plugin-nfc` | NFC tag reading |
| **Haptics** | `tauri-plugin-haptics` | Vibration feedback |
### Community plugins — worth knowing about

| Plugin | Source | Use case |
|---|---|---|
| **window-vibrancy** | community | Frosted glass / translucent window effects (macOS blur, Windows Acrylic/Mica) |
| **sentry-tauri** | Sentry | Error monitoring — captures JS errors, Rust panics, native crash minidumps |
| **tauri-plugin-aptabase** | Aptabase | Privacy-first, minimal analytics for tracking app usage |
| **tauri-plugin-clipboard** (extended) | community | Clipboard beyond text: images, monitoring for changes |
| **tauri-plugin-serialport** | community | Serial port communication (Arduino, microcontrollers, hardware integration) |

### Developer tools

| Tool | What it does | Status |
|---|---|---|
| **tauri-specta** | TypeSafe TypeScript bindings for Rust commands — eliminates manual IPC sync errors | Worth adding as app grows |
| **tauri-plugin-webdriver** | W3C WebDriver for E2E testing and screenshots | **USING** (feature-gated) |
| **Tauri JetBrains Plugin** | Scaffolding and debugging in IntelliJ/WebStorm | IDE plugin, not a crate |

---

## NOT USING — Core Tauri features (no plugin needed)

Built into Tauri, configured via `tauri.conf.json` or Rust code:

- **Multi-window** — spawn windows from Rust or JS
- **Menu bar** — native app menus with keyboard shortcuts
- **Drag and drop** — file drop events on windows
- **Window decorations** — custom titlebar, transparent, always-on-top, frameless
- **Embedded assets** — bundle static files efficiently
- **App lifecycle** — `setup`, window close requested, `will-quit`
- **Custom URI scheme** — `tauri://` protocol for loading assets
- **Security** — CSP, capabilities, permission scoping per window
- **iOS/Android bridges** — Swift/Kotlin plugin bridges for native APIs

---

## Signing notes

There are **two completely different** types of signing:

| Type | Who checks it | Cost | What it does |
|---|---|---|---|
| **OS code signing** | Windows SmartScreen, macOS Gatekeeper | $100-500/year | Removes "Unknown Publisher" warnings |
| **Tauri update signing** | Your app's updater plugin | Free (CLI-generated) | Verifies update packages haven't been tampered with |

The free Tauri signing keys (configured in `plugins.updater.pubkey`) do NOT remove OS warnings. They only enable safe auto-updates. The OS signing paywall is Apple/Microsoft's — no plugin fixes it.

To generate keys: `cargo tauri signer generate -w .tauri-key`

## Architecture notes

- `withGlobalTauri: true` in `tauri.conf.json` (under `app`, not `build`) exposes all APIs at `window.__TAURI__`
- Capabilities split by platform: `default.json` (shared), `desktop.json`, `mobile.json`
- The store plugin writes to the app's data directory (platform-specific)
- Logging writes to the app's log directory with automatic rotation
- System tray is `#[cfg(desktop)]` — excluded from mobile builds automatically
- WebDriver plugin is behind a Cargo feature flag (`--features webdriver`) — not in production builds
- Updater plugin **requires** `plugins.updater.pubkey` in `tauri.conf.json` or it panics on startup. Endpoints can be empty for demo.
- Single instance plugin must be registered **first** in the builder chain
- Global shortcut `CmdOrCtrl+Shift+T` is registered at startup for window toggle
- Window state plugin works silently — no UI, no config, just register it
- All official plugins: [tauri-apps/plugins-workspace](https://github.com/tauri-apps/plugins-workspace)
- Community plugins: [tauri-apps/awesome-tauri](https://github.com/tauri-apps/awesome-tauri)
