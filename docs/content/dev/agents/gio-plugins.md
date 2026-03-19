---
title: "gio-plugins Reference"
date: 2025-12-21
draft: false
---

# gio-plugins Agent Guide

## Overview

**Repository**: https://github.com/gioui-plugins/gio-plugins  
**Local Source**: `.src/gio-plugins/`  
**Version**: v0.8.0

The gio-plugins repository provides native plugins for Gio UI applications, enabling features like webviews, authentication, and hyperlinks across multiple platforms.

## Key Plugins

### 1. WebViewer (`webviewer/`)

**Purpose**: Embed native webviews in Gio applications

**Location in .src**: `.src/gio-plugins/webviewer/`

**Key Directories**:
- `webview/` - Core webview implementation (platform-specific)
- `giowebview/` - Gio integration layer
- `demo/` - Example browser application
- `installview/` - Installation webview helper

**Platform Support**:
- ✅ macOS (WKWebView)
- ✅ iOS (WKWebView)
- ✅ Android (Chromium WebView)
- ✅ Windows (WebView2)
- ✅ Linux (WebKitGTK)
- ❓ Web/WASM (unclear)

**Key Files to Review**:
```
.src/gio-plugins/webviewer/
├── webview/
│   ├── webview.go          # Main webview interface
│   ├── webview_darwin.go   # macOS/iOS implementation
│   ├── webview_android.go  # Android implementation
│   ├── webview_windows.go  # Windows implementation
│   └── webview_linux.go    # Linux implementation
├── giowebview/
│   ├── webview.go          # Gio operations and events
│   └── ...
└── demo/
    └── demo.go             # Full browser example (our webviewer is based on this)
```

**API Patterns**:
```go
// Operations
giowebview.WebViewOp{Tag: viewTag}      // Create/render webview
giowebview.NavigateCmd{View: tag, URL: url}  // Navigate to URL
giowebview.OffsetOp{Point: point}       // Position webview
giowebview.RectOp{Size: size}           // Size webview

// Events
giowebview.TitleEvent         // Page title changed
giowebview.NavigationEvent    // URL changed
giowebview.CookiesEvent       // Cookies data
giowebview.StorageEvent       // localStorage/sessionStorage
giowebview.MessageEvent       // JavaScript messages
```

**Common Patterns**:

1. **Creating a WebView**:
```go
tag := new(int)  // Unique tag for this webview
defer giowebview.WebViewOp{Tag: tag}.Push(gtx.Ops).Pop(gtx.Ops)
giowebview.RectOp{Size: size}.Add(gtx.Ops)
```

2. **Handling Events**:
```go
for {
    evt, ok := gioplugins.Event(gtx, giowebview.Filter{Target: tag})
    if !ok { break }
    
    switch evt := evt.(type) {
    case giowebview.TitleEvent:
        title = evt.Title
    case giowebview.NavigationEvent:
        currentURL = evt.URL
    }
}
```

3. **Executing Commands**:
```go
gioplugins.Execute(gtx, giowebview.NavigateCmd{
    View: tag,
    URL:  "https://example.com",
})
```

### 2. Hyperlink (`hyperlink/`)

**Purpose**: Open URLs in system browser

**Key Files**: `.src/gio-plugins/hyperlink/hyperlink.go`

### 3. Auth (`auth/`)

**Purpose**: OAuth and authentication flows

**Location**: `.src/gio-plugins/auth/`

### 4. Explorer (`explorer/`)

**Purpose**: File system explorer/picker

**Location**: `.src/gio-plugins/explorer/`

## Integration with utm-dev

**Example Projects**:
- `examples/gio-plugin-webviewer/` - Multi-tab browser
- `examples/gio-plugin-hyperlink/` - Hyperlink demo

**Build Commands**:
```bash
# Build webviewer example for different platforms
go run . build macos examples/gio-plugin-webviewer
go run . build windows examples/gio-plugin-webviewer
go run . build android examples/gio-plugin-webviewer
go run . build ios examples/gio-plugin-webviewer
```

## Development Workflow

### Reading Plugin Source

When working on webview features:

1. Check implementation in `.src/gio-plugins/webviewer/`
2. Review platform-specific files (`*_darwin.go`, `*_android.go`, etc.)
3. Study the demo app in `.src/gio-plugins/webviewer/demo/demo.go`
4. Test with our example in `examples/gio-plugin-webviewer/`

### Common Tasks

**Understanding Platform Behavior**:
```bash
# Find platform-specific implementation
ls .src/gio-plugins/webviewer/webview/webview_*.go

# Read macOS/iOS implementation
cat .src/gio-plugins/webviewer/webview/webview_darwin.go
```

**Finding Event Definitions**:
```bash
# Search for event types
grep -r "type.*Event struct" .src/gio-plugins/webviewer/
```

**Understanding Operations**:
```bash
# Find all operations
grep -r "type.*Op struct" .src/gio-plugins/webviewer/giowebview/
```

## Tips for AI Agents

1. **Always check .src/gio-plugins/** before asking about plugin behavior
2. **Platform differences** are in `*_platform.go` files
3. **Demo app** (`demo/demo.go`) shows best practices
4. **Events** are async - handle them in event loop
5. **Tags** must be unique per webview instance
6. **Context (gtx)** is required for all operations

## Common Pitfalls

1. **Forgetting to hijack events**: Use `gioplugins.Hijack(window)` at start
2. **Reusing tags**: Each webview needs a unique tag
3. **Not handling events**: Events will queue up if not consumed
4. **Platform assumptions**: Features may work differently per platform
5. **WASM support**: Not all plugins work in WebAssembly builds

## Platform-Specific Notes

### macOS/iOS
- Uses WKWebView (modern WebKit)
- Best standards support
- Requires macOS 10.13+ / iOS 11+

### Android
- Uses Chromium-based WebView
- Check Android WebView version on device
- May need permissions in AndroidManifest.xml

### Windows
- Uses WebView2 (Edge-based)
- Requires WebView2 runtime installed
- Best Chromium compatibility

### Linux
- Uses WebKitGTK
- May need system packages installed
- Check GTK version compatibility

## Further Reading

- **Gio UI Docs**: https://gioui.org/doc
- **gio-plugins README**: `.src/gio-plugins/README.md`
- **WebView Analysis**: `docs/WEBVIEW-ANALYSIS.md`
