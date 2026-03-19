# utm-dev TODO

**Status**: Tool works! Builds hybrid apps for macOS, iOS, Android successfully.  
**Next**: Polish the experience and expand platform support.

See [docs/IMPROVEMENTS.md](docs/IMPROVEMENTS.md) for comprehensive improvement analysis.

---

Now i want the whole root taskfile system designed so that in each example in the examples folder, i can have a task file that allows me to run from there . The intent is to allow my Developers to easily use the system on any project via task file include.  



---

## 🔥 High Priority (Do First)

### 1. Better Developer Experience
**Problem**: Builds are silent, errors are cryptic, no progress feedback  
**Impact**: Makes tool frustrating to use

- [ ] **Rich progress bars** with download speed, ETA, file sizes
- [ ] **Better error messages** with actionable suggestions and docs links
- [ ] **Structured logging** with --verbose mode
- [ ] **Build stage visibility** (Dependencies → Compilation → Packaging)

**Implementation**: Week 1-2 quick win from roadmap

---

### 2. Update Documentation
**Problem**: README doesn't reflect what tool actually does
**Impact**: People don't understand the value proposition

- [x] **Rewrite README** - show hybrid app capability ✅ (2025-10-21)
- [ ] **Quick start guide** that actually works end-to-end
- [ ] **Example showcase** - what you can build
- [x] **Platform support matrix** - what's tested vs what's possible ✅ (2025-10-21)

**Implementation**: 1-2 hours, do ASAP

---

### 2.5. Screenshot Command ✅ IMPLEMENTED
**Problem**: Need automated screenshots for docs/marketing
**Impact**: Can't show what the tool produces visually

**Solution**: robotgo-based screenshot system with App Store presets

**Completed Tasks**:
- [x] Implement screenshot command using robotgo (CGO-based, build tag gated)
- [x] Window capture functions: `CaptureActiveWindow`, `CaptureWindowByPID`
- [x] App Store screenshot presets (iOS, macOS, Android, Windows)
- [x] `run-and-capture` command - automated app launch + screenshot workflow
- [x] Taskfile integration - `screenshot-hybrid`, `screenshot-appstore-all`
- [x] Fallback to full-screen capture when window detection fails

**Current Status**: ✅ Working! Can capture screenshots of all examples.

**Remaining Improvements** (Move to dedicated section):
- [ ] Better preset display with `--store` filter
- [ ] Screenshot metadata/watermarking for App Store submissions
- [ ] Manual interactive mode (`--manual` flag to skip auto-detection)
- [ ] Screenshot comparison tool (HTML grid view)
- [ ] Screenshot validation (dimensions, file size, format)
- [ ] Better permission detection and error messages
- [ ] Screenshot cropping/post-processing (`--crop-*` flags)
- [ ] Screenshot workflow/batch mode for different app states

---

### 3. Webview Integration Improvements
**Problem**: Core feature but not well documented/supported  
**Impact**: People can't build production hybrid apps

- [ ] **Go ↔ JavaScript bridge** - declarative API for function exposure
- [ ] **TypeScript definitions generator** - type-safe bridge from Go types
- [ ] **DevTools integration** - forward console.log, enable network inspection
- [ ] **Hot reload** for web content during development
- [ ] **Production example** - real hybrid app showing best practices

**Implementation**: Phase 3 of roadmap (Q3 2025)

---

### 4. Windows Testing Automation
**Problem**: Can't easily test Windows builds from macOS  
**Impact**: Windows support is untested, might be broken

**The UTM Vision** (from old TODO):

```
┌─────────────────────────────────────────┐
│     macOS Development Machine           │
│                                          │
│  ┌────────────────────────────────┐     │
│  │  UTM (Virtual Machine)         │     │
│  │                                │     │
│  │  ┌──────────────────────────┐  │     │
│  │  │  Windows 11              │  │     │
│  │  │                          │  │     │
│  │  │  - Git (via winget)      │  │     │
│  │  │  - Go (via winget)       │  │     │
│  │  │  - VSCode (via winget)   │  │     │
│  │  │                          │  │     │
│  │  │  → Run utm-dev tests   │  │     │
│  │  │  → Build Windows apps    │  │     │
│  │  └──────────────────────────┘  │     │
│  └────────────────────────────────┘     │
└─────────────────────────────────────────┘

Automated via:
1. Packer + UTM plugin creates VM image
2. Go code provisions VM with winget packages
3. CI/CD runs tests in VM automatically
```

**Tasks**:
- [ ] **UTM automation** - Create Windows 11 VM from Go code
  - Use: https://github.com/naveenrajm7/packer-plugin-utm
- [ ] **Winget provisioning** - Install dev tools in VM
  - Use: https://github.com/mbarbita/go-winget
- [ ] **Test runner** - Execute utm-dev tests in Windows VM
- [ ] **CI integration** - Run Windows tests on every commit

**Implementation**: Phase 4 (Q4 2025) or when Windows support needed

---

## 🚀 Medium Priority

### 5. Screenshot System Enhancements
**Status**: Basic system working, these are polish improvements
**Impact**: Better App Store submission workflow, easier documentation

#### 5.1 Better Preset Management
- [ ] Add `--store` filter to `--list-presets`
  - Example: `utm-dev screenshot --list-presets --store ios`
  - Shows only iOS presets, cleaner output
- [ ] Group presets by store in `showPresets()` output
- [ ] Add preset search by dimensions
  - Example: `utm-dev screenshot --find-preset 1920x1080`

#### 5.2 Manual Interactive Mode
**Quick Win - Highest Priority**
- [ ] Add `--manual` flag to `run-and-capture`
  - Skips window detection entirely
  - Just launches app, waits configurable time, captures
  - User has full control over window positioning
- [ ] Add `--wait-manual` duration flag (default 5s)
- [ ] Print clear instructions: "Position window now, capturing in 5s..."

**Implementation**:
```go
runAndCaptureCmd.Flags().Bool("manual", false, "Manual mode - skip window detection")
runAndCaptureCmd.Flags().Int("wait-manual", 5000, "Wait time in manual mode (ms)")
```

#### 5.3 App Store Submission Tools
- [ ] Screenshot metadata/watermarking
  - Optional device frame overlay (iPhone frame around screenshot)
  - Optional text label (e.g., "iPhone 16 Pro Max")
  - Date/version watermark for tracking submissions
- [ ] Screenshot validation command
  - Check dimensions match App Store requirements
  - Check file size limits (iOS: 8MB max)
  - Check format (PNG or JPEG only)
  - Check color space (sRGB required)
- [ ] Screenshot comparison HTML generator
  - `task screenshot-compare-appstore`
  - Generates HTML grid showing all screenshots
  - Side-by-side view of all App Store sizes

#### 5.4 Advanced Window Detection
- [ ] macOS-specific: Try AppleScript for better window detection
  - `osascript` can get window bounds more reliably than robotgo
  - Fallback chain: robotgo → AppleScript → full screen
- [ ] Interactive window selection mode
  - User clicks on window they want to capture
  - Uses robotgo mouse tracking
- [ ] Better permission detection
  - Check for Screen Recording permission before capturing
  - Show macOS notification to grant permission
  - Provide clickable link to System Settings

#### 5.5 Post-Processing Features
- [ ] Screenshot cropping
  - `--crop-top 28` (remove macOS menu bar)
  - `--crop-bottom 0`
  - `--crop-left 0`
  - `--crop-right 0`
- [ ] Auto-trim whitespace/borders
- [ ] Resize to specific dimensions (for App Store compliance)
- [ ] Format conversion (PNG ↔ JPEG)
- [ ] Compression optimization

#### 5.6 Batch/Workflow Automation
- [ ] Capture multiple screenshots of same app at different stages
  - Example: Login screen → Dashboard → Settings
- [ ] Support for deep link triggered states
  - `--deeplink hybrid://settings` before screenshot
- [ ] Scripted interactions before capture
  - Click buttons, navigate tabs, fill forms
  - Uses robotgo automation capabilities
- [ ] Screenshot sequences for App Store preview videos

**Implementation Priority**:
1. **Week 1**: Manual interactive mode (#5.2) - Immediate value
2. **Week 2**: Screenshot validation (#5.3) - App Store workflow
3. **Week 3**: Better window detection (#5.4) - Polish
4. **Later**: Post-processing and batch features (#5.5, #5.6)

---

### 6. Performance Improvements
- [ ] **Incremental builds** - hash-based caching, skip unchanged
- [ ] **Parallel builds** - build multiple platforms concurrently
- [ ] **Parallel icon generation** - 5-10x faster
- [ ] **Docker build cache** - consistent, fast CI/CD builds

**Implementation**: Phase 2 (Q2 2025)

---

### 7. Configuration System
- [ ] **goup.yaml** - project configuration file
- [ ] **Platform-specific settings** - bundle IDs, permissions, signing
- [ ] **Build profiles** - debug, release, staging
- [ ] **CLI overrides** - flags override config file

**Implementation**: Week 2 quick win + ongoing

---

### 8. Testing & Deployment
- [ ] **Simulator/emulator automation** - `utm-dev test ios --simulator`
- [ ] **Device deployment** - `utm-dev deploy android --device`
- [ ] **Store helpers** - `utm-dev deploy appstore --testflight`
- [ ] **CI/CD templates** - GitHub Actions, CircleCI configs

**Implementation**: Phase 4 (Q4 2025)

---

## 🔮 Future (Nice to Have)

### 9. Cross-Compilation Fixes
- [ ] **Linux cross-compile** - Docker-based builds from macOS
- [ ] **Windows cross-compile** - Docker or remote builds
- [ ] **Better CGo detection** - warn early about cross-compile issues

### 10. Plugin System
- [ ] **Custom commands** - extend utm-dev via plugins
- [ ] **Build hooks** - pre-build, post-build, pre-deploy
- [ ] **Plugin marketplace** - share community plugins

### 11. Enhanced Examples
- [ ] **Hybrid dashboard** - Gio UI + web charts/graphs
- [ ] **Offline-first app** - IndexedDB + Go backend sync
- [ ] **Camera integration** - Native camera + Go processing
- [ ] **Push notifications** - FCM/APNs integration
- [ ] **OAuth flow** - Authentication with webview

---

## ✅ Completed

- [x] **Core build system** - macOS, iOS, Android working
- [x] **Webviewer example** - Multi-tab browser builds successfully
- [x] **Icon generation** - All platforms supported
- [x] **SDK management** - Caching, auto-install works
- [x] **Workspace support** - Multi-module projects
- [x] **Documentation** - CLAUDE.md, agents/, IMPROVEMENTS.md, WEBVIEW-ANALYSIS.md
- [x] **Testing** - Validated on real platforms
- [x] **Deep analysis** - Know what needs improvement

---

## 📊 Progress Tracking

**Current Phase**: Proof of Concept → Production Ready  
**Next Milestone**: Week 1-4 Quick Wins (Better UX)  
**Long-term Goal**: Best Go hybrid app framework

---

## 🎯 This Week

Focus on **immediate impact**:

1. **Tonight**: Update README (1 hour)
2. **This week**: Better build feedback (2-3 days)
3. **Next week**: Production example app (3-4 days)

Small wins → momentum → adoption → ecosystem

---

## 💡 Ideas Parking Lot

Random ideas to evaluate later:

- **Winget MDM** (https://github.com/jantari/rewinged) - Host winget manifests for internal tools
- **Desktop PWA mode** - Gio app that IS a web browser for PWAs
- **Bridge tooling** - Auto-generate bridge code from OpenAPI specs
- **Visual builder** - GUI for designing Gio layouts
- **Hot reload for Go** - Recompile and restart on code changes
- **Remote build farm** - Build iOS apps without owning a Mac

---

**See also**:
- [docs/IMPROVEMENTS.md](docs/IMPROVEMENTS.md) - Comprehensive improvement roadmap
- [docs/WEBVIEW-ANALYSIS.md](docs/WEBVIEW-ANALYSIS.md) - Cross-platform webview deep dive
- [docs/agents/](docs/agents/) - AI assistant collaboration guides

---

## 🔗 Gio v0.9.1 + gogio Updates (2025-12-20)

New features from the latest Gio, gio-plugins, and gio-cmd updates.

### gogio New Features (gio-cmd PRs merged Dec 2025)

**PR #9: Deep Linking / Custom URI Schemes** ✨ FULLY IMPLEMENTED
- [x] `-schemes` flag merged into gogio (Dec 15, 2025)
- [x] Add `--schemes` flag to utm-dev build command ✅
- [x] Support for Android, iOS, macOS, Windows deep links ✅
- [x] Example: `utm-dev build macos --schemes hybrid` ✅
- [x] Integration with webviewer (app.URLEvent → webview navigation) ✅
- [x] Taskfile tasks: `demo:deeplink`, `test:deeplink`, `build:hybrid:*:deeplink` ✅
- [x] hybrid-dashboard example with deep link handling ✅

**PR #23: Android App Queries** ✅ IMPLEMENTED
- [x] `-queries` flag merged (Dec 16, 2025)
- [x] Add `--queries` flag to utm-dev build command ✅
- [x] Enables checking if apps are installed before launching intents ✅
- [x] Example: `utm-dev build android --queries com.google.android.apps.maps` ✅

**PR #21: iOS App Store Compatibility Fixes**
- [x] All 6 validation issues fixed (Dec 16, 2025)
- [x] Bitcode stripping (Asset validation 90482)
- [x] 3-part version format (1.2.3 not 1.2.3.4)
- [x] iPad multitasking disabled by default
- [x] iPad 152x152 icon auto-generated
- [ ] Test App Store upload with utm-dev built apps

**PR #22: WASM Go 1.23+ Compatibility**
- [x] Fixed WASM compatibility (Dec 15, 2025)
- [ ] Test `utm-dev build wasm` with Go 1.25

**PR #19: Android 15+ / 16KB Page Size**
- [x] 64KB page-size for Android 15+ (May 2025)
- [x] Google Play requires 16KB-compatible by Nov 2025
- [ ] Verify utm-dev Android builds are compatible

**PR #20: macOS/iOS Signing & Profiles** ✅ IMPLEMENTED
- [x] Custom profile support merged
- [x] Add `--signkey` flag (keystore, Keychain key, or provisioning profile) ✅

### gio-plugins Updates

**Auth Module (Issue #106)**
- [ ] Test OAuth flows (Apple, Google sign-in)
- [ ] Verify auth callbacks work correctly

### Platform Improvements
- [ ] Test Windows touch screen support in webviewer
- [ ] Test macOS fullscreen MaxSize with webviewer apps
- [ ] Verify Android text rendering fixes

**Current versions (2025-12-20):**
```bash
# Update gio-cmd (gogio) to get new features
go install gioui.org/cmd/gogio@latest

# In your project
go get gioui.org@7bcb315ee174  # v0.9.1-0.20251215212054-7bcb315ee174
go get github.com/gioui-plugins/gio-plugins@v0.9.1
```

---

## 🎯 Screenshot & Documentation Tasks

### Capture App Screenshots
Use Playwright MCP or native tools to create visual documentation:

- [ ] **macOS webviewer** - Running desktop app with tabs/browser
- [ ] **iOS simulator** - App running in iPhone simulator
- [ ] **Android emulator** - App running in Android emulator
- [ ] **All three side-by-side** - Show cross-platform capability

Save to `docs/screenshots/` and link in README.

### Create Complete Hybrid Example

**Problem**: Current webviewer just loads Google.com (external URL)

**Better**: `examples/hybrid-app-complete/` with:

```
hybrid-app-complete/
├── main.go              # Gio UI + embedded HTTP server
├── go.mod
├── icon-source.png
└── web/
    ├── index.html       # Landing page
    ├── app.js           # JavaScript with Go bridge calls
    ├── styles.css       # Styling
    └── assets/
        └── logo.png
```

**Features to demonstrate**:
- ✅ Embedded `//go:embed` web content (no external deps)
- ✅ Local HTTP server on localhost:8080
- ✅ Go ↔ JavaScript bridge (call Go functions from JS)
- ✅ Native Gio UI navigation (tabs, buttons)
- ✅ WebView displaying embedded content
- ✅ Offline-capable (all assets embedded)
- ✅ Works on all platforms (iOS, Android, macOS, Windows)

**This becomes THE showcase example** - proves the vision works end-to-end.

Priority: HIGH (After README update)
