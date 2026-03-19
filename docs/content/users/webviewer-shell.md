---
title: "Webviewer Shell"
date: 2026-02-11
draft: false
weight: 5
---

# Webviewer Shell - Run Any Website as a Desktop App

The **Webviewer Shell** is a pre-built native desktop app that loads any website inside a native webview. No programming, no compilation, no SDKs needed.

## How It Works

```
┌──────────────────────────────────┐
│  Your Website (any URL)          │
│  loaded inside a native webview  │
│                                  │
│  ┌────────────────────────────┐  │
│  │  https://your-website.com  │  │
│  │                            │  │
│  │  Your web app runs here    │  │
│  │  with full browser APIs    │  │
│  │  (OPFS, Service Workers,   │  │
│  │   WebSocket, etc.)         │  │
│  └────────────────────────────┘  │
│                                  │
│  Native window (macOS/Windows)   │
└──────────────────────────────────┘
```

## Quick Start

### 1. Download

Go to [GitHub Releases](https://github.com/joeblew999/utm-dev/releases) and download the zip for your platform:

- **macOS**: `webviewer-shell-macos.zip`
- **Windows**: `webviewer-shell-windows.zip`

### 2. Unzip

Extract the zip file. You'll see:

```
webviewer-shell/
├── gio-plugin-webviewer.app   (macOS) or .exe (Windows)
├── app.json                   ← Edit this!
└── README.txt
```

### 3. Configure

Open `app.json` in any text editor and change the `url` to your website:

```json
{
    "url": "https://your-website.com",
    "name": "My App",
    "width": 1200,
    "height": 800
}
```

That's it — just change the URL.

### 4. Launch

- **macOS**: Double-click the `.app` file
- **Windows**: Double-click the `.exe` file

Your website is now running as a native desktop app.

## Configuration Reference

The `app.json` file controls how the shell behaves:

| Field    | Required | Default          | Description                     |
|----------|----------|------------------|---------------------------------|
| `url`    | Yes      | —                | Website URL to load             |
| `name`   | No       | "Gio WebViewer"  | Window title                    |
| `width`  | No       | 1200             | Window width in pixels          |
| `height` | No       | 800              | Window height in pixels         |

### Minimal Config

```json
{
    "url": "https://your-website.com"
}
```

### Full Config

```json
{
    "url": "https://your-website.com",
    "name": "My Cool App",
    "width": 1200,
    "height": 800,
    "update": {
        "repo": "your-github-user/your-repo",
        "asset": "webviewer-shell"
    }
}
```

## Self-Update

The shell can update itself from GitHub releases. It checks automatically on startup and prints a notice if a new version is available.

### Update Config

Add an `update` section to `app.json`:

```json
{
    "url": "https://your-website.com",
    "update": {
        "repo": "joeblew999/utm-dev",
        "asset": "webviewer-shell"
    }
}
```

- `repo`: GitHub owner/repo where release zips are published
- `asset`: The prefix of the zip file name (e.g., `webviewer-shell` matches `webviewer-shell-macos.zip`)

### Running an Update

On **macOS**, open Terminal and run:

```bash
./gio-plugin-webviewer.app/Contents/MacOS/gio-plugin-webviewer --update
```

On **Windows**, open Command Prompt and run:

```cmd
gio-plugin-webviewer.exe --update
```

## Platform Notes

### macOS - Gatekeeper

The first time you open the app, macOS may block it with:

> "gio-plugin-webviewer.app can't be opened because Apple cannot check it for malicious software."

**Fix (easiest):** Right-click the app → **Open** → click **Open** in the dialog.

**Fix (Terminal):**
```bash
xattr -cr gio-plugin-webviewer.app
```

### Windows - SmartScreen

Windows may show a SmartScreen warning. Click **More info** → **Run anyway**.

### Supported Web APIs

The webview uses the platform's native web engine:
- **macOS**: WKWebView (Safari engine)
- **Windows**: WebView2 (Edge/Chromium engine)

Most modern web APIs work, including:
- Service Workers
- OPFS (Origin Private File System)
- SharedWorker
- WebSocket
- IndexedDB
- WebAssembly

## Offline/Online Apps

The shell works great with Progressive Web Apps and offline-capable websites. If your website uses Service Workers, OPFS, or WASM SQLite for local storage, it all works inside the native webview.

## Troubleshooting

| Problem | Solution |
|---------|----------|
| Black screen | Check that `app.json` has a valid URL starting with `http://` or `https://` |
| App won't open (macOS) | Right-click → Open → Open (Gatekeeper fix) |
| App won't open (Windows) | Click "More info" → "Run anyway" (SmartScreen) |
| Wrong website | Edit `app.json` and relaunch |
| Window too small/large | Change `width` and `height` in `app.json` |
| Update fails | Check internet connection and that `update.repo` is correct |

## For Developers

### Building from Source

```bash
# Build webviewer shell for macOS
task build:webviewer:macos

# Or manually
go run . build macos examples/gio-plugin-webviewer
```

### Using Taskfile

```bash
# Build and run with a custom URL
task run:shell URL=https://your-website.com

# Test self-update
task run:shell:update
```

### Publishing Your Own Shell

1. Fork the repo or set up your own GitHub releases
2. Build the shell for your target platforms
3. Create a zip with the binary + your customized `app.json` + `README.txt`
4. Publish as a GitHub release
5. Set `update.repo` and `update.asset` in `app.json` to point to your releases

### CI/CD

The shell is built automatically by GitHub Actions on tagged releases. See `.github/workflows/demo-release.yml`.

Published artifacts:
- `webviewer-shell-macos.zip` — macOS app bundle + app.json + README
- `webviewer-shell-windows.zip` — Windows exe + app.json + README
