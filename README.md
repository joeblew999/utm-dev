# utm-dev

https://github.com/joeblew999/utm-dev

**About:** Write HTML/HTMX once, deploy everywhere: Web, iOS, Android, Desktop—with instant updates and no app store approval.

---

**Build cross-platform applications using standard web technologies.**

Write HTML/CSS once → Deploy everywhere: Web, iOS, Android, Desktop

![Status](https://img.shields.io/badge/status-alpha-orange)
![Go Version](https://img.shields.io/badge/go-1.25%2B-blue)
![Platforms](https://img.shields.io/badge/platforms-macOS%20%7C%20iOS%20%7C%20Android%20%7C%20Windows%20%7C%20Linux-green)

**[Documentation](https://joeblew999.github.io/utm-dev/)** | **[Quick Start](https://joeblew999.github.io/utm-dev/users/quickstart/)** | **[Platform Support](https://joeblew999.github.io/utm-dev/users/platforms/)** | **[Examples](examples/)**

<p align="center">
  <img src="docs/screenshots/hybrid-dashboard-macos.png" alt="Hybrid Dashboard - Native app with web UI" width="600">
  <br>
  <em>Hybrid Dashboard: Native macOS app with embedded web UI (HTML/CSS/HTMX)</em>
</p>

## 📊 Project Status

**Current Phase**: Production Polish (Core System Complete)

### ✅ What Works Today
- ✅ **Build system fully functional** - macOS, iOS, Android builds working
- ✅ **Cross-platform packaging** - Native app bundles generated successfully
- ✅ **SDK management** - Automatic installation and caching
- ✅ **Icon generation** - Multi-platform icon creation
- ✅ **Webview integration** - Native webviews working on all tested platforms
- ✅ **Example apps** - Working demos for webviewer and hybrid apps

### 🔨 Current Focus (Polish Phase)
- 🔧 Self-updating improvements for utm-dev binary
- 🔧 Enhanced error messages and user feedback
- 🔧 Progress indicators for long operations
- 🔧 Documentation refinement
- 🔧 Windows/Linux cross-compilation fixes
- 🔧 **App Store deployment & analytics** - Single pane of glass for POs and devs
  - Automated App Store/Google Play submission
  - Real-time user feedback monitoring
  - Review tracking and sentiment analysis
  - 360° operational dashboard

### 🎯 Ready For
- ✅ **Early adopters** - Build real applications today
- ✅ **Proof of concepts** - Validate the approach
- ✅ **Pilot projects** - Small to medium deployments
- ⚠️ **Not yet**: Large-scale production (still polishing rough edges)

**Want to help?** See [Contributing](#contributing) or [open an issue](https://github.com/joeblew999/utm-dev/issues)

---

## Why This Matters (For Decision Makers & Investors)

### The Problem: Building Apps is Expensive and Slow

**Traditional app development requires:**
- 5+ specialized developers (iOS Swift, Android Kotlin, Backend, Frontend, DevOps)
- Months to build initial version
- **Weeks to deploy updates** (app store review process)
- **Multiple codebases** to maintain (iOS, Android, Web, Desktop)
- **Version hell**: Supporting multiple client versions simultaneously
- **Vendor lock-in**: Dependent on Apple, Google, Microsoft ecosystems

**Cost**: $500K-$2M for enterprise-grade cross-platform app
**Time to market**: 6-12 months
**Update cycle**: 1-4 weeks (app store approval)

---

### The Solution: Digital Sovereignty with Tiny Teams

**utm-dev + HTMX/Datastar enables:**

✅ **1-2 developers** instead of 5+ specialists
✅ **Weeks to build** instead of months
✅ **Minutes to deploy updates** (no app store approval)
✅ **One codebase** for Web, iOS, Android, Desktop
✅ **One version** to support (server-side UI)
✅ **Zero vendor lock-in** (your infrastructure, your control)

**Cost**: $50K-$200K for same capability
**Time to market**: 4-8 weeks
**Update cycle**: **Instant** (server-side deployment)

---

### What Makes This Possible

**The breakthrough**: Your UI is **standard web technology (HTML/CSS)** served from your server. Native apps are just thin wrappers around webviews.

```
┌─────────────────────────────────────────────────────────┐
│  What Your Developers Actually Write                    │
├─────────────────────────────────────────────────────────┤
│  95% → HTML/CSS with HTMX or Datastar (web tech)       │
│   4% → Go backend (your business logic)                 │
│   1% → utm-dev commands (packaging)                   │
│   0% → Swift, Kotlin, Xcode, Android Studio            │
└─────────────────────────────────────────────────────────┘
```

**When you update your HTML/CSS on the server, ALL devices get it instantly.**

No iOS build. No Android build. No app store submission. No waiting.

---

### Real-World Impact

**🏛️ Government: Digital Sovereignty**
- Build citizen services without foreign cloud dependency
- Update instantly for policy/regulatory changes
- Deploy on government infrastructure
- No Apple/Google gatekeepers

**🏢 Enterprise: Vendor Independence**
- CRM, ERP, dashboards without SaaS lock-in
- Small internal team builds/maintains
- On-premises or private cloud deployment
- Instant updates for compliance

**🏪 SME: Efficiency & Control**
- One developer builds iOS + Android + Web + Desktop
- No monthly fees to Salesforce/ServiceNow/etc
- Update pricing/features in real-time
- Own your customer relationships and data

**🏥 Healthcare: Compliance & Security**
- HIPAA/GDPR on your infrastructure
- Air-gapped deployment possible
- Full audit trail
- No third-party code review delays

---

### The Economics

**Traditional Enterprise App Development:**
```
Team:
  - iOS Developer:        $150K/year
  - Android Developer:    $150K/year
  - Frontend Developer:   $130K/year
  - Backend Developer:    $140K/year
  - DevOps Engineer:      $160K/year
  - Project Manager:      $120K/year
────────────────────────────────────
  Total Payroll:         $850K/year

App Store Fees:          $20K-50K/year
Cloud Infrastructure:    $30K-100K/year
────────────────────────────────────
  Annual Cost:           $900K-$1M+
```

**utm-dev + HTMX/Datastar:**
```
Team:
  - Go Developer:         $140K/year
  - Junior Go Dev:        $90K/year
────────────────────────────────────
  Total Payroll:         $230K/year

Infrastructure:          $10K-30K/year
────────────────────────────────────
  Annual Cost:           $240K-$260K

  Savings:               $650K-$750K/year (75% reduction)
```

**Plus**: 10x faster iteration, instant updates, no version hell.

---

### Investment Opportunity

**Market**: Global app development market ($200B+, growing 15% annually)

**Opportunity**: Enable organizations to:
- Reduce app dev costs by 70-80%
- Achieve digital sovereignty (own their stack)
- Eliminate vendor lock-in
- Deploy updates instantly (no app store gatekeepers)

**Target Customers**:
- Governments seeking digital independence
- Enterprises wanting vendor independence
- SMEs needing efficiency
- Healthcare/regulated industries requiring control

**Competitive Advantage**:
- **Electron/Tauri**: Desktop only, no mobile
- **Flutter**: Complex, requires Dart specialists
- **React Native**: JavaScript hell, version chaos
- **Native**: Most expensive, slowest

**utm-dev**: Simple (web tech), fast (instant updates), cheap (tiny teams), sovereign (your infrastructure).

---

## Technical Overview (For Developers)

### What You Actually Write

**You write standard web applications with HTMX or Datastar:**

```html
<!-- Your entire UI is HTML + HTMX (NO Gio code, NO Swift, NO Kotlin) -->
<!DOCTYPE html>
<html>
<head>
    <script src="https://unpkg.com/htmx.org"></script>
</head>
<body>
    <!-- Real-time dashboard -->
    <div hx-get="/api/stats" hx-trigger="every 2s">
        Loading stats...
    </div>

    <!-- Button that posts to backend -->
    <button hx-post="/api/process"
            hx-swap="outerHTML">
        Process Data
    </button>
</body>
</html>
```

**Your Go backend:**
```go
package main

import (
    "embed"
    "net/http"
)

//go:embed web/*
var webContent embed.FS

func main() {
    // Serve your HTML/CSS/JS
    http.Handle("/", http.FileServer(http.FS(webContent)))

    // Handle HTMX requests
    http.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "text/html")
        fmt.Fprintf(w, `<div>Users: %d, Sales: $%d</div>`, users, sales)
    })

    http.ListenAndServe("localhost:8080", nil)
}
```

**That's it.** 95% HTML/CSS, 4% Go, 1% packaging. **Zero Gio code, zero Swift, zero Kotlin.**

---

### Package for All Platforms

```bash
# Build native apps (utm-dev creates thin wrappers automatically)
utm-dev build ios myapp
utm-dev build android myapp
utm-dev build macos myapp
utm-dev build windows myapp

# Users install native apps
# Apps load HTML from YOUR server
# You update server → everyone gets changes instantly
```

The native apps are just **thin webview wrappers** created automatically. You never write native code.

---

### Access Native Features (When Needed)

**Most apps just need HTML.** But when you need native capabilities (file picker, camera, etc.), import a plugin:

```go
import "github.com/gioui-plugins/gio-plugins/explorer"

http.HandleFunc("/api/pick-file", func(w http.ResponseWriter, r *http.Request) {
    file := explorer.PickFile()  // Native file picker on all platforms!
    // Process file...
})
```

**Your HTML just calls the API:**
```html
<button hx-post="/api/pick-file">Pick File</button>
```

**Available native features** (no platform-specific code):
- 📂 File picker / save dialog
- 🔗 Open URLs in browser
- 📤 Native share sheets
- 🔐 OAuth / authentication
- 💾 Secure storage
- 📸 Camera (coming soon)
- 🔔 Notifications (coming soon)
- 📍 Location (coming soon)

👉 [See plugin roadmap](https://github.com/orgs/gioui-plugins/projects/1)

---

### The Complete Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Your App (Mostly Web Tech)                    │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  Web UI (HTML/CSS + HTMX/Datastar)                        │  │
│  │  - Standard web frontend (95% of your code)                │  │
│  │  - Lives on YOUR server                                    │  │
│  │  - Update instantly (no app rebuild)                       │  │
│  └────────────────────────────────────────────────────────────┘  │
│                              ↕                                    │
│                       Go HTTP Server                              │
│                              ↕                                    │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  Go Backend (4% of your code)                             │  │
│  │                                                             │  │
│  │  Optional Plugins (import when needed):                    │  │
│  │  📂 File Picker    🔗 Hyperlinks    📤 Share               │  │
│  │  🔐 OAuth/Auth     💾 Storage       📧 Email               │  │
│  │  📸 Camera         🔔 Notifications 📍 Location            │  │
│  └────────────────────────────────────────────────────────────┘  │
│                              ↕                                    │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │  Thin Native Wrapper (Automatic - utm-dev handles this) │  │
│  │  (1% of your work - just run utm-dev build)             │  │
│  │                                                             │  │
│  │  🍎 iOS: WKWebView      🤖 Android: WebView               │  │
│  │  🖥️  macOS: WKWebView    🪟 Windows: WebView2             │  │
│  │  🐧 Linux: WebKitGTK     🌐 Web: Direct browser           │  │
│  └────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────┘
```

**Key insight**: You write HTML/HTMX (web code). Native apps are just wrappers created by utm-dev. **You never write Gio/Swift/Kotlin code** unless you need custom native UI (extremely rare).

---

### Why HTMX & Datastar?

**HTMX** - Hypermedia-driven interactions (most popular choice)
- Server renders HTML, browser displays it
- No client-side state management
- No React, no Vue, no Angular, no webpack, no npm hell
- Progressive enhancement (works without JS)

**Datastar** - Real-time reactivity without complexity
- Server-Sent Events (SSE) for live updates
- Server controls state, client reflects it
- Signals pattern without JavaScript framework complexity

**Both**: Minimal JavaScript, maximum power. Perfect for webviews.

**Example with HTMX:**
```html
<!-- Your web UI - works identically on browser, iOS, Android, desktop -->
<button hx-post="/api/process"
        hx-trigger="click"
        hx-swap="outerHTML">
    Process Data
</button>

<!-- Server responds with HTML fragment -->
<div class="result">
    ✅ Processed 1,234 records in 0.5s
</div>
```

**Example with Datastar:**
```html
<!-- Real-time updates from server -->
<div data-star-watch="$get('/api/live-stats')">
    Users: <span data-star-text="$users"></span>
    Status: <span data-star-text="$status"></span>
</div>
```

No React. No build tools. Just HTML served from your Go backend.

---

### Instant Updates: No App Store Required

**The breakthrough**: Your web UI lives on YOUR server. Updates happen **instantly**.

**Traditional native apps:**
```
1. Write code
2. Build for iOS, Android, macOS, Windows
3. Submit to app stores
4. Wait 1-7 days for review
5. Hope it's approved
6. Users gradually update (months!)
7. Now support 5 different versions in production
```

**utm-dev + web UI:**
```
1. Update HTML/CSS on your server
2. That's it. Everyone gets it instantly.
```

**For critical bugs**: Fix in minutes, not weeks.
**For new features**: Deploy and iterate rapidly.
**For compliance**: Government/enterprise requirements met immediately.

---

### No More Version Hell

**Traditional problem:**
```
Mobile App v1.0.0  →  Expects API v1
Mobile App v1.1.0  →  Expects API v1 + v2
Mobile App v1.2.0  →  Expects API v2

Your backend must support ALL THREE simultaneously!
```

**Web UI solution:**
```
All devices →  Latest HTML from server
               Always in sync
               One version to support
```

Your server serves the current UI. No client-side versioning. No API compatibility matrix. **No version hell.**

---

### Cross-Platform URI Routing & Deep Linking

**[wellknown](https://github.com/joeblew999/wellknown)** - Universal deep linking system:

```go
import "github.com/joeblew999/wellknown"

// Open email in user's preferred client (Gmail, Apple Mail, Outlook)
wellknown.OpenEmail("user@example.com", "Subject", "Body")

// Open calendar event (Google Calendar, Apple Calendar, etc.)
wellknown.OpenCalendar(event)

// Your custom app URI scheme
wellknown.RegisterScheme("myapp://")
wellknown.HandleURI("myapp://dashboard?tab=analytics")
```

**What this enables**:
- 📱 Deep Linking (Web → Native app, App → App)
- 🌐 Universal Links (iOS) / App Links (Android)
- 🔗 Cross-Platform Actions (open in preferred app)
- 🎯 Custom URL Schemes

**You own the routing. You decide the flow. Big Tech becomes optional enhancement, not requirement.**

---

## Platform Support

| Platform | Build | Status | Notes |
|----------|-------|--------|-------|
| **macOS** | ✅ | Tested | Native .app bundles with WKWebView |
| **iOS** | ✅ | Tested | App Store ready, WKWebView integration |
| **Android** | ✅ | Tested | APK generation, Chromium WebView |
| **Windows** | ⚠️ | Untested | WebView2 support (cross-compile issue) |
| **Linux** | ⚠️ | Untested | WebKitGTK support (cross-compile issue) |
| **Web** | 🔜 | Planned | WASM deployment |

**All tested platforms work successfully!** ✨

---

## Quick Start

### Prerequisites

- **Go 1.25+**
- **macOS**: Xcode Command Line Tools
- **Android**: Auto-installed via `utm-dev install ndk-bundle`

### Install utm-dev

**Quick Install (Recommended)**:

```bash
# macOS (via curl)
curl -fsSL https://raw.githubusercontent.com/joeblew999/utm-dev/main/scripts/macos-bootstrap.sh | bash

# Windows (via PowerShell as Administrator)
iwr https://raw.githubusercontent.com/joeblew999/utm-dev/main/scripts/windows-bootstrap.ps1 -UseBasicParsing | iex
```

This installs:
- ✅ Go (via Homebrew/winget)
- ✅ Task (Taskfile runner)
- ✅ utm-dev (latest release binary)
- ✅ Git (if needed)

**Manual Install**:

```bash
# Clone the repository
git clone https://github.com/joeblew999/utm-dev.git
cd utm-dev

# Build from source
go build .

# Or use pre-built binaries from GitHub Releases
# https://github.com/joeblew999/utm-dev/releases/latest
```

**Update utm-dev**:

```bash
utm-dev self upgrade
```

---

## Using Taskfile (Recommended)

We provide a [Taskfile](https://taskfile.dev) for common operations:

```bash
# Install Task first (if you don't have it)
brew install go-task/tap/go-task

# IMPORTANT: Fix Gio version compatibility (MUST DO FIRST!)
task fix-versions

# Check version compatibility
task doctor

# See all available tasks
task --list

# Quick demo - builds and runs hybrid-dashboard
task demo

# Build and run specific examples
task run:hybrid        # Hybrid dashboard with embedded server
task run:webviewer     # Multi-tab browser
task run:basic         # Simple Gio app

# Build for different platforms
task build:hybrid:macos
task build:hybrid:ios
task build:hybrid:android
task build:hybrid:all   # All platforms

# SDK management
task install:ndk        # Install Android NDK
task list:sdks          # Show available SDKs

# Development
task workspace:init     # Initialize Go workspace
task clean             # Clean build artifacts
task test              # Run tests

# Full setup from scratch
task setup             # Initialize workspace + install NDKs
```

**Quick start with Task:**
```bash
git clone https://github.com/joeblew999/utm-dev.git
cd utm-dev
task setup    # One command to set everything up
task demo     # See hybrid-dashboard in action!
```

---

## What Can You Build?

### Hybrid Apps with Web Content
- **Dashboards** - HTML charts/graphs with native shell
- **Content Apps** - Web articles with native navigation
- **Dev Tools** - Web inspector with native IDE
- **Documentation** - Rendered markdown with native app

### Business Applications
- **CRM/ERP** - Without Salesforce/ServiceNow fees
- **Point-of-Sale** - Update pricing instantly
- **Inventory Management** - Real-time updates
- **Employee Portals** - Instant compliance updates

### Government & Enterprise
- **Citizen Services** - Digital sovereignty
- **Internal Tools** - Vendor independence
- **Healthcare Apps** - HIPAA/GDPR compliance
- **Regulated Industries** - Air-gapped deployment

---

## Examples

### Basic Gio App
```bash
go run . build macos examples/gio-basic
```
Simple pure-Gio application showing native UI capabilities.

### Webviewer (Multi-tab Browser)
```bash
go run . build macos examples/gio-plugin-webviewer
```
**The key example** - demonstrates:
- ✅ Multiple webviews (tabs)
- ✅ URL navigation
- ✅ JavaScript execution
- ✅ Storage access (cookies, localStorage)
- ✅ Native UI + web content integration

### Hybrid Dashboard
```bash
go run . build macos examples/hybrid-dashboard
```
Complete example with:
- ✅ Embedded HTTP server
- ✅ HTMX real-time updates
- ✅ Native file picker integration
- ✅ Deploy to iOS, Android, Desktop

---

## Features

### 🎨 Automatic Icon Generation
```bash
# Generate platform-specific icons from one source
go run . icons macos examples/my-app
go run . icons android examples/my-app
go run . icons ios examples/my-app
```

### 📦 SDK Management
```bash
# Auto-installs and caches SDKs
go run . install ndk-bundle        # Android NDK
go run . install android-sdk        # Android SDK
go run . list                       # Show available SDKs
```

### 🔧 Workspace Integration
```bash
# Manage multi-module projects
go run . workspace list
go run . ensure-workspace examples/my-app
```

### 🚀 Self-Building
```bash
# Build utm-dev itself
go run . self build
```

### 📊 App Store Deployment & Analytics (Coming Soon)

**Single pane of glass for Product Owners and Developers** - Complete 360° operational view:

```bash
# Deploy to app stores (coming soon)
utm-dev deploy ios myapp --store
utm-dev deploy android myapp --store

# Monitor user feedback
utm-dev analytics myapp --reviews
utm-dev analytics myapp --dashboard
```

**What you get:**
- 🚀 **Automated deployment** - Push to App Store & Google Play with one command
- 📱 **Review monitoring** - Real-time user feedback from all app stores
- 📊 **Sentiment analysis** - Track user satisfaction trends
- 🎯 **Dashboard** - Single view of deployments, reviews, ratings across platforms
- 🔔 **Alerts** - Get notified of negative reviews or rating drops
- 📈 **Analytics** - Usage patterns, crash reports, performance metrics

**Why this matters:**
- ✅ **POs**: Track user sentiment without logging into multiple stores
- ✅ **Devs**: See real user feedback immediately after deployment
- ✅ **Support**: Respond to negative reviews quickly
- ✅ **Product**: Data-driven decisions from unified view

No more jumping between App Store Connect, Google Play Console, and analytics tools. **One command. One dashboard. Complete visibility.**

---

## Architecture

**Idempotent**: All operations are safe to run multiple times
**DRY**: Centralized path management, no duplication
**Clean**: Service layer ready for future API use
**Caching**: SDKs downloaded once, reused forever

---

## Documentation

- **[IMPROVEMENTS.md](docs/IMPROVEMENTS.md)** - Roadmap and future enhancements
- **[WEBVIEW-ANALYSIS.md](docs/WEBVIEW-ANALYSIS.md)** - Deep dive into cross-platform webview support
- **[TODO.md](TODO.md)** - Current tasks and priorities
- **[CLAUDE.md](CLAUDE.md)** - AI assistant guide (for development)
- **[docs/agents/](docs/agents/)** - Dependency guides for AI collaboration
- **[docs/platforms.md](docs/platforms.md)** - Platform-specific build information

---

## Project Status

**Current Phase**: Proof of Concept → Production Ready

**What Works**:
- ✅ Builds succeed on macOS, iOS, Android
- ✅ Webviewer hybrid apps work on all tested platforms
- ✅ Icon generation for all platforms
- ✅ SDK caching and management
- ✅ Multi-module workspace support

**What's Next** (see [IMPROVEMENTS.md](docs/IMPROVEMENTS.md)):
1. **Better UX** - Progress bars, error messages, feedback
2. **Performance** - Incremental builds, parallel operations
3. **Webview Excellence** - Go ↔ JS bridge, TypeScript defs, DevTools
4. **Testing** - Automated testing, deployment helpers
5. **Windows/Linux** - Fix cross-compilation issues

---

## Why utm-dev?

### vs Electron/Tauri
- ✅ **Much smaller binaries** (~5MB vs 100MB+)
- ✅ **Native webviews** (WKWebView, WebView2, not embedded Chromium)
- ✅ **Better performance** (no Node.js/V8 overhead)
- ✅ **Pure Go backend** (one language, one ecosystem)
- ✅ **True mobile support** (iOS + Android, not just desktop)
- ✅ **Lower memory usage** (system webview, not full browser)
- ✅ **Instant updates** (server-side UI)

### vs Flutter
- ✅ **Use web technologies** (HTML/CSS/HTMX you already know)
- ✅ **Pure Go backend** (no Dart required)
- ✅ **Leverage web ecosystem** (existing web skills)
- ✅ **Progressive enhancement** (start as web app, package natively)
- ✅ **Simpler architecture** (no custom rendering engine)
- ✅ **Instant updates** (no app store submission)

### vs Native (SwiftUI/Jetpack Compose)
- ✅ **Cross-platform** (write once, deploy everywhere)
- ✅ **One language** (Go for backend, HTML for UI)
- ✅ **Web-first workflow** (develop in browser, package as native)
- ✅ **Instant updates** (no app rebuild)
- ✅ **Faster iteration** (web dev tools, hot reload)
- ✅ **75% cost reduction** (1-2 devs instead of 5+)

### vs React Native
- ✅ **No JavaScript chaos** (HTMX instead of React)
- ✅ **No version hell** (server-side UI)
- ✅ **No npm dependency hell** (Go modules)
- ✅ **True native webviews** (not JavaScript bridge)
- ✅ **Instant updates** (no CodePush complexity)

---

## Contributing

We're in active development! See [TODO.md](TODO.md) for current priorities.

**Quick wins needed**:
- Better progress feedback during builds
- Error messages with suggestions
- Complete hybrid app example with embedded server
- Screenshots and visual documentation
- Windows/Linux cross-compilation fixes

---

## License

[Check LICENSE file]

---

## Credits

Built on top of:
- **[Gio UI](https://gioui.org)** - Pure Go immediate-mode UI (thin wrapper layer)
- **[gio-plugins](https://github.com/gioui-plugins/gio-plugins)** - Native feature access
- **[Cobra](https://github.com/spf13/cobra)** - CLI framework
- **[HTMX](https://htmx.org)** - Hypermedia for web UIs
- **[Datastar](https://data-star.dev)** - Real-time reactivity
- **[wellknown](https://github.com/joeblew999/wellknown)** - URI scheme routing

### Special Thanks

A huge thank you to **[@inkeliz](https://github.com/inkeliz)** for creating and maintaining [gio-plugins](https://github.com/gioui-plugins/gio-plugins) and [gioismobile](https://github.com/inkeliz/gioismobile). The native webview integration that makes this entire project possible is built on inkeliz's work. The webviewer plugin provides cross-platform WKWebView/WebView2/WebKitGTK support that is the foundation of utm-dev's hybrid app architecture, and gioismobile provides runtime mobile platform detection.

---

## Vision

**Make Go the best choice for cross-platform web-based application development.**

No Swift. No Kotlin. Just standard web tech (HTML/CSS/HTMX) + Go backend.

Build sovereign systems. Own your stack. Control your destiny. 🚀

**Resources:**
- 📚 [HTMX Documentation](https://htmx.org)
- 🌟 [Datastar Documentation](https://data-star.dev)
- 🔌 [gio-plugins roadmap](https://github.com/orgs/gioui-plugins/projects/1)
- 🔐 [wellknown - Self-sovereign routing](https://github.com/joeblew999/wellknown)
- 🔄 [Automerge - Distributed data](https://github.com/joeblew999/automerge-wazero-example)

---

## Contact & Support

### For Investors & Enterprise Inquiries

Interested in digital sovereignty solutions or investment opportunities?

**Website**: [ubuntusoftware.net](https://www.ubuntusoftware.net)

### For Developers

- 💬 **Discussions**: [GitHub Discussions](https://github.com/joeblew999/utm-dev/discussions)
- 🐛 **Bug Reports**: [GitHub Issues](https://github.com/joeblew999/utm-dev/issues)
- 📖 **Documentation**: See [docs/](docs/) folder
- 💡 **Feature Requests**: [GitHub Issues](https://github.com/joeblew999/utm-dev/issues)

### Commercial Support & Consulting

Building mission-critical applications? Need enterprise support?

Contact us via [ubuntusoftware.net](https://www.ubuntusoftware.net) for:
- 🏢 Enterprise licensing & support
- 🎓 Training & onboarding
- 🔧 Custom development & integration
- 🏛️ Government & compliance consulting
- 🚀 Migration from existing platforms

---

**Built with ❤️ for digital sovereignty**
