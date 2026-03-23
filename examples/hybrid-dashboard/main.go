package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/op"
	"gioui.org/text"
	"gioui.org/widget/material"
	"github.com/gioui-plugins/gio-plugins/plugin/gioplugins"
	"github.com/gioui-plugins/gio-plugins/webviewer/giowebview"
	"github.com/gioui-plugins/gio-plugins/webviewer/webview"
)

//go:embed web/*
var webContent embed.FS

// SystemStats represents system information exposed to JavaScript
type SystemStats struct {
	Platform    string  `json:"platform"`
	GoVersion   string  `json:"goVersion"`
	CPUUsage    float64 `json:"cpuUsage"`
	MemoryUsage float64 `json:"memoryUsage"`
	Uptime      int64   `json:"uptime"`
}

// DeepLinkInfo represents information about an incoming deep link
type DeepLinkInfo struct {
	OriginalURL string `json:"originalUrl"`
	Scheme      string `json:"scheme"`
	Host        string `json:"host"`
	Path        string `json:"path"`
	Query       string `json:"query"`
	Fragment    string `json:"fragment"`
	ReceivedAt  string `json:"receivedAt"`
}

var (
	startTime     = time.Now()
	th            = material.NewTheme()
	pendingURLs   = make(chan string, 10) // Channel for deep link URLs
	lastDeepLink  *DeepLinkInfo           // Most recent deep link for API
)

func main() {
	th.Shaper = text.NewShaper(text.WithCollection(gofont.Collection()))

	// Enable webview debug mode
	webview.SetDebug(true)

	// Start embedded HTTP server
	serverURL := startWebServer()
	fmt.Printf("Web server started at %s\n", serverURL)

	// Launch Gio UI app
	go runApp(serverURL)
	app.Main()
}

func startWebServer() string {
	// Find available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	// Serve embedded web content
	webFS, err := fs.Sub(webContent, "web")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(webFS)))
	
	// API endpoint: Get system stats (called from JavaScript)
	mux.HandleFunc("/api/stats", handleStats)
	
	// API endpoint: Say hello from Go
	mux.HandleFunc("/api/hello", handleHello)

	// API endpoint: Get last deep link (for JavaScript to poll)
	mux.HandleFunc("/api/deeplink", handleDeepLink)

	serverAddr := fmt.Sprintf("127.0.0.1:%d", port)
	go func() {
		log.Printf("HTTP server listening on http://%s\n", serverAddr)
		if err := http.ListenAndServe(serverAddr, mux); err != nil {
			log.Fatal(err)
		}
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	// Verify server is ready
	serverURL := fmt.Sprintf("http://%s", serverAddr)
	for i := 0; i < 10; i++ {
		resp, err := http.Get(serverURL)
		if err == nil {
			resp.Body.Close()
			log.Printf("Server verified ready at %s", serverURL)
			break
		}
		log.Printf("Waiting for server... attempt %d", i+1)
		time.Sleep(100 * time.Millisecond)
	}

	return serverURL
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	stats := SystemStats{
		Platform:    runtime.GOOS,
		GoVersion:   runtime.Version(),
		CPUUsage:    rand.Float64() * 100,      // Simulated
		MemoryUsage: 50 + rand.Float64()*40,    // Simulated
		Uptime:      time.Since(startTime).Milliseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func handleHello(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{
		"message": "Hello from Go! 🚀",
		"time":    time.Now().Format(time.RFC3339),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleDeepLink(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if lastDeepLink == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"hasDeepLink": false,
			"message":     "No deep link received yet. Try opening: hybrid://dashboard/stats",
		})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{
		"hasDeepLink": true,
		"deepLink":    lastDeepLink,
	})
}

// parseDeepLink parses a URL string into DeepLinkInfo
func parseDeepLink(rawURL string) *DeepLinkInfo {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		log.Printf("Failed to parse deep link URL: %v", err)
		return nil
	}

	return &DeepLinkInfo{
		OriginalURL: rawURL,
		Scheme:      parsed.Scheme,
		Host:        parsed.Host,
		Path:        parsed.Path,
		Query:       parsed.RawQuery,
		Fragment:    parsed.Fragment,
		ReceivedAt:  time.Now().Format(time.RFC3339),
	}
}

// mapDeepLinkToWebPath converts a deep link path to a web app path
// Examples:
//   hybrid://dashboard/stats  -> /#/stats
//   hybrid://dashboard/hello  -> /#/hello
//   hybrid://open?url=https://example.com -> navigates to external URL
func mapDeepLinkToWebPath(info *DeepLinkInfo, baseURL string) string {
	// Handle special "open" command for external URLs
	if info.Host == "open" && info.Query != "" {
		values, err := url.ParseQuery(info.Query)
		if err == nil {
			if externalURL := values.Get("url"); externalURL != "" {
				return externalURL
			}
		}
	}

	// Map deep link paths to web app hash routes
	path := strings.TrimPrefix(info.Path, "/")
	if path == "" {
		path = "home"
	}

	return fmt.Sprintf("%s/#/%s", baseURL, path)
}

func runApp(serverURL string) {
	window := &app.Window{}
	window.Option(app.Title("Hybrid Dashboard - Gio + WebView"))
	window.Option(app.Size(1200, 800))

	var ops op.Ops
	webviewTag := new(int)
	navigated := false
	frameCount := 0
	pendingNavigation := "" // URL to navigate to (from deep link)

	// Trigger initial frame
	window.Invalidate()

	for {
		evt := gioplugins.Hijack(window)

		switch evt := evt.(type) {
		case app.DestroyEvent:
			os.Exit(0)
			return

		// Handle deep link URLs (app.URLEvent)
		// This is triggered when the app is opened via a custom URL scheme
		// e.g., hybrid://dashboard/stats or https://example.com/app/path
		case app.URLEvent:
			rawURL := evt.URL.String()
			log.Printf("Deep link received: %s", rawURL)

			// Create DeepLinkInfo directly from the parsed URL
			info := &DeepLinkInfo{
				OriginalURL: rawURL,
				Scheme:      evt.URL.Scheme,
				Host:        evt.URL.Host,
				Path:        evt.URL.Path,
				Query:       evt.URL.RawQuery,
				Fragment:    evt.URL.Fragment,
				ReceivedAt:  time.Now().Format(time.RFC3339),
			}
			lastDeepLink = info
			log.Printf("Parsed deep link - Scheme: %s, Host: %s, Path: %s",
				info.Scheme, info.Host, info.Path)

			// Map to web app path and queue navigation
			pendingNavigation = mapDeepLinkToWebPath(info, serverURL)
			log.Printf("Will navigate to: %s", pendingNavigation)

			// Request a frame to process the navigation
			window.Invalidate()

		case app.FrameEvent:
			gtx := app.NewContext(&ops, evt)

			// Process webview events
			for {
				ev, ok := gioplugins.Event(gtx, giowebview.Filter{Target: webviewTag})
				if !ok {
					break
				}
				switch e := ev.(type) {
				case giowebview.NavigationEvent:
					log.Printf("Navigation event: %s", e.URL)
				case giowebview.TitleEvent:
					log.Printf("Title event: %s", e.Title)
				}
			}

			// Render WebView FIRST - fills entire window
			// Must render before navigation commands work
			webviewStack := giowebview.WebViewOp{Tag: webviewTag}.Push(gtx.Ops)
			giowebview.OffsetOp{Point: f32.Point{X: 0, Y: 0}}.Add(gtx.Ops)
			giowebview.RectOp{
				Size: f32.Point{
					X: float32(gtx.Constraints.Max.X),
					Y: float32(gtx.Constraints.Max.Y),
				},
			}.Add(gtx.Ops)
			webviewStack.Pop(gtx.Ops)

			// Navigate after many frames to ensure webview is fully initialized
			frameCount++
			if !navigated && frameCount > 10 {
				log.Printf("Initial navigation to: %s (frame %d)", serverURL, frameCount)
				gioplugins.Execute(gtx, giowebview.NavigateCmd{
					View: webviewTag,
					URL:  serverURL,
				})
				navigated = true
			}

			// Handle pending deep link navigation
			if pendingNavigation != "" {
				log.Printf("Navigating webview to: %s", pendingNavigation)
				gioplugins.Execute(gtx, giowebview.NavigateCmd{
					View: webviewTag,
					URL:  pendingNavigation,
				})
				pendingNavigation = ""
			}

			// Request more frames until navigated
			if !navigated {
				gtx.Execute(op.InvalidateCmd{})
			}

			evt.Frame(gtx.Ops)
		}
	}
}
