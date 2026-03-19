---
title: "Platform Support"
date: 2025-12-21
draft: false
weight: 2
---

# Platform Support

utm-dev builds native Gio applications for macOS, iOS, Android, and Windows. Each platform uses the system's native webview engine for hybrid apps.

## Platform Matrix

| Platform | Build Command | Webview Engine | Host OS Required | Status |
|----------|--------------|----------------|------------------|--------|
| macOS | `build macos` | WKWebView (Safari) | macOS | Working |
| iOS | `build ios` | WKWebView (Safari) | macOS + Xcode | Working |
| iOS Simulator | `build ios-simulator` | WKWebView | macOS + Xcode | Working |
| Android | `build android` | Chromium WebView | Any (needs Android SDK) | Working |
| Windows | `build windows` | WebView2 (Edge) | Windows or cross-compile | Working |

## macOS

**Build:**
```bash
utm-dev build macos examples/hybrid-dashboard
```

**Output:** `.app` bundle in `<app>/.bin/macos/`

**Requirements:**
- macOS host
- Xcode Command Line Tools (`xcode-select --install`)

**Webview:** WKWebView (Safari engine). Full HTML5, Service Workers, OPFS, WebSocket, IndexedDB, WebAssembly support.

**Distribution:** Use `utm-dev bundle macos` for code-signed bundles, then `utm-dev package macos` for tar.gz archives. See [Packaging](/users/packaging/).

**Deep linking:** Supported via `--schemes` flag:
```bash
utm-dev build macos examples/hybrid-dashboard --schemes "myapp://,https://example.com"
```

## iOS

**Build:**
```bash
# For device
utm-dev build ios examples/hybrid-dashboard

# For simulator
utm-dev build ios-simulator examples/hybrid-dashboard
```

**Output:** `.app` bundle in `<app>/.bin/ios/` or `<app>/.bin/ios-simulator/`

**Requirements:**
- macOS host
- Xcode (install from App Store)
- For device builds: Apple Developer account and provisioning profile

**Webview:** WKWebView (Safari engine). Same capabilities as macOS.

**Signing:**
```bash
utm-dev build ios examples/hybrid-dashboard --signkey /path/to/profile.mobileprovision
```

**Deep linking:**
```bash
utm-dev build ios examples/hybrid-dashboard --schemes "myapp://,https://example.com"
```

## Android

**Build:**
```bash
utm-dev build android examples/hybrid-dashboard
```

**Output:** `.apk` in `<app>/.bin/android/`

**Requirements:**
- Any host OS
- Android SDK and NDK (utm-dev installs these):
  ```bash
  utm-dev install android-sdk
  utm-dev install android-ndk
  ```

**Webview:** System Chromium WebView. Full modern web API support.

**Signing:**
```bash
utm-dev build android examples/hybrid-dashboard --signkey /path/to/keystore.jks
```

**Deep linking and intent queries:**
```bash
utm-dev build android examples/hybrid-dashboard \
  --schemes "myapp://,https://example.com" \
  --queries "com.google.android.apps.maps"
```

## Windows

**Build:**
```bash
utm-dev build windows examples/hybrid-dashboard
```

**Output:** `.exe` in `<app>/.bin/windows/`

**Requirements:**
- Windows host (or cross-compilation from macOS/Linux with CGo)
- For webview apps: WebView2 runtime (included in Windows 10/11)

**Webview:** WebView2 (Edge/Chromium engine). Full modern web API support.

**Note:** Cross-compiling Windows apps from macOS works for pure Go apps. Webview-based apps may require a Windows build environment. utm-dev supports [UTM virtual machines](/dev/cicd/) for Windows builds from macOS.

## Linux

**Status:** Gio UI supports Linux natively. utm-dev does not currently have a dedicated `build linux` command, but you can build Gio apps for Linux using standard Go:

```bash
GOOS=linux GOARCH=amd64 go build -o myapp ./examples/hybrid-dashboard
```

**Webview:** WebKitGTK. Requires `libwebkit2gtk-4.0-dev` system package.

## Web / WASM

**Status:** Not currently supported by utm-dev. Gio UI compiles to WASM, but the webview plugin does not work in a browser context (a webview inside a browser doesn't make sense). Pure Gio UI apps (without webview) can be compiled to WASM using standard Go tools.

## Gio Version Compatibility

**This is critical.** Mismatched Gio and gio-plugins versions cause runtime panics.

Use these specific versions for webview-based apps:

```bash
go get gioui.org@7bcb315ee174
go get github.com/gioui-plugins/gio-plugins@v0.9.1
go mod tidy
```

This gives you:
- `gioui.org v0.9.1-0.20251215212054-7bcb315ee174`
- `github.com/gioui-plugins/gio-plugins v0.9.1`

Do **not** use `@latest` -- it may pull incompatible versions.

## All Build Flags

```bash
utm-dev build [platform] [app-directory] [flags]

Flags:
  --force            Force rebuild even if up-to-date
  --check            Check if rebuild needed (exit 0=no, 1=yes)
  --output           Custom output directory
  --schemes          Deep linking URI schemes (comma-separated)
  --queries          Android app package queries (comma-separated)
  --signkey          Signing key path (keystore, Keychain key, or provisioning profile)
  --skip-icons       Skip icon generation during build
```
