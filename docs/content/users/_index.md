---
title: "User Guides"
date: 2025-12-21
draft: false
weight: 1
---

# User Guides

Everything you need to build, package, and distribute Gio applications with utm-dev.

## Getting Started

1. **[Quick Start](/users/quickstart/)** -- Install utm-dev and build your first app in 5 minutes
2. **[Platform Support](/users/platforms/)** -- What works on each platform, requirements, and known limitations

## Building and Distributing

3. **[Packaging](/users/packaging/)** -- The three-tier system: Build, Bundle, Package
4. **[Webviewer Shell](/users/webviewer-shell/)** -- Ship any website as a native desktop app with zero coding

## Command Reference

```bash
# Build for a platform
utm-dev build <platform> <app-directory>

# Build and run immediately
utm-dev run <platform> <app-directory>

# Create signed bundle for distribution
utm-dev bundle <platform> <app-directory>

# Package into archive (tar.gz / zip)
utm-dev package <platform> <app-directory>

# Generate platform icons from source image
utm-dev icons <app-directory>

# Install platform SDKs
utm-dev install <sdk-name>

# List available SDKs
utm-dev list

# Full help
utm-dev --help
```

**Supported platforms:** `macos`, `ios`, `ios-simulator`, `android`, `windows`
