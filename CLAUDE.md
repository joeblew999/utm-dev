# Claude Instructions for utm-dev

## Philosphy

MUST use task file !!! SO we have consistency 

MUST use the .src folder as needed.

## Project Overview

**utm-dev** is a specialized build tool for creating **cross-platform hybrid applications** using Go and Gio UI.

### The Real Mission

Enable developers to build **one codebase** that runs everywhere:
- 🖥️ Desktop: macOS, Windows, Linux
- 📱 Mobile: iOS, Android  
- 🌐 Web: Browser (via WASM)

**Key capability**: Hybrid apps mixing **native Gio UI** (for shell/controls) with **native webviews** (for rich content).

### Why This Matters

Traditional cross-platform tools require multiple languages (Swift, Kotlin, JavaScript). **utm-dev enables pure Go development** for hybrid apps by:
1. Managing platform SDKs (Android SDK, Xcode tools)
2. Building platform-specific binaries from Go source
3. Handling native integrations (webviews, icons, packaging)
4. Supporting the full app lifecycle (build → package → release)

### Key Principles
- **Pure Go development**: One language for all platforms
- **Hybrid architecture**: Native UI + webview content
- **Idempotent operations**: All operations are safe to run multiple times
- **DRY (Don't Repeat Yourself)**: Centralized path management and shared utilities
- **Developer-focused**: Clean CLI interface with clear commands
- **True cross-platform**: Web, desktop, and mobile from single codebase

## CRITICAL: Two Separate Systems - Don't Mix Them!

**utm-dev has TWO completely separate concerns:**

### 1. The `self` System (Meta - Managing utm-dev Itself)

**Purpose**: Build, install, and manage **utm-dev the tool**

```
pkg/self/          # SELF-CONTAINED - manages utm-dev itself
├── architecture.go  # utm-dev's supported platforms
├── build.go        # Build utm-dev binaries
├── install.go      # Install utm-dev to system
├── scripts.go      # Bootstrap scripts for utm-dev
└── templates/      # Bootstrap script templates

cmd/self.go        # CLI commands for utm-dev management
```

**Commands**: `self build`, `self install`, `self setup`, `self upgrade`
**Builds**: utm-dev binaries (the tool itself)
**Uses**: Standard Go cross-compilation
**Output**: `utm-dev-darwin-arm64`, `utm-dev-linux-amd64`, etc.

### 2. The App Building System (What Users Use)

**Purpose**: Build **Gio applications** that users create

```
pkg/
├── buildcache/    # Cache for Gio app builds
├── config/        # User app configuration
├── icons/         # Generate icons for user apps
├── installer/     # Install SDKs for building user apps
├── packaging/     # Package user apps for distribution
├── project/       # Detect user app structure
└── constants/     # Build directories for user apps (.bin, .build, .dist)

cmd/
├── build.go       # Build user's Gio apps
├── icons.go       # Generate icons for user apps
├── package.go     # Package user apps
└── install.go     # Install SDKs for building user apps
```

**Commands**: `build macos examples/webviewer`, `icons myapp`, `package myapp`
**Builds**: User Gio applications
**Uses**: Gio, gogio, platform SDKs
**Output**: `examples/myapp/.bin/myapp.app`, `.dist/myapp.apk`, etc.

### Why This Separation Matters

**WRONG - Mixing the Systems:**
```go
// ❌ DON'T use pkg/self for user apps
func buildUserApp() {
    self.Build()  // NO! This builds utm-dev, not user apps
}

// ❌ DON'T use app build dirs for utm-dev
func buildSelf() {
    outputPath := constants.BinDir  // NO! That's for user apps
}

// ❌ DON'T use app config for utm-dev
func installSelf() {
    cfg := config.Load()  // NO! That's user app config
}
```

**RIGHT - Keeping Them Separate:**
```go
// ✅ Self system is isolated
func buildGoupUtil() {
    self.Build(self.BuildOptions{UseLocal: false})
    // Outputs to project root: utm-dev-darwin-arm64
}

// ✅ App building uses its own system
func buildUserApp() {
    project := project.Detect("examples/webviewer")
    builder.Build(project, "macos")
    // Outputs to examples/webviewer/.bin/
}
```

### Rules for Working with `self`

1. **pkg/self/ is SELF-CONTAINED** - No imports from other pkg/ directories (except utils)
2. **No cross-contamination** - App building code never calls pkg/self, self never calls app building
3. **Different outputs** - Self outputs to project root, apps output to `.bin/.build/.dist`
4. **Different configs** - Self uses hardcoded config, apps use user config files
5. **Different purposes** - Self is for developers OF utm-dev, rest is for developers USING utm-dev

### Easy Way to Remember

```
┌─────────────────────────────────────────┐
│  pkg/self/  →  Manages utm-dev        │
│  Everything else  →  Manages user apps  │
└─────────────────────────────────────────┘

If you're working on:
- Bootstrap scripts → pkg/self/
- Installing utm-dev → pkg/self/
- Building utm-dev → pkg/self/

- Building Gio apps → pkg/project, pkg/icons, cmd/build.go
- Installing SDKs → pkg/installer, cmd/install.go
- Packaging apps → pkg/packaging, cmd/package.go
```

**When in doubt**: Ask yourself "Is this for utm-dev itself or for the apps it builds?"

## JSON-Only Output (Self System)

**CRITICAL**: All `self` commands output **JSON ONLY** - no human-readable text by default!

### Why JSON-Only?

The self system uses JSON output to enable:
1. **Remote execution** - Parse output when controlling utm-dev on Windows VMs, Docker containers, SSH
2. **Automation** - CI/CD pipelines can easily parse results
3. **Consistent interface** - Same parsing code works locally and remotely
4. **Machine-readable** - No need to parse human-friendly text

### JSON Output Structure

Every self command outputs this consistent schema:

```json
{
  "command": "self version",
  "version": "1",
  "timestamp": "2025-10-22T12:34:56Z",
  "status": "ok",
  "exit_code": 0,
  "data": { ... },
  "error": null
}
```

**Fields:**
- `command` - The command that was run
- `version` - JSON schema version (currently "1")
- `timestamp` - ISO8601 UTC timestamp
- `status` - `"ok"`, `"warning"`, or `"error"`
- `exit_code` - Exit code (0 = success)
- `data` - Command-specific result data
- `error` - Error details (only present if status is "error")

### Self Commands JSON Output

**self version:**
```json
{
  "command": "self version",
  "status": "ok",
  "data": {
    "version": "dev",
    "os": "darwin",
    "arch": "arm64",
    "location": "/usr/local/bin/utm-dev"
  }
}
```

**self status:**
```json
{
  "command": "self status",
  "status": "ok",
  "data": {
    "installed": true,
    "current_version": "dev",
    "latest_version": "v1.0.1",
    "update_available": true,
    "location": "/usr/local/bin/utm-dev"
  }
}
```

**self doctor:**
```json
{
  "command": "self doctor",
  "status": "warning",
  "data": {
    "installations": [
      {"path": "/usr/local/bin/utm-dev", "active": true, "shadowed": false}
    ],
    "issues": ["Multiple utm-dev installations found"],
    "suggestions": ["Remove: /opt/bin/utm-dev"]
  }
}
```

**self build:**
```json
{
  "command": "self build",
  "status": "ok",
  "data": {
    "binaries": ["utm-dev-darwin-arm64", "utm-dev-linux-amd64"],
    "scripts_generated": true,
    "output_dir": "/path/to/utm-dev",
    "local_mode": false
  }
}
```

**self setup:**
```json
{
  "command": "self setup",
  "status": "ok",
  "data": {
    "installed": true,
    "location": "/usr/local/bin/utm-dev",
    "in_path": true,
    "dependencies_ok": true
  }
}
```

**self uninstall:**
```json
{
  "command": "self uninstall",
  "status": "ok",
  "data": {
    "removed": ["/usr/local/bin/utm-dev"],
    "failed": []
  }
}
```

**self test:**
```json
{
  "command": "self test",
  "status": "ok",
  "data": {
    "phase": "bootstrap_test",
    "passed": true,
    "steps": ["Building with --local flag", "Verifying scripts"],
    "errors": []
  }
}
```

**self upgrade:**
```json
{
  "command": "self upgrade",
  "status": "ok",
  "data": {
    "previous_version": "v1.0.0",
    "new_version": "v1.0.1",
    "downloaded": true,
    "installed": true,
    "location": "/usr/local/bin/utm-dev"
  }
}
```

**self release:**
```json
{
  "command": "self release",
  "status": "ok",
  "data": {
    "version": "v1.0.1",
    "tests_passed": true,
    "built": true,
    "tagged": true,
    "pushed": true,
    "binaries": ["utm-dev-darwin-arm64", "utm-dev-linux-amd64", "utm-dev-windows-amd64"]
  }
}
```

### Parsing JSON Output

**IMPORTANT**: The Data field is `json.RawMessage` which enables **bidirectional parsing** - you can both create and parse JSON with type safety!

**Go (Type-Safe Parsing):**
```go
import "github.com/joeblew999/utm-dev/pkg/self/output"

// Step 1: Parse BaseResult
var base output.BaseResult
json.Unmarshal([]byte(stdout), &base)

if base.Status != "ok" {
    log.Fatal(base.Error.Message)
}

// Step 2: Parse typed data using helper method
versionData, err := base.ParseVersionData()
if err != nil {
    log.Fatal(err)
}

// Step 3: Use typed data (autocompletion works!)
fmt.Printf("Version: %s\n", versionData.Version)
fmt.Printf("OS: %s\n", versionData.OS)
fmt.Printf("Arch: %s\n", versionData.Arch)
```

**Available Parse Methods:**
- `ParseVersionData()` → `*VersionResult`
- `ParseStatusData()` → `*StatusResult`
- `ParseDoctorData()` → `*DoctorResult`
- `ParseBuildData()` → `*BuildResult`
- `ParseSetupData()` → `*SetupResult`
- `ParseUninstallData()` → `*UninstallResult`
- `ParseTestData()` → `*TestResult`
- `ParseUpgradeData()` → `*UpgradeResult`
- `ParseReleaseData()` → `*ReleaseResult`

**PowerShell:**
```powershell
$result = ./utm-dev.exe self version | ConvertFrom-Json
if ($result.status -eq "ok") {
    Write-Host "Version: $($result.data.version)"
}
```

**Bash:**
```bash
result=$(./utm-dev self version)
status=$(echo "$result" | jq -r '.status')
if [ "$status" = "ok" ]; then
    version=$(echo "$result" | jq -r '.data.version')
    echo "Version: $version"
fi
```

**Python:**
```python
import json, subprocess

result = json.loads(subprocess.check_output(['./utm-dev', 'self', 'version']))
if result['status'] == 'ok':
    print(f"Version: {result['data']['version']}")
```

### Remote Execution Pattern

The JSON-only output enables a remote client pattern using the same types:

```go
// pkg/remote/client.go
type Client struct {
    Executor Executor // SSH, UTM, Docker, etc.
}

func (c *Client) SelfVersion() (*output.VersionResult, error) {
    stdout, err := c.Executor.Execute([]string{"utm-dev", "self", "version"})
    if err != nil {
        return nil, err
    }

    // Parse BaseResult
    var base output.BaseResult
    if err := json.Unmarshal([]byte(stdout), &base); err != nil {
        return nil, err
    }

    // Check status
    if base.Status != "ok" {
        return nil, fmt.Errorf("command failed: %s", base.Error.Message)
    }

    // Parse typed data using helper method
    return base.ParseVersionData()
}
```

**Same code works for:**
- Local execution: `Executor = LocalExecutor`
- SSH: `Executor = SSHExecutor{host: "windows-vm"}`
- UTM: `Executor = UTMExecutor{vm: "Win11"}`
- Docker: `Executor = DockerExecutor{container: "builder"}`

### Implementation Details

**Package**: `pkg/self/output/`
```
pkg/self/output/
├── result.go     # BaseResult, VersionResult, StatusResult, etc.
├── output.go     # Print(), PrintError(), PrintSuccess()
└── wrapper.go    # SafeExecute() with panic recovery
```

**Key types:**
- `BaseResult` - Universal JSON structure
- `Result` interface - All result types implement ToBaseResult()
- Typed results: `VersionResult`, `StatusResult`, `DoctorResult`, etc.

**Error handling:**
- Command errors output valid JSON with `status: "error"`
- Panics are caught and output JSON with stack trace
- Exit codes: 0 (success), 1 (error), 2 (panic)

**IMPORTANT - What outputs JSON:**
- ✅ **Command execution** → JSON (e.g., `self version`, `self doctor`)
- ✅ **Command errors** → JSON with error field
- ❌ **Help text** → Human-readable (e.g., `self --help`, `self version --help`)
- ❌ **Cobra errors** → Human-readable (e.g., invalid command)

**Commands with JSON output:**
- ✅ `self version` - Version information
- ✅ `self status` - Installation status
- ✅ `self doctor` - Validate installation
- ✅ `self build` - Build binaries
- ✅ `self setup` - Install utm-dev
- ✅ `self uninstall` - Remove utm-dev
- ✅ `self test` - Test bootstrap scripts
- ✅ `self upgrade` - Upgrade to latest release
- ✅ `self release` - Create and push release

**ALL self commands now output JSON!** This enables full remote automation including upgrading utm-dev inside Windows VMs.

### Testing JSON Output

JSON-enabled commands MUST output valid JSON:

```bash
# Test that output is valid JSON
go run . self version | jq -e '.command == "self version"'

# Test all commands
for cmd in version status doctor; do
    echo -n "Testing self $cmd... "
    go run . self $cmd 2>&1 | jq -e ".command == \"self $cmd\"" > /dev/null && echo "✓" || echo "✗"
done
```

### Remember

- **JSON ONLY** - No human-readable output in self system
- **Consistent schema** - All commands use BaseResult structure
- **Machine-first** - Designed for automation and remote execution
- **Self-contained** - Output package lives in `pkg/self/output/`

**User-facing commands** (build, icons, package) can still use human-friendly output!

## Project Structure

```
utm-dev/
├── cmd/                    # CLI commands (Cobra-based)
│   ├── root.go            # Root command
│   ├── build.go           # Build Gio apps for platforms
│   ├── install.go         # Install SDKs
│   ├── self.go            # Build/update utm-dev itself
│   ├── icons.go           # Generate platform icons
│   ├── package.go         # Package apps for distribution
│   ├── workspace.go       # Manage Go workspaces
│   └── ...
├── pkg/                   # Shared packages
│   ├── config/           # Configuration management
│   ├── installer/        # SDK installation logic
│   ├── icons/            # Icon generation
│   ├── workspace/        # Go workspace utilities
│   └── ...
├── examples/             # Example Gio applications
│   ├── gio-basic/                # Simple Gio UI demo
│   ├── gio-plugin-hyperlink/     # Hyperlink plugin demo
│   └── gio-plugin-webviewer/     # Multi-tab webview browser (THE KEY EXAMPLE)
├── docs/                 # End-user documentation
│   ├── agents/           # AI assistant collaboration guides
│   └── WEBVIEW-ANALYSIS.md  # Cross-platform webview deep dive
├── .src/                 # Dependency source code (gitignored)
│   └── gio-plugins/      # gio-plugins source for reference
└── main.go              # Entry point

```

## The Hybrid App Vision

**utm-dev exists to make this possible**:

```
┌─────────────────────────────────────┐
│     Your App (Pure Go)              │
│                                     │
│  ┌─────────────────────────────┐   │
│  │  Gio UI (Native Controls)   │   │
│  │  - Tabs, buttons, layout    │   │
│  │  - Native performance       │   │
│  └─────────────────────────────┘   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │  Native WebView             │   │
│  │  - Rich web content         │   │
│  │  - HTML/CSS/JavaScript      │   │
│  │  - Platform webview engine  │   │
│  └─────────────────────────────┘   │
│                                     │
│  ↕ Go ↔ JavaScript Bridge          │
└─────────────────────────────────────┘

Built once → Runs on all platforms
```

## CRITICAL: Gio Version Compatibility

**VERSION MANAGEMENT IS CRITICAL** - Gio and gio-plugins version mismatches cause runtime panics!

### The Problem

- Using mismatched Gio and gio-plugins versions causes: `panic: Gio version not supported`
- The version tags don't guarantee compatibility - specific commit hashes may be required
- See issue: https://github.com/gioui-plugins/gio-plugins/issues/104

### The Solution

**Always use these specific versions** (as of 2025-12-20):

```bash
# For projects using gio-plugins (webviewer, hyperlink, etc.)
go get gioui.org@7bcb315ee174
go get github.com/gioui-plugins/gio-plugins@v0.9.1
go mod tidy

# For projects using only Gio UI (no plugins)
go get gioui.org@7bcb315ee174
go mod tidy
```

This gives you:
- `gioui.org v0.9.1-0.20251215212054-7bcb315ee174` (latest compatible with gio-plugins)
- `github.com/gioui-plugins/gio-plugins v0.9.1` (official release tag)

### What's New in v0.9.1

**Gio UI:**
- ✨ **Custom URI Scheme Support** - Launch apps via `gio://some/data` on all platforms
- 🖱️ **Touch Screen Support on Windows** - Windows Pointer API for touch detection
- 🔧 Fixed text rendering on some Android devices
- 🔧 Fixed GPU clipping causing 1px overlaps
- 🔧 macOS fullscreen now respects MaxSize

**gio-plugins:**
- 🔧 Auth global event listener fix (#106)
- 📦 Updated to Gio v0.9.1 (#105)
- 🌿 New `deeplink2025` branch for deep linking work

### When Adding New Examples

1. **ALWAYS** use the version commands above after `go mod init`
2. **NEVER** use `@latest` tags - they are incompatible
3. **TEST** the app actually launches before committing
4. **UPDATE** this section if recommended versions change

### Version Management TODO

utm-dev should eventually automate this:
- `utm-dev init <project>` - Initialize with correct versions
- `utm-dev doctor` - Check and fix version compatibility
- Auto-update go.mod when building examples

## Development Workflow

### Self System Commands (Managing utm-dev)

The `self` commands manage utm-dev itself. These are organized into three categories:

#### Information Commands (What's installed?)

```bash
# Check installed version
go run . self version
# Output:
#   utm-dev version v1.2.3
#   OS: darwin
#   Arch: arm64
#   Location: /usr/local/bin/utm-dev

# Check status and updates
go run . self status
# Output:
#   ✅ utm-dev is installed
#   📦 Update available: v1.2.3 → v1.2.4

# Validate installation
go run . self doctor
# Output:
#   ✅ utm-dev: installed
#   ✅ Homebrew: installed
#   ✅ git: installed
#   ✅ go: installed
#   ✅ task: installed
```

#### Installation Commands (Get it working)

```bash
# Full setup (dependencies + binary)
go run . self setup

# Upgrade to latest release
go run . self upgrade

# Remove from system
go run . self uninstall
```

#### Development Commands (For utm-dev developers)

```bash
# Build all platform binaries (for release)
go run . self build

# Build with local testing mode
go run . self build --local

# Test bootstrap scripts locally
go run . self test

# Release workflow
go run . self release patch   # or minor, major, v1.2.3
```

### Building Hybrid Apps (What users do)

**These commands are separate from self management!**

```bash
# Run tests
go test ./...
```

### Building Hybrid Apps

```bash
# The webviewer example is THE reference implementation
go run . build macos examples/gio-plugin-webviewer
go run . build windows examples/gio-plugin-webviewer
go run . build android examples/gio-plugin-webviewer
go run . build ios examples/gio-plugin-webviewer

# Install required SDKs
go run . install android-sdk
go run . install android-ndk

# Generate platform icons
go run . icons examples/gio-plugin-webviewer
```

## Idempotency Guarantees

**ALL build operations are idempotent** - Safe to run multiple times, skips unnecessary work.

### Build Cache System

Located in `pkg/buildcache/`, tracks:
- **Source hashes** (SHA256 of .go, .mod, .sum files)
- **Output paths** and timestamps
- **Build success** status
- **Platform-specific** caching

### Smart Rebuild Detection

```bash
# First build - compiles everything
go run . build macos examples/hybrid-dashboard
# Building hybrid-dashboard for macos...
# ✓ Built successfully

# Second build - skips (no changes)
go run . build macos examples/hybrid-dashboard
# ✓ hybrid-dashboard for macos is up-to-date (use --force to rebuild)

# Force rebuild
go run . build --force macos examples/hybrid-dashboard
# Rebuilding: forced rebuild requested

# Check if rebuild needed (for CI/CD)
go run . build --check macos examples/hybrid-dashboard
echo $?  # 0=up-to-date, 1=needs rebuild
```

### What Triggers Rebuilds

✅ **Triggers rebuild:**
- Source code changes (.go files)
- Dependencies change (go.mod, go.sum)
- Assets change (.png, .jpg for icons)
- Output missing or corrupted
- `--force` flag

❌ **Skips rebuild:**
- No source changes
- Output exists and valid
- Previous build successful
- Same platform

### Build Flags

All build commands support:
- `--force` - Force rebuild even if up-to-date
- `--check` - Check if rebuild needed (exit code 0=no, 1=yes)

## Three-Tier Packaging System

utm-dev provides **three distinct operations** for the app lifecycle:

### 1. Build - Compile & Create Basic Structures

```bash
go run . build <platform> <app>
```

**Purpose:** Fast iteration during development
- Compiles Go source to binaries
- Creates basic app structures (.app bundles, APKs)
- **Idempotent**: Uses build cache
- **Fast**: Skips unnecessary rebuilds
- Output: `<app>/.bin/`

### 2. Bundle - Create Signed App Bundles

```bash
go run . bundle <platform> <app> [--bundle-id ID] [--sign IDENTITY]
```

**Purpose:** Prepare for distribution
- Creates proper app bundles with metadata
- Generates Info.plist from templates
- **Code signing** (auto-detect or specified)
- Hardened runtime entitlements (macOS)
- **Pure Go**: No bash scripts
- Output: `<app>/.dist/<name>.app`

```bash
# Examples
go run . bundle macos examples/hybrid-dashboard
go run . bundle macos examples/hybrid-dashboard --bundle-id com.company.app
go run . bundle macos examples/hybrid-dashboard --sign "Developer ID Application: Name"
```

**Code Signing:**
- Auto-detects "Developer ID Application" certificates
- Falls back to "Apple Development" if needed
- Uses ad-hoc signature (`-`) if no certificates found
- Ad-hoc suitable for local testing, not public distribution

### 3. Package - Create Distribution Archives

```bash
go run . package <platform> <app>
```

**Purpose:** Final distribution packages
- Creates tar.gz (macOS/iOS) or zip (Windows) archives
- Copies APKs (Android)
- Ready for upload/distribution
- **Pure Go**: Uses pkg/packaging/archive.go
- Output: `<app>/.dist/<name>-<platform>.tar.gz`

### Complete Release Workflow

```bash
# 1. Build (idempotent)
go run . build macos examples/hybrid-dashboard

# 2. Create signed bundle
go run . bundle macos examples/hybrid-dashboard \
  --bundle-id com.company.myapp \
  --version 1.0.0

# 3. Test the bundle
open examples/hybrid-dashboard/.dist/hybrid-dashboard.app

# 4. Package for distribution
go run . package macos examples/hybrid-dashboard

# 5. Upload the archive
ls examples/hybrid-dashboard/.dist/*.tar.gz
```

See [docs/PACKAGING.md](docs/PACKAGING.md) for complete details.

## Key Commands to Understand

- `build <platform> <app>` - Build Gio apps for different platforms (macos, windows, android, ios)
- `install <sdk>` - Install platform SDKs (Android SDK, NDK, etc.)
- `self build` - Build utm-dev binaries for distribution
- `icons <app>` - Generate platform-specific icons from source images
- `package <app>` - Package built apps for distribution
- `workspace` - Manage Go workspace files for multi-module projects
- `gitignore` - Manage .gitignore templates for Gio projects

## Important Files

- `cmd/*.go` - All CLI command implementations
- `pkg/config/` - Config file handling and directory management
- `pkg/installer/` - SDK installation logic
- `examples/gio-plugin-webviewer/main.go` - **THE KEY EXAMPLE** - Multi-tab browser showing full webview capabilities
- `go.mod` - Dependencies (cobra, progressbar, icns, gio-plugins, etc.)
- `.gitignore` - Build binaries are excluded (utm-dev*)

## Common Tasks

### Understanding Webview Integration

**This is the core use case**. Study these files:

1. **Local example**: `examples/gio-plugin-webviewer/main.go`
2. **Plugin source**: `.src/gio-plugins/webviewer/`
3. **Demo app**: `.src/gio-plugins/webviewer/demo/demo.go`
4. **Analysis**: `docs/WEBVIEW-ANALYSIS.md`
5. **Agent guide**: `docs/agents/gio-plugins.md`

### Adding a New Command

1. Create `cmd/newcommand.go`
2. Use Cobra patterns from existing commands
3. Add to root command in `cmd/root.go`
4. Test with `go run . newcommand`

### Modifying Build Logic

- See `cmd/build.go` for platform-specific build commands
- Build logic uses idempotent patterns
- Platform detection and SDK path resolution in `pkg/`

### Working with Icons

- Icon generation in `pkg/icons/`
- Supports PNG source → platform formats (icns, ico, Android drawables)
- Test with example projects

## Dependencies

### Core Dependencies
- `github.com/spf13/cobra` - CLI framework
- `github.com/schollz/progressbar/v3` - Progress display
- `github.com/JackMordaunt/icns` - macOS icon generation

### Gio Ecosystem (THE IMPORTANT ONES)
- `gioui.org` - Core Gio UI framework
- `github.com/gioui-plugins/gio-plugins` - Native plugins
  - **webviewer** - Native webview integration (WKWebView, WebView2, etc.)
  - **hyperlink** - Open URLs in system browser
  - **auth** - OAuth flows
  - **explorer** - File system access

### Platform Tools
- Android SDK, NDK - For Android builds
- Xcode - For iOS/macOS builds  
- WebView2 - For Windows (Edge-based webview)

## Code Obfuscation with Garble

**CRITICAL**: utm-dev uses [garble](https://github.com/burrowers/garble) for code obfuscation to protect:
1. **The tool itself** - When building utm-dev binaries (`self build`)
2. **User applications** - When building Gio apps for distribution

### Why Garble?

Garble obfuscates Go code by:
- Renaming exported and unexported symbols
- Scrambling string literals
- Removing debug information
- Making reverse engineering difficult

**Important**: Constants are NOT obfuscated, string literals ARE obfuscated.

### Configuration Constants Pattern

To ensure garble compatibility, we use **constants instead of string literals** for critical values:

**pkg/self/config.go** - Configuration for the self system:
```go
package self

// Repository configuration
const (
    GitHubOwner  = "joeblew999"
    GitHubRepo   = "utm-dev"
    FullRepoName = GitHubOwner + "/" + GitHubRepo
)

// GitHub URLs
const (
    GitHubAPIBase = "https://api.github.com"
    GitHubBase    = "https://github.com"
)

// Binary configuration
const (
    BinaryName = "utm-dev"
)

// Installation paths
const (
    UnixInstallDir  = "/usr/local/bin"
    UnixInstallPath = UnixInstallDir + "/" + BinaryName
)

// Helper functions
func GetInstallPath() string { ... }
func GetLatestReleaseURL() string { ... }
func GetRepoGitURL() string { ... }
```

**pkg/self/output/config.go** - JSON schema configuration:
```go
package output

// JSON schema version
const JSONSchemaVersion = "1"

// Status values
const (
    StatusOK      = "ok"
    StatusWarning = "warning"
    StatusError   = "error"
)

// Error types
const (
    ErrorTypeExecution = "execution_error"
    ErrorTypePanic     = "panic"
)

// Exit codes
const (
    ExitSuccess = 0
    ExitError   = 1
    ExitPanic   = 2
)
```

### Garble Installation

Garble is installed automatically during `self setup` or can be installed separately:

```bash
# Install garble
go install mvdan.cc/garble@v0.15.0

# Verify installation
garble version
```

**Current supported version**: v0.15.0
**Download**: https://github.com/burrowers/garble/releases/tag/v0.15.0

### Build Integration

#### Building utm-dev with Garble

When building utm-dev itself:

```bash
# Without garble (development)
go run . self build

# With garble (production/release)
go run . self build --obfuscate
```

The `self build` command automatically uses garble when:
1. `--obfuscate` flag is provided
2. Building for release (via `self release`)
3. Garble is installed and in PATH

#### Building Gio Apps with Garble

When building user applications:

```bash
# Without garble (development)
go run . build macos examples/gio-plugin-webviewer

# With garble (production)
go run . build macos examples/gio-plugin-webviewer --obfuscate
```

### What Gets Obfuscated

**Obfuscated:**
- ✅ Function names (unexported)
- ✅ Variable names (unexported)
- ✅ Type names (unexported)
- ✅ String literals (inline strings)
- ✅ Stack traces
- ✅ Build paths

**NOT Obfuscated:**
- ❌ Constants (const declarations)
- ❌ Exported names (Go's capitalized names)
- ❌ Standard library
- ❌ Dependencies (unless also obfuscated)

### String Literals vs Constants

**WRONG (gets obfuscated, breaks functionality):**
```go
// ❌ String literal - garble will scramble this
releaseURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)

// ❌ If garble obfuscates the URL, GitHub API calls break!
```

**CORRECT (constants are preserved):**
```go
// ✅ Constant - garble leaves this alone
releaseURL := fmt.Sprintf("%s/repos/%s/releases/latest", GitHubAPIBase, FullRepoName)

// ✅ URLs remain functional after obfuscation
```

### Testing Obfuscated Builds

Always test obfuscated builds before release:

```bash
# Build with obfuscation
go run . self build --obfuscate

# Test the obfuscated binary
./utm-dev-darwin-arm64 self version  # Should output JSON
./utm-dev-darwin-arm64 self doctor   # Should work normally

# Test upgrade works (critical!)
./utm-dev-darwin-arm64 self upgrade
```

### Common Garble Issues

**Issue 1: API calls failing**
- **Cause**: URL string literals got obfuscated
- **Fix**: Move URLs to constants in `pkg/self/config.go`

**Issue 2: JSON parsing broken**
- **Cause**: JSON field tags or schema version obfuscated
- **Fix**: Use `JSONSchemaVersion` constant, not string literal

**Issue 3: Path resolution failing**
- **Cause**: Hardcoded paths as string literals
- **Fix**: Use `GetInstallPath()` or constants from config.go

**Issue 4: Garble not found**
- **Cause**: Garble not installed
- **Fix**: Run `go install mvdan.cc/garble@v0.15.0`

### Garble and Remote Execution

When using utm-dev on remote machines (Windows VMs, Docker, SSH):

1. **Install garble on the remote** - Required for building obfuscated binaries
2. **JSON output works fine** - JSON parsing is not affected by obfuscation
3. **Constants ensure compatibility** - Remote utm-dev can still make GitHub API calls
4. **Upgrade still works** - Remote upgrade downloads and installs correctly

### Development Workflow

**During development (no obfuscation needed):**
```bash
go run . self build              # Fast, readable stack traces
go run . build macos examples/webviewer
```

**Before release (obfuscate for security):**
```bash
go run . self release            # Automatically uses garble
go run . self test               # Test obfuscated binaries work
```

**Testing specific features:**
```bash
# Test upgrade with obfuscation
go run . self build --obfuscate
./utm-dev-darwin-arm64 self upgrade

# Test JSON output still valid
./utm-dev-darwin-arm64 self version | jq
```

### CI/CD Integration

GitHub Actions automatically uses garble for releases:

```yaml
# .github/workflows/release.yml
- name: Build release binaries
  run: go run . self release   # Uses garble automatically

- name: Test obfuscated binaries
  run: |
    ./utm-dev-darwin-arm64 self version
    ./utm-dev-linux-amd64 self version
```

### Future: Garble for Gio Apps

When packaging Gio apps for distribution:

```bash
# Package with obfuscation
go run . package --obfuscate examples/gio-plugin-webviewer

# Produces obfuscated binaries:
# - Harder to reverse engineer
# - Protects your code
# - Still fully functional
```

### Reference

- **Garble GitHub**: https://github.com/burrowers/garble
- **Current version**: v0.15.0
- **Releases**: https://github.com/burrowers/garble/releases
- **Documentation**: https://github.com/burrowers/garble#readme
## SDK System Architecture

**IMPORTANT**: utm-dev has a sophisticated SDK management system for installing platform tools.

### How SDKs Work

SDKs are defined in JSON files and installed to platform-specific directories:

```
macOS:    ~/utm-dev-sdks/
Linux:    ~/.local/share/utm-dev/sdks/
Windows:  %LOCALAPPDATA%\utm-dev\sdks\
```

### SDK Definition Files

Located in `pkg/config/*.json`:

- **sdk-android-list.json** - Android SDK, NDK, build-tools, platform-tools
- **sdk-ios-list.json** - Xcode command line tools (manual install)
- **sdk-build-tools.json** - Build tools like garble (NEW)

### SDK JSON Structure

```json
{
  "sdks": {
    "sdk-name": [
      {
        "version": "1.0.0",
        "goupName": "sdk-name",
        "installPath": "category/sdk-name",
        "downloadUrl": "https://example.com/sdk.zip",
        "checksum": "sha256:abc123...",
        "platforms": {
          "darwin/amd64": {
            "downloadUrl": "https://example.com/darwin-amd64.tar.gz",
            "checksum": "sha256:def456..."
          },
          "darwin/arm64": {
            "downloadUrl": "https://example.com/darwin-arm64.tar.gz",
            "checksum": "sha256:ghi789..."
          }
        }
      }
    ]
  }
}
```

### SDK Types

**1. Direct Download SDKs** (most common)
- Downloads archive from URL
- Extracts to `installPath` under SDK directory
- Verifies checksum
- Example: Android NDK, command-line tools

**2. Platform-Specific SDKs**
- Different download URL per OS/arch
- Uses `platforms` map in JSON
- Example: Pre-built binaries for different architectures

**3. SDK Manager SDKs** (Android-specific)
- Uses Android sdkmanager to install
- Requires cmdline-tools and openjdk-17
- Example: build-tools, platform-tools, emulator
- Specified via `sdkmanagerName` field

**4. Go Install SDKs** (special case)
- Installed via `go install` command
- Goes to $GOPATH/bin or $GOBIN
- Example: garble
- Handled in code, not JSON

**5. Manual Install SDKs**
- No downloadUrl provided
- User must install manually
- Example: Xcode (too large, requires App Store)

### Adding a New SDK

**Step 1: Choose the right JSON file**
- Android tools → `sdk-android-list.json`
- iOS tools → `sdk-ios-list.json`
- Build tools → `sdk-build-tools.json`

**Step 2: Add SDK definition**
```json
{
  "version": "2.0.0",
  "goupName": "my-tool",
  "installPath": "tools/my-tool",
  "platforms": {
    "darwin/amd64": {
      "downloadUrl": "https://github.com/org/tool/releases/download/v2.0.0/tool-darwin-amd64.tar.gz",
      "checksum": "sha256:YOUR_CHECKSUM_HERE"
    },
    "darwin/arm64": {
      "downloadUrl": "https://github.com/org/tool/releases/download/v2.0.0/tool-darwin-arm64.tar.gz",
      "checksum": "sha256:YOUR_CHECKSUM_HERE"
    },
    "linux/amd64": {
      "downloadUrl": "https://github.com/org/tool/releases/download/v2.0.0/tool-linux-amd64.tar.gz",
      "checksum": "sha256:YOUR_CHECKSUM_HERE"
    },
    "windows/amd64": {
      "downloadUrl": "https://github.com/org/tool/releases/download/v2.0.0/tool-windows-amd64.zip",
      "checksum": "sha256:YOUR_CHECKSUM_HERE"
    }
  }
}
```

**Step 3: Get checksums**
```bash
# Download and calculate checksum
curl -L https://github.com/org/tool/releases/download/v2.0.0/tool-darwin-arm64.tar.gz | sha256sum
```

**Step 4: Test installation**
```bash
go run . install my-tool
```

### Special Case: Go Install SDKs (like garble)

For Go-based tools installed via `go install`:

**Step 1: Create installer function** in `pkg/installer/`:
```go
// pkg/installer/toolname.go
package installer

const (
	ToolVersion = "v1.0.0"
	ToolPackage = "github.com/org/tool"
)

func InstallTool() error {
	cmd := exec.Command("go", "install", ToolPackage+"@"+ToolVersion)
	// ... installation logic
}

func IsToolInstalled() bool {
	_, err := exec.LookPath("tool")
	return err == nil
}
```

**Step 2: Add to install command** in `cmd/install.go`:
```go
func installSdk(sdkName string, cache *installer.Cache) error {
	// Special case for go-install tools
	if sdkName == "toolname" {
		return installer.InstallTool()
	}
	// ... rest of function
}
```

### SDK Installation Flow

```
User runs: go run . install android-ndk

1. cmd/install.go → installSdk()
2. utils.FindSDKItem() → searches all *.json files
3. Finds SDK definition in sdk-android-list.json
4. installer.Install() → downloads, extracts, verifies
5. Cache entry created in ~/.cache/utm-dev/cache.json
6. Binary/tools available for use
```

### SDK Cache System

SDKs are tracked in `~/.cache/utm-dev/cache.json`:

```json
{
  "entries": {
    "android-ndk-r26b": {
      "name": "android-ndk-r26b",
      "version": "r26b",
      "installPath": "sdks/ndk/26.1.10909125",
      "checksum": "sha256:...",
      "installedAt": "2025-10-22T10:30:00Z"
    }
  }
}
```

This prevents re-downloading if already installed.

### Updating SDK Versions

**CRITICAL**: When updating SDK versions, you MUST:

1. **Update the JSON file** - Change version, URLs, checksums
2. **Test on all platforms** - Download and verify checksums
3. **Update CLAUDE.md** - Document version change
4. **Test installation** - Run `go run . install sdk-name`
5. **Test builds** - Ensure builds still work with new version

### SDK Version Strategy

**Android SDKs:**
- Android SDK: Use latest stable API level
- NDK: Use r26b+ (supports M1/ARM64 macOS)
- Build tools: Match target API level
- Command-line tools: Use latest stable

**Build Tools:**
- garble: v0.15.0 (supports Go 1.25)
- Keep versions updated as Go updates

**iOS/macOS:**
- Xcode: User installs via App Store (we don't manage)
- Command line tools: Manual via xcode-select

### Verifying SDK Installations

```bash
# List installed SDKs
go run . list

# Check specific SDK
ls ~/utm-dev-sdks/

# View cache
cat ~/.cache/utm-dev/cache.json

# Doctor command checks all dependencies
go run . self doctor
```

### Common SDK Issues

**Issue 1: Checksum mismatch**
- **Cause**: Downloaded file corrupted or wrong version
- **Fix**: Delete cache entry, re-download

**Issue 2: SDK not found**
- **Cause**: JSON file not loaded or typo in goupName
- **Fix**: Check filename ends with `.json` in `pkg/config/`

**Issue 3: Platform not supported**
- **Cause**: Missing platform entry in JSON
- **Fix**: Add platform-specific URL and checksum

**Issue 4: Go install fails**
- **Cause**: Go not installed or GOPATH not set
- **Fix**: Install Go first, ensure GOPATH/bin in PATH

### SDK Maintenance Checklist

When maintaining SDK definitions:

- [ ] Check for new versions monthly
- [ ] Test downloads on all platforms (macOS, Linux, Windows)
- [ ] Verify checksums match
- [ ] Update CLAUDE.md with version changes
- [ ] Test actual builds with new SDK versions
- [ ] Update cache if installPath changes
- [ ] Document breaking changes

### Future SDK Enhancements

Planned improvements:

1. **Auto-update checker** - Notify when SDKs are outdated
2. **Multi-version support** - Allow multiple SDK versions side-by-side
3. **Dependency resolution** - Auto-install prerequisites
4. **Cleanup command** - Remove old SDK versions
5. **Mirror support** - Fallback download URLs for reliability

### SDK File Locations Reference

```
# SDK Definitions
pkg/config/sdk-android-list.json
pkg/config/sdk-ios-list.json
pkg/config/sdk-build-tools.json

# Installation Code
pkg/installer/installer.go          # Main SDK installer
pkg/installer/garble.go             # Garble (go install)
cmd/install.go                      # Install command

# Utilities
pkg/utils/utils.go                  # FindSDKItem()
pkg/config/config.go                # GetSDKDir()

# Cache
~/.cache/utm-dev/cache.json       # Tracks installed SDKs

# Installed SDKs
~/utm-dev-sdks/                   # macOS
~/.local/share/utm-dev/sdks/      # Linux
%LOCALAPPDATA%\utm-dev\sdks\      # Windows
```

### Adding Verification Tools

**redress** - Go binary analysis tool for verifying obfuscation:

```json
// In sdk-build-tools.json
{
  "version": "1.2.41",
  "goupName": "redress",
  "installPath": "tools/redress",
  "platforms": {
    "darwin/arm64": {
      "downloadUrl": "https://github.com/goretk/redress/releases/download/v1.2.41/redress_1.2.41_macOS_arm64.tar.gz",
      "checksum": "sha256:TO_BE_CALCULATED"
    }
  }
}
```

**Usage:**
```bash
# Install redress
go run . install redress

# Analyze obfuscated binary
redress utm-dev-darwin-arm64

# Should show minimal symbol information if properly obfuscated
```

This helps verify that garble obfuscation is working correctly.

## Testing Guidelines

- Test commands using `go run .` before building
- **Use the webviewer example for integration testing**
- Verify idempotency (running twice should produce same result)
- Test on target platforms when modifying build logic
- Check webview behavior on each platform (they differ!)

## CI/CD

- GitHub Actions in `.github/workflows/`
- `build.yml` - Main CI pipeline using `go run . self build`
- Builds binaries for all platforms
- Artifacts uploaded as releases

## Future Plans (See TODO.md)

- **UTM integration** - Automated Windows VM testing from macOS
- **Winget** - Windows package management for dependencies
- **Automated testing infrastructure** - Test builds on all platforms
- **JavaScript ↔ Go bridge patterns** - Better hybrid app communication
- **Production templates** - Ready-to-use hybrid app starters

## Source Code References (.src/)

The `.src/` folder contains cloned source code of key dependencies for easy reference. This folder is gitignored and local-only.

### Available Sources

- **gio-plugins** (`.src/gio-plugins/`) - Gio UI native plugins
  - WebViewer implementation and examples
  - Platform-specific code for macOS, iOS, Android, Windows, Linux
  - See [docs/agents/gio-plugins.md](docs/agents/gio-plugins.md) for detailed guide

- **robotgo** (`.src/robotgo/`) - Desktop automation and screenshots
  - Screenshot capabilities (CaptureScreen, CaptureImg, SaveCapture)
  - Multi-display support, keyboard/mouse automation
  - Platform-specific C code for macOS, Windows, Linux
  - See [docs/agents/robotgo.md](docs/agents/robotgo.md) for detailed guide
  - **Note**: Used optionally via build tags to avoid CGO in main build

### Usage

When working with dependencies:

```bash
# Clone a new dependency for reference
git clone --depth 1 https://github.com/org/repo .src/repo

# Search for implementations
grep -r "pattern" .src/gio-plugins/

# View platform-specific code
ls .src/gio-plugins/webviewer/webview/webview_*.go

# Read the webview demo (our example is based on this)
cat .src/gio-plugins/webviewer/demo/demo.go
```

### Agent Collaboration

For AI assistants working on this project:

1. **Read source before asking** - Check `.src/` for dependency behavior
2. **Update agent docs** - Add guides in `docs/agents/` when learning new patterns
3. **See agent guides** - Read `docs/agents/README.md` for collaboration patterns

The agent documentation helps multiple AI assistants work effectively on the codebase by providing context about dependencies, patterns, and architecture.

## Tips for Claude

1. **The webviewer example is CRITICAL** - This shows the real use case
2. **Always test with `go run .`** - Don't build binaries during development
3. **Maintain idempotency** - Operations should be safe to run multiple times
4. **Follow existing patterns** - Look at similar commands for consistency
5. **Update docs/** - Keep end-user docs in sync with code changes
6. **Check .gitignore** - Don't commit build binaries (utm-dev*)
7. **Use examples/** - Test changes with the example Gio projects
8. **Cross-platform awareness** - Code runs on macOS, Linux, Windows, Android, iOS
9. **Hybrid apps are the goal** - Native UI + webview content in pure Go
10. **Read .src/ dependencies** - Source code is available locally

## Common Debugging

```bash
# Check configuration
go run . config

# List available SDKs
go run . list

# Verbose output (add -v flag if available)
go run . build macos examples/gio-basic -v

# Check Go workspace
go run . workspace list

# Test webviewer on desktop (fastest iteration)
go run . build macos examples/gio-plugin-webviewer
open examples/gio-plugin-webviewer/.bin/macos/gio-plugin-webviewer.app
```

## Code Style

- Follow standard Go conventions
- Use `cobra` for CLI structure
- Error handling with clear messages
- Progress bars for long operations
- Idempotent file operations
- Platform-specific code in separate files (`*_darwin.go`, `*_android.go`, etc.)

## The Big Picture

**utm-dev is a developer tool for building a specific class of apps:**

**Cross-platform hybrid applications where:**
- Shell/controls are native Gio UI (Go)
- Content can be web (via native webviews)
- Everything is written in Go
- Deploys to web, desktop, and mobile from one codebase

This is about **enabling pure Go development** for the kind of apps that traditionally require Swift + Kotlin + JavaScript. The webview integration is what makes hybrid apps possible while keeping native performance.

## Screenshots

**System**: robotgo-based screenshot capture with App Store presets.

**Usage**:
```bash
# Capture example app screenshots
task screenshot-hybrid                # Single app
task screenshot-appstore-all          # All App Store sizes

# Direct command
CGO_ENABLED=1 go run -tags screenshot . run-and-capture examples/hybrid-dashboard output.png
```

**macOS Setup**: Grant Screen Recording permission in System Settings → Privacy & Security.

**Files**:
- `pkg/screenshot/` - robotgo integration
- `cmd/runandcapture.go` - Automated capture workflow
- Screenshots are gitignored, manually commit finals with `git add -f`

## Taskfile Maintenance

**CRITICAL**: When adding new commands, examples, or features, **ALWAYS update Taskfile.yml**!

### Why This Matters

The Taskfile is the **primary developer interface**. Users run `task --list` to discover what they can do. If you add a feature but don't add a task for it, **nobody will know it exists**.

### When to Update Taskfile

**Add a new task whenever you:**
- ✅ Create a new example app (`examples/new-app/`)
- ✅ Add a new command to `cmd/`
- ✅ Add a new platform target
- ✅ Create a new workflow or common operation
- ✅ Add testing or deployment capabilities

### Task Naming Convention

Follow this pattern:
```yaml
# Format: <action>:<target>:<platform>
task build:hybrid:macos        # Build hybrid-dashboard for macOS
task build:hybrid:ios          # Build hybrid-dashboard for iOS
task run:hybrid                # Build and run hybrid-dashboard
task build:examples:android    # Build all examples for Android
```

**Categories:**
- `run:*` - Build and launch (for quick testing)
- `build:*` - Build only
- `install:*` - Install SDKs/dependencies
- `test:*` - Run tests
- `clean:*` - Clean up artifacts
- `workspace:*` - Workspace management
- `setup` - One-time setup tasks
- `demo` - Quick demonstrations

### Example: Adding a New Example App

When you create `examples/new-app/`:

```yaml
# Add these tasks to Taskfile.yml

vars:
  NEW_APP_EXAMPLE: examples/new-app

tasks:
  run:new-app:
    desc: Build and run new-app example (macOS)
    cmds:
      - "{{.GOUP}} build macos {{.NEW_APP_EXAMPLE}}"
      - open {{.NEW_APP_EXAMPLE}}/.bin/new-app.app

  build:new-app:macos:
    desc: Build new-app for macOS
    cmds:
      - "{{.GOUP}} build macos {{.NEW_APP_EXAMPLE}}"

  build:new-app:ios:
    desc: Build new-app for iOS
    cmds:
      - "{{.GOUP}} build ios {{.NEW_APP_EXAMPLE}}"

  build:new-app:android:
    desc: Build new-app for Android
    cmds:
      - "{{.GOUP}} build android {{.NEW_APP_EXAMPLE}}"
```

**Then update the composite tasks:**
```yaml
  build:examples:macos:
    desc: Build all examples for macOS
    cmds:
      - "{{.GOUP}} build macos {{.BASIC_EXAMPLE}}"
      - "{{.GOUP}} build macos {{.WEBVIEWER_EXAMPLE}}"
      - "{{.GOUP}} build macos {{.HYBRID_EXAMPLE}}"
      - "{{.GOUP}} build macos {{.NEW_APP_EXAMPLE}}"  # ADD THIS
```

### Testing Your Tasks

Before committing, **always test**:
```bash
# Verify task syntax
task --list

# Test the new task
task run:new-app

# Test composite tasks still work
task build:examples:macos
```

### Taskfile Anti-Patterns

**DON'T:**
- ❌ Add features without corresponding tasks
- ❌ Use inconsistent naming
- ❌ Forget to update composite tasks (build:examples:all, etc.)
- ❌ Hardcode paths (use vars instead)
- ❌ Create duplicate tasks

**DO:**
- ✅ Keep tasks simple and composable
- ✅ Use descriptive names
- ✅ Add helpful descriptions
- ✅ Test before committing
- ✅ Update README if adding major workflows

### Quick Reference

```bash
# See all tasks
task --list

# Run a task
task demo

# Run with verbose output
task -v demo

# See what a task will do (dry run)
task --dry demo
```

### Remember

**The Taskfile is the front door.** Keep it updated, or features will be invisible to users.

**Golden Rule**: If you can do it with `go run .`, there should be a task for it.

## Testing Taskfile Targets

**IMPORTANT**: The Taskfile is our PRIMARY testing mechanism right now. Always verify tasks work!

### Why Test Tasks?

Currently, utm-dev has **limited unit tests**. The Taskfile tasks serve as:
- ✅ Integration tests (build → run workflows)
- ✅ Smoke tests (does it build at all?)
- ✅ Platform compatibility tests
- ✅ User workflow validation

**If a task is broken, users can't use the tool!**

### Test Before Committing

**Always run these before pushing**:

```bash
# 1. Verify all tasks are listed
task --list

# 2. Test info/config tasks (fast)
task config
task list:sdks
task workspace:list

# 3. Test icon generation (fast)
task icons:hybrid

# 4. Test at least one build (moderate)
task run:hybrid      # Builds + launches

# 5. Run Go tests (if they exist)
task test
```

### Full Task Test Suite

Create a test script to verify all tasks:

```bash
#!/bin/bash
# test-tasks.sh

echo "Testing Taskfile targets..."

# Info tasks
task config || echo "❌ config failed"
task list:sdks || echo "❌ list:sdks failed"
task workspace:list || echo "❌ workspace:list failed"

# Icon tasks
task icons:hybrid || echo "❌ icons:hybrid failed"

# Build tasks (one per platform to save time)
task build:hybrid:macos || echo "❌ build:hybrid:macos failed"

# Run task (check if app launches)
task run:hybrid || echo "❌ run:hybrid failed"

echo "✓ Task testing complete"
```

### Common Task Issues

**Problem**: Task fails with "invalid keys in command"
- **Cause**: YAML syntax error (usually unescaped colons in strings)
- **Fix**: Use single quotes or escape colons

**Problem**: Task fails with "command not found"
- **Cause**: Wrong path to binary or missing dependency
- **Fix**: Check {{.GOUP}} variable, verify binary exists

**Problem**: Task hangs
- **Cause**: Waiting for user input or long operation
- **Fix**: Add timeout or make operation non-interactive

### Task Testing Checklist

When modifying Taskfile.yml:

- [ ] Run `task --list` (verify syntax)
- [ ] Test the modified task
- [ ] Test any dependent tasks
- [ ] Verify task description is clear
- [ ] Check task works on clean environment
- [ ] Update this checklist if adding new categories

### Integration with CI/CD

**Future**: Add GitHub Actions workflow to test all tasks:

```yaml
# .github/workflows/test-tasks.yml
name: Test Taskfile
on: [push, pull_request]
jobs:
  test:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      - uses: arduino/setup-task@v1
      - run: task test
      - run: task build:hybrid:macos
      # etc.
```

### Remember

**Every task is a promise to users.** If `task run:hybrid` doesn't work, you've broken that promise.

Test your tasks. Keep them working. They're the user interface.
