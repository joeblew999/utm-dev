# Hybrid Dashboard Example

**Complete hybrid app demonstration** showing Go + embedded web server + native WebView.

## What This Demonstrates

### Architecture

```
┌─────────────────────────────────────┐
│     Gio UI Window (Native)          │
│                                     │
│  ┌─────────────────────────────┐   │
│  │  WebView (HTML/CSS/JS)      │   │
│  │                             │   │
│  │  Served from localhost      │   │
│  └─────────────────────────────┘   │
│                                     │
│  ↕ HTTP API                        │
│                                     │
│  ┌─────────────────────────────┐   │
│  │  Go HTTP Server             │   │
│  │  (Embedded in Binary)       │   │
│  │                             │   │
│  │  //go:embed web/*           │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

### Features

- ✅ **Embedded HTTP server** - Runs on random available port
- ✅ **Web content from `//go:embed`** - All HTML/CSS/JS in binary
- ✅ **Real-time data** - Updates every second via HTTP API
- ✅ **Go ↔ JavaScript bridge** - Call Go functions from JavaScript
- ✅ **Responsive design** - Works on desktop and mobile
- ✅ **Offline-capable** - No external dependencies
- ✅ **Single binary** - Everything embedded

## Building

```bash
# Add to workspace
cd ../..  # back to utm-dev root
go work use examples/hybrid-dashboard

# Build for macOS
go run . build macos examples/hybrid-dashboard

# Build for iOS
go run . build ios examples/hybrid-dashboard

# Build for Android
go run . build android examples/hybrid-dashboard

# Launch
open examples/hybrid-dashboard/.bin/hybrid-dashboard.app
```

## How It Works

### 1. Embedded Web Content

```go
//go:embed web/*
var webContent embed.FS
```

All files in `web/` are embedded into the Go binary at compile time.

### 2. HTTP Server

```go
func startWebServer() string {
    // Find available port
    listener, err := net.Listen("tcp", "127.0.0.1:0")
    port := listener.Addr().(*net.TCPAddr).Port
    
    // Serve embedded content
    webFS, _ := fs.Sub(webContent, "web")
    http.Handle("/", http.FileServer(http.FS(webFS)))
    
    // API endpoints
    http.HandleFunc("/api/stats", handleStats)
    http.HandleFunc("/api/hello", handleHello)
    
    go http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", port), nil)
    
    return fmt.Sprintf("http://127.0.0.1:%d", port)
}
```

Server runs on `127.0.0.1` (localhost) on a random available port.

### 3. WebView Integration

```go
// Navigate WebView to embedded server
gioplugins.Execute(gtx, giowebview.NavigateCmd{
    View: webviewTag,
    URL:  serverURL,  // http://127.0.0.1:XXXXX
})
```

WebView loads from the embedded HTTP server.

### 4. Go ↔ JavaScript Communication

**JavaScript → Go** (via HTTP fetch):
```javascript
const response = await fetch('/api/stats');
const data = await response.json();
```

**Go responds**:
```go
func handleStats(w http.ResponseWriter, r *http.Request) {
    stats := SystemStats{
        Platform:    os.Getenv("GOOS"),
        CPUUsage:    getCPUUsage(),
        MemoryUsage: getMemoryUsage(),
    }
    json.NewEncoder(w).Encode(stats)
}
```

## File Structure

```
hybrid-dashboard/
├── main.go              # Gio UI + HTTP server
├── go.mod
├── icon-source.png      # App icon
├── README.md
└── web/                 # Embedded web content
    ├── index.html       # Dashboard UI
    ├── css/
    │   └── styles.css   # Styling
    └── js/
        └── app.js       # JavaScript logic
```

## Benefits of This Approach

### ✅ **Offline-Capable**
Everything is embedded in the binary. No external web server needed.

### ✅ **Portable**
Single binary contains app + web content + HTTP server.

### ✅ **Secure**
Server only binds to `127.0.0.1` (localhost). Not accessible from network.

### ✅ **Fast**
Local HTTP is extremely fast. No network latency.

### ✅ **Cross-Platform**
Same code works on macOS, iOS, Android, Windows, Linux.

### ✅ **Developer-Friendly**
- Familiar web technologies (HTML/CSS/JS)
- Go for business logic and native integrations
- Clean separation of concerns

## Use Cases

This pattern is perfect for:
- **Dashboards** - System monitoring, analytics
- **Admin Tools** - Configuration UIs, control panels
- **Dev Tools** - Code editors, debuggers, profilers
- **Content Apps** - Documentation, tutorials, e-books
- **Hybrid Apps** - Mix native + web capabilities

## Extending This Example

### Add More API Endpoints

```go
http.HandleFunc("/api/users", handleUsers)
http.HandleFunc("/api/settings", handleSettings)
```

### Add WebSocket Support

```go
import "github.com/gorilla/websocket"

http.HandleFunc("/ws", handleWebSocket)
```

### Add Database

```go
import "database/sql"
import _ "modernc.org/sqlite"

db, _ := sql.Open("sqlite", "app.db")
```

### Add Authentication

```go
http.HandleFunc("/api/login", handleLogin)
// Add JWT or session-based auth
```

## Next Steps

1. **Customize the UI** - Edit `web/index.html` and `web/css/styles.css`
2. **Add features** - Extend the API with your app logic
3. **Add native integrations** - Use Gio plugins for camera, location, etc.
4. **Deploy** - Build for your target platforms

This is **THE template** for building production hybrid apps with utm-dev! 🚀
