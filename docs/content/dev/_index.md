---
title: "Developer Guides"
date: 2025-12-21
draft: false
weight: 2
---

# Developer Guides

Documentation for developers working on or contributing to utm-dev.

## Development Setup

```bash
# Clone and build
git clone https://github.com/joeblew999/utm-dev
cd utm-dev
go build .

# Run tests
go test ./...

# Use Task for common workflows
task --list
```

## Project Structure

```
utm-dev/
  cmd/                # CLI commands (Cobra-based)
  pkg/
    self/             # Self-management (build/install utm-dev itself)
    buildcache/       # Idempotent build cache
    config/           # SDK configuration and JSON definitions
    icons/            # Platform icon generation
    installer/        # SDK download and installation
    packaging/        # Bundle creation and code signing
    project/          # Project detection and structure
    utm/              # UTM virtual machine control
  examples/           # Working Gio example apps
    hybrid-dashboard/ # Gio UI + webview (start here)
    gio-plugin-webviewer/  # Multi-tab webview browser
    gio-basic/        # Simple Gio UI
    gio-plugin-hyperlink/  # Hyperlink plugin demo
  docs/               # This documentation (Hugo)
  .src/               # Cloned dependency source (gitignored)
```

**Key separation:** `pkg/self/` manages utm-dev itself (building, installing, upgrading the tool). Everything else manages the apps that utm-dev builds. Don't mix them.

## Guides

- **[CI/CD Integration](/dev/cicd/)** -- GitHub Actions workflows for automated builds
- **[AI Collaboration](/dev/agents/)** -- Reference guides for AI assistants working on this codebase
- **[Webview Analysis](/architecture/webview/)** -- Cross-platform webview deep dive

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run `go test ./...` and `go vet ./...`
5. Submit a pull request

### Conventions

- Standard Go conventions
- Cobra for CLI commands
- Idempotent operations (safe to run multiple times)
- Platform-specific code in separate files (`*_darwin.go`, `*_android.go`)
- All `self` commands output JSON (see `pkg/self/output/`)
- Use Taskfile for workflows (`task --list`)
