---
title: "CI/CD Integration"
date: 2025-12-21
draft: false
weight: 1
---

# CI/CD Integration

utm-dev integrates with GitHub Actions for automated cross-platform builds.

## GitHub Actions

### macOS Build

```yaml
name: Build macOS

on:
  push:
    branches: [main]

jobs:
  build:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Build utm-dev
        run: go build .

      - name: Build macOS app
        run: go run . build macos examples/hybrid-dashboard

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: macos-app
          path: examples/hybrid-dashboard/.bin/macos/
```

### Android Build

```yaml
  build-android:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Install Android SDK and NDK
        run: |
          go run . install android-sdk
          go run . install android-ndk

      - name: Build Android APK
        run: go run . build android examples/hybrid-dashboard

      - name: Upload APK
        uses: actions/upload-artifact@v4
        with:
          name: android-apk
          path: examples/hybrid-dashboard/.bin/android/
```

### iOS Build (macOS runner required)

```yaml
  build-ios:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Build iOS app
        run: go run . build ios examples/hybrid-dashboard
```

### Windows Build

```yaml
  build-windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Build Windows app
        run: go run . build windows examples/hybrid-dashboard
```

### Multi-Platform Workflow

```yaml
name: Cross-Platform Build

on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: go test ./...
      - run: go vet ./...

  build-macos:
    runs-on: macos-latest
    needs: test
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: go run . build macos examples/hybrid-dashboard
      - uses: actions/upload-artifact@v4
        with:
          name: macos
          path: examples/hybrid-dashboard/.bin/macos/

  build-android:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'
      - run: |
          go run . install android-sdk
          go run . install android-ndk
          go run . build android examples/hybrid-dashboard
      - uses: actions/upload-artifact@v4
        with:
          name: android
          path: examples/hybrid-dashboard/.bin/android/
```

## SDK Caching

Cache SDKs across CI runs to avoid re-downloading:

```yaml
- name: Cache SDKs
  uses: actions/cache@v4
  with:
    path: |
      ~/utm-dev-sdks/
      ~/.cache/utm-dev/
    key: ${{ runner.os }}-sdks-${{ hashFiles('pkg/config/sdk-*.json') }}
    restore-keys: |
      ${{ runner.os }}-sdks-
```

## Build Caching

utm-dev's build cache is automatic. It hashes source files and skips rebuilds when nothing changed:

```bash
# First build -- compiles
go run . build macos examples/hybrid-dashboard

# Second build -- skips (no changes)
go run . build macos examples/hybrid-dashboard
# Output: up-to-date (use --force to rebuild)

# Force rebuild
go run . build --force macos examples/hybrid-dashboard

# Check if rebuild needed (for CI scripts)
go run . build --check macos examples/hybrid-dashboard
echo $?  # 0 = up-to-date, 1 = needs rebuild
```

## Release Workflow

### Tagged Release with Artifacts

```yaml
name: Release

on:
  push:
    tags: ['v*']

jobs:
  release:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.24'

      - name: Build and bundle
        run: |
          go run . build macos examples/hybrid-dashboard
          go run . bundle macos examples/hybrid-dashboard
          go run . package macos examples/hybrid-dashboard

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          files: |
            examples/hybrid-dashboard/.dist/*.tar.gz
```

## Windows from macOS (UTM)

utm-dev includes UTM virtual machine control for building and testing Windows apps from macOS:

```bash
# List VMs
utm-dev utm list

# Start a Windows VM
utm-dev utm start "Windows 11"

# Run a command in the VM
utm-dev utm exec "Windows 11" "utm-dev build windows examples/hybrid-dashboard"
```

This enables macOS-based CI to produce Windows builds without a separate Windows runner.

## Environment Variables

```bash
# Override SDK installation directory
export GOUP_SDK_DIR=/custom/sdk/path

# Android SDK paths (set automatically by utm-dev install)
export ANDROID_SDK_ROOT=~/utm-dev-sdks/android-sdk
export ANDROID_NDK_ROOT=~/utm-dev-sdks/ndk/26.1.10909125
```
