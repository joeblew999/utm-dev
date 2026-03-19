package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"gioui.org/font"
	"github.com/gioui-plugins/gio-plugins/plugin/gioplugins"
	"github.com/gioui-plugins/gio-plugins/webviewer/giowebview"
	"github.com/joeblew999/goup-util/pkg/logging"

	"golang.org/x/exp/shiny/materialdesign/icons"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/font/gofont"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"github.com/gioui-plugins/gio-plugins/webviewer/webview"
)

var (
	GlobalShaper = text.NewShaper(text.WithCollection(gofont.Collection()))
	DefaultURL   = "https://google.com"

	IconAdd, _            = widget.NewIcon(icons.ContentAdd)
	IconClose, _          = widget.NewIcon(icons.NavigationClose)
	IconGo, _             = widget.NewIcon(icons.NavigationArrowForward)
	IconCookie, _         = widget.NewIcon(icons.ContentArchive)
	IconLocalStorage, _   = widget.NewIcon(icons.DeviceStorage)
	IconSessionStorage, _ = widget.NewIcon(icons.ImageTimer)
	IconJavascript, _     = widget.NewIcon(icons.AVPlayArrow)
)

//go:embed app.json
var embeddedConfig []byte

// appConfig defines the runtime configuration loaded from app.json.
type appConfig struct {
	URL    string       `json:"url"`
	Name   string       `json:"name,omitempty"`
	Width  int          `json:"width,omitempty"`
	Height int          `json:"height,omitempty"`
	Update updateConfig `json:"update,omitempty"`
}

// updateConfig tells the shell where to find updates on GitHub.
type updateConfig struct {
	Repo  string `json:"repo"`  // GitHub owner/repo (e.g. "joeblew999/goup-util")
	Asset string `json:"asset"` // Asset name prefix (e.g. "webviewer-shell")
}

// loadAppConfig tries to load app.json from the executable's directory first,
// then the current working directory. Returns defaults if not found.
func loadAppConfig() *appConfig {
	cfg := &appConfig{
		URL:    "https://google.com",
		Name:   "Gio WebViewer",
		Width:  1200,
		Height: 800,
	}

	// Try executable directory first (for pre-built shell binaries)
	if exePath, err := os.Executable(); err == nil {
		if data, err := os.ReadFile(filepath.Join(filepath.Dir(exePath), "app.json")); err == nil {
			json.Unmarshal(data, cfg)
			return cfg
		}
	}

	// Try current working directory
	if data, err := os.ReadFile("app.json"); err == nil {
		json.Unmarshal(data, cfg)
		return cfg
	}

	// Fallback: use embedded app.json (works on all platforms including Android)
	if len(embeddedConfig) > 0 {
		json.Unmarshal(embeddedConfig, cfg)
	}

	return cfg
}

// selfUpdate downloads the latest release asset and replaces the current binary.
// Only works on desktop (macOS, Windows, Linux). Mobile uses app stores.
func selfUpdate(cfg *appConfig) error {
	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		// OK — desktop can self-update
	default:
		return fmt.Errorf("self-update not supported on %s (use app store)", runtime.GOOS)
	}
	if cfg.Update.Repo == "" || cfg.Update.Asset == "" {
		return fmt.Errorf("update not configured in app.json (need update.repo and update.asset)")
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", cfg.Update.Repo)
	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("failed to fetch release info: %s", resp.Status)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("failed to parse release info: %w", err)
	}

	// Find matching asset: e.g. "webviewer-shell-macos.zip" for asset prefix "webviewer-shell"
	// Match by prefix + current OS
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "macos"
	}
	wantPrefix := fmt.Sprintf("%s-%s", cfg.Update.Asset, osName)

	var downloadURL, assetName string
	for _, a := range release.Assets {
		if len(a.Name) >= len(wantPrefix) && a.Name[:len(wantPrefix)] == wantPrefix {
			downloadURL = a.BrowserDownloadURL
			assetName = a.Name
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no matching asset for %s in release %s", wantPrefix, release.TagName)
	}

	fmt.Printf("Downloading %s (%s)...\n", assetName, release.TagName)

	// Download to temp file
	tmpFile, err := os.CreateTemp("", "webviewer-update-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	dlResp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != 200 {
		return fmt.Errorf("download failed: %s", dlResp.Status)
	}

	if _, err := io.Copy(tmpFile, dlResp.Body); err != nil {
		return fmt.Errorf("failed to save download: %w", err)
	}
	tmpFile.Close()

	// Get current executable path
	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}
	exePath, _ = filepath.EvalSymlinks(exePath)
	exeDir := filepath.Dir(exePath)

	// Unzip the downloaded archive into the executable's directory
	if err := unzipUpdate(tmpFile.Name(), exeDir); err != nil {
		return fmt.Errorf("failed to extract update: %w", err)
	}

	fmt.Printf("Updated to %s\n", release.TagName)
	return nil
}

// checkForUpdate quietly checks GitHub for a newer release and prints a notice.
// Runs in a goroutine so it never blocks app startup.
func checkForUpdate(cfg *appConfig) {
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", cfg.Update.Repo)
	resp, err := http.Get(apiURL)
	if err != nil {
		return // silently ignore network errors
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return
	}

	if release.TagName != "" {
		fmt.Printf("[update] Latest release: %s — run with --update to install\n", release.TagName)
	}
}

// unzipUpdate extracts a zip file to the destination directory.
func unzipUpdate(zipPath, destDir string) error {
	// Use system unzip/tar — keeps it simple and handles .app bundles correctly
	switch runtime.GOOS {
	case "darwin", "linux":
		cmd := exec.Command("unzip", "-o", zipPath, "-d", destDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	case "windows":
		cmd := exec.Command("powershell", "-Command",
			fmt.Sprintf("Expand-Archive -Force -Path '%s' -DestinationPath '%s'", zipPath, destDir))
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

func main() {
	proxy := flag.String("proxy", "", "proxy")
	update := flag.Bool("update", false, "self-update from GitHub releases")
	if proxy != nil && *proxy != "" {
		u, err := url.Parse(*proxy)
		if err != nil {
			panic(err)
		}
		if err := webview.SetProxy(u); err != nil {
			panic(err)
		}
	}
	flag.Parse()

	// Load config from app.json (if present)
	cfg := loadAppConfig()

	// Validate URL for non-dev users
	if cfg.URL == "" {
		fmt.Fprintln(os.Stderr, "ERROR: No URL configured. Edit app.json and set \"url\" to your website address.")
		fmt.Fprintln(os.Stderr, "Example: {\"url\": \"https://your-website.com\", \"name\": \"My App\"}")
		os.Exit(1)
	}
	if u, err := url.Parse(cfg.URL); err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		fmt.Fprintf(os.Stderr, "ERROR: Invalid URL in app.json: %q\n", cfg.URL)
		fmt.Fprintln(os.Stderr, "URL must start with http:// or https://")
		fmt.Fprintln(os.Stderr, "Example: {\"url\": \"https://your-website.com\"}")
		os.Exit(1)
	}

	// Handle --update flag
	if *update {
		if err := selfUpdate(cfg); err != nil {
			fmt.Fprintf(os.Stderr, "Update failed: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	DefaultURL = cfg.URL
	fmt.Printf("Loading %s (%s)\n", cfg.Name, cfg.URL)

	// Initialize structured logger (APP role — this is a user app)
	log, err := logging.New(logging.Config{
		AppName: cfg.Name,
		Role:    logging.RoleApp,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: logging init failed: %v\n", err)
	}
	defer log.Close()

	plat := logging.DetectPlatform()
	log.Info("app starting",
		"url", cfg.URL,
		"platform", plat.DisplayName(),
		"canSelfUpdate", plat.CanSelfUpdate(),
	)

	// Check for updates in the background (non-blocking)
	if cfg.Update.Repo != "" && cfg.Update.Asset != "" {
		go checkForUpdate(cfg)
	}

	webview.SetDebug(true)
	window := &app.Window{}
	window.Option(app.Title(cfg.Name))
	window.Option(app.Size(unit.Dp(cfg.Width), unit.Dp(cfg.Height)))

	browsers := NewBrowser()
	browsers.add()
	browsers.InitialURL = DefaultURL
	browsers.Address[0].SetText(DefaultURL)
	browsers.window = window
	browsers.log = log

	go func() {
		ops := new(op.Ops)
		for {
			evt := gioplugins.Hijack(window)

			switch evt := evt.(type) {
			case app.DestroyEvent:
				log.Event("app_exit")
				os.Exit(0)
				return
			case app.FrameEvent:
				gtx := app.NewContext(ops, evt)
				browsers.Layout(gtx)
				evt.Frame(ops)
			}
		}
	}()

	app.Main()
}

const (
	VisibleLocal = 1 << iota
	VisibleSession
	VisibleCookies
)

type Browsers struct {
	Selected int

	Go    widget.Clickable
	Add   widget.Clickable
	Close widget.Clickable

	JavascriptCode widget.Editor
	JavascriptRun  widget.Clickable

	Tabs    []widget.Clickable
	Address []widget.Editor

	Tags   []*int
	Titles []string

	LocalStorage   [][]webview.StorageData
	SessionStorage [][]webview.StorageData
	CookieStorage  [][]webview.CookieData

	StorageVisible uint8

	LocalButton   widget.Clickable
	SessionButton widget.Clickable
	CookieButton  widget.Clickable

	HeaderFlex []layout.FlexChild
	TabsFlex   []layout.FlexChild

	// InitialURL is navigated to automatically on first Layout.
	InitialURL   string
	navigateSent bool            // NavigateCmd has been sent
	startTime    time.Time       // when first frame was rendered
	window       *app.Window     // for calling Invalidate directly
	log          *logging.Logger // structured runtime logger
}

func NewBrowser() *Browsers {
	b := &Browsers{}
	b.HeaderFlex = []layout.FlexChild{
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			defer clip.Outline{Path: clip.Rect{Max: gtx.Constraints.Max}.Path()}.Op().Push(gtx.Ops).Pop()
			paint.ColorOp{Color: color.NRGBA{R: 24, G: 26, B: 33, A: 255}}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)

			gtx.Constraints.Min.Y = 0
			gtx.Constraints.Max.X -= gtx.Dp(16)
			macro := op.Record(gtx.Ops)

			textMaterial := op.Record(gtx.Ops)
			paint.ColorOp{Color: color.NRGBA{R: 255, G: 255, B: 255, A: 255}}.Add(gtx.Ops)
			tmat := textMaterial.Stop()

			selectMaterial := op.Record(gtx.Ops)
			paint.ColorOp{Color: color.NRGBA{R: 123, G: 123, B: 123, A: 255}}.Add(gtx.Ops)
			smat := selectMaterial.Stop()

			dims := b.Address[b.Selected].Layout(gtx, GlobalShaper, font.Font{}, gtx.Metric.DpToSp(16), tmat, smat)
			call := macro.Stop()

			defer op.Offset(image.Point{X: gtx.Dp(8), Y: (gtx.Constraints.Max.Y - dims.Size.Y - dims.Baseline) / 2}).Push(gtx.Ops).Pop()
			call.Add(gtx.Ops)

			gtx.Constraints.Max.X += gtx.Dp(16)

			return layout.Dimensions{Size: gtx.Constraints.Max}
		}),
		layout.Rigid(layout.Spacer{Width: 4}.Layout),
		layout.Rigid(Button{Clickable: &b.Go, Icon: IconGo, Text: "Go"}.Layout),
		layout.Rigid(layout.Spacer{Width: 4}.Layout),
		layout.Rigid(Button{Clickable: &b.Close, Icon: IconClose, Text: "Close"}.Layout),
		layout.Rigid(layout.Spacer{Width: 4}.Layout),
		layout.Rigid(Button{Clickable: &b.Add, Icon: IconAdd, Text: "Add"}.Layout),
		layout.Rigid(layout.Spacer{Width: 4}.Layout),
		layout.Rigid(Button{Clickable: &b.CookieButton, Icon: IconCookie}.Layout),
		layout.Rigid(layout.Spacer{Width: 4}.Layout),
		layout.Rigid(Button{Clickable: &b.LocalButton, Icon: IconLocalStorage}.Layout),
		layout.Rigid(layout.Spacer{Width: 4}.Layout),
		layout.Rigid(Button{Clickable: &b.SessionButton, Icon: IconSessionStorage}.Layout),
	}
	return b
}

func (b *Browsers) add() {
	b.Tabs = append(b.Tabs, widget.Clickable{})
	b.Tags = append(b.Tags, new(int))
	b.Titles = append(b.Titles, "")
	b.Address = append(b.Address, widget.Editor{SingleLine: true, Submit: true})
	b.LocalStorage = append(b.LocalStorage, nil)
	b.SessionStorage = append(b.SessionStorage, nil)
	b.CookieStorage = append(b.CookieStorage, nil)

	if cap(b.TabsFlex) < len(b.Tabs) {
		b.TabsFlex = make([]layout.FlexChild, len(b.Tabs))
	} else {
		b.TabsFlex = b.TabsFlex[:len(b.Tabs)]
	}
}

func (b *Browsers) remove(i int) {
	if len(b.Tabs) == 1 {
		return
	}
	if b.Selected >= len(b.Tabs)-1 {
		b.Selected--
	}
	b.Tabs = append(b.Tabs[:i], b.Tabs[i+1:]...)
	b.Tags = append(b.Tags[:i], b.Tags[i+1:]...)
	b.Titles = append(b.Titles[:i], b.Titles[i+1:]...)
	b.TabsFlex = append(b.TabsFlex[:i], b.TabsFlex[i+1:]...)
	b.Address = append(b.Address[:i], b.Address[i+1:]...)
	b.SessionStorage = append(b.SessionStorage[:i], b.SessionStorage[i+1:]...)
	b.LocalStorage = append(b.LocalStorage[:i], b.LocalStorage[i+1:]...)
	b.CookieStorage = append(b.CookieStorage[:i], b.CookieStorage[i+1:]...)
}

func (b *Browsers) Layout(gtx layout.Context) layout.Dimensions {
	if b.startTime.IsZero() {
		b.startTime = time.Now()
	}

	if b.Add.Clicked(gtx) {
		b.add()
	}
	if b.Close.Clicked(gtx) {
		b.remove(b.Selected)
	}

	currentStoragePanel := b.StorageVisible
	if b.LocalButton.Clicked(gtx) {
		currentStoragePanel ^= VisibleLocal
	}
	if b.SessionButton.Clicked(gtx) {
		currentStoragePanel ^= VisibleSession
	}
	if b.CookieButton.Clicked(gtx) {
		currentStoragePanel ^= VisibleCookies
	}
	b.StorageVisible = currentStoragePanel

	submittedIndex := -1
	if b.Go.Clicked(gtx) {
		submittedIndex = b.Selected
	}

	for i := range b.Address {
		submited := i == submittedIndex

		for {
			evt, ok := b.Address[i].Update(gtx)
			if !ok {
				break
			}
			switch evt.(type) {
			case widget.SubmitEvent:
				submited = true
			}
		}

		if submited {
			gioplugins.Execute(gtx, giowebview.NavigateCmd{View: b.Tags[i], URL: b.Address[i].Text()})
		}
	}

	// Auto-navigate: keep generating frames until we've waited long enough
	// for the native WKWebView to be created on the main thread, then send
	// NavigateCmd once. window.Invalidate() from within Layout generates the
	// next frame (goroutine-based Invalidate doesn't work with gioplugins).
	if b.InitialURL != "" && !b.navigateSent {
		if b.window != nil {
			b.window.Invalidate()
		}
		if time.Since(b.startTime) > 2*time.Second {
			gioplugins.Execute(gtx, giowebview.NavigateCmd{View: b.Tags[0], URL: b.InitialURL})
			b.navigateSent = true
		}
	}

	for i, t := range b.Tabs {
		if t.Clicked(gtx) {
			b.Selected = i
		}
	}

	for i := range b.Tags {
		for {
			evt, ok := gioplugins.Event(gtx, giowebview.Filter{Target: b.Tags[i]})
			if !ok {
				break
			}

			switch evt := evt.(type) {
			case giowebview.TitleEvent:
				b.Titles[i] = evt.Title
				b.log.Event("title", "tab", i, "title", evt.Title)
			case giowebview.NavigationEvent:
				b.Address[i].SetText(evt.URL)
				b.log.Event("navigate", "tab", i, "url", evt.URL)
			case giowebview.CookiesEvent:
				b.log.Event("cookies", "tab", i, "count", len(evt.Cookies))
			case giowebview.StorageEvent:
				b.log.Event("storage", "tab", i, "count", len(evt.Storage))
			case giowebview.MessageEvent:
				b.log.Event("message", "tab", i, "msg", evt.Message)
			}
		}
	}

	gtxi := gtx
	return Rows{}.Layout(gtx, 4, func(i int, gtx layout.Context) layout.Dimensions {
		switch i {
		case 0:
			gtx.Constraints.Max.Y = gtx.Dp(48)
			gtx.Constraints.Min.Y = gtx.Constraints.Max.Y
			defer clip.Outline{Path: clip.Rect{Max: gtx.Constraints.Max}.Path()}.Op().Push(gtx.Ops).Pop()
			paint.ColorOp{Color: color.NRGBA{R: 48, G: 52, B: 67, A: 255}}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)

			gtx.Constraints.Max.Y = gtx.Dp(40)
			gtx.Constraints.Max.X = gtx.Constraints.Max.X - gtx.Dp(20)
			defer op.Offset(image.Point{X: gtx.Dp(20) / 2, Y: 4}).Push(gtx.Ops).Pop()

			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, b.HeaderFlex...)
		case 1:
			gtx.Constraints.Max.Y = gtx.Dp(38)

			b.TabsFlex = b.TabsFlex[:len(b.Tags)]
			for i := range b.Tags {
				i := i
				b.TabsFlex[i] = layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Min = gtx.Constraints.Max

					defer clip.Outline{Path: clip.Rect{Max: gtx.Constraints.Max}.Path()}.Op().Push(gtx.Ops).Pop()
					if b.Selected == i {
						paint.ColorOp{Color: color.NRGBA{R: 48, G: 52, B: 67, A: 255}}.Add(gtx.Ops)
					} else {
						paint.ColorOp{Color: color.NRGBA{R: 61, G: 61, B: 69, A: 255}}.Add(gtx.Ops)
					}
					paint.PaintOp{}.Add(gtx.Ops)

					return b.Tabs[i].Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						return layout.UniformInset(4).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							colorMaterial := op.Record(gtx.Ops)
							paint.ColorOp{Color: color.NRGBA{R: 255, G: 255, B: 255, A: 255}}.Add(gtx.Ops)
							pcolor := colorMaterial.Stop()

							macro := op.Record(gtx.Ops)
							gtx.Constraints.Min.Y = 0
							dims := widget.Label{Alignment: text.Start, MaxLines: 1}.Layout(gtx, GlobalShaper, font.Font{}, gtx.Metric.DpToSp(16), b.Titles[i], pcolor)
							call := macro.Stop()

							defer op.Offset(image.Point{X: 0, Y: (gtx.Constraints.Max.Y - dims.Size.Y) / 2}).Push(gtx.Ops).Pop()
							call.Add(gtx.Ops)
							return dims
						})
					})
				})
			}
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx, b.TabsFlex...)
		case 2:
			defer giowebview.WebViewOp{Tag: b.Tags[b.Selected]}.Push(gtx.Ops).Pop(gtx.Ops)
			giowebview.OffsetOp{Point: f32.Point{Y: float32(gtxi.Constraints.Max.Y - gtx.Constraints.Max.Y)}}.Add(gtx.Ops)
			giowebview.RectOp{Size: f32.Point{X: float32(gtx.Constraints.Max.X), Y: float32(gtx.Constraints.Max.Y)}}.Add(gtx.Ops)
			return layout.Dimensions{Size: gtx.Constraints.Max}
		default:
			return layout.Dimensions{}
		}
	})
}

type Rows struct {
	Size layout.Dimensions
}

func (r Rows) Layout(gtx layout.Context, n int, fn func(i int, gtx layout.Context) layout.Dimensions) layout.Dimensions {
	for i := 0; i < n; i++ {
		offset := op.Offset(image.Point{Y: r.Size.Size.Y}).Push(gtx.Ops)
		dims := fn(i, gtx)
		if dims.Size.X > r.Size.Size.X {
			r.Size.Size.X = dims.Size.X
		}
		r.Size.Size.Y += dims.Size.Y
		gtx.Constraints.Max.Y -= dims.Size.Y
		offset.Pop()
	}
	return r.Size
}

type Columns struct {
	Size layout.Dimensions
}

func (c Columns) Layout(gtx layout.Context, n int, fn func(i int, gtx layout.Context) layout.Dimensions) layout.Dimensions {
	for i := 0; i < n; i++ {
		offset := op.Offset(image.Point{X: c.Size.Size.X}).Push(gtx.Ops)
		dims := fn(i, gtx)
		if dims.Size.Y > c.Size.Size.Y {
			c.Size.Size.Y = dims.Size.Y
		}
		c.Size.Size.X += dims.Size.X
		gtx.Constraints.Max.X -= dims.Size.X
		offset.Pop()
	}
	return c.Size
}

type Button struct {
	Clickable *widget.Clickable
	Icon      *widget.Icon
	Text      string
}

func (b Button) Layout(gtx layout.Context) layout.Dimensions {
	return b.Clickable.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		macro := op.Record(gtx.Ops)

		colorMaterial := op.Record(gtx.Ops)
		c := color.NRGBA{R: 32, G: 32, B: 32, A: 255}
		paint.ColorOp{Color: c}.Add(gtx.Ops)
		pcolor := colorMaterial.Stop()

		gtx.Constraints.Min.Y = 0
		var dims layout.Dimensions
		if b.Icon == nil {
			dims = widget.Label{Alignment: text.Start, MaxLines: 1}.Layout(gtx, GlobalShaper, font.Font{}, gtx.Metric.DpToSp(16), b.Text, pcolor)
		} else {
			gtx := gtx
			gtx.Constraints.Max.Y = gtx.Sp(gtx.Metric.DpToSp(16))
			gtx.Constraints.Max.X = gtx.Constraints.Max.Y
			dims = b.Icon.Layout(gtx, c)
		}
		call := macro.Stop()

		gtx.Constraints.Max.X = dims.Size.X + gtx.Dp(16)

		defer clip.Outline{Path: clip.Rect{Max: gtx.Constraints.Max}.Path()}.Op().Push(gtx.Ops).Pop()
		paint.ColorOp{Color: color.NRGBA{R: 237, G: 237, B: 237, A: 255}}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
		pointer.CursorPointer.Add(gtx.Ops)

		defer op.Offset(image.Point{X: gtx.Dp(8), Y: (gtx.Constraints.Max.Y - dims.Size.Y) / 2}).Push(gtx.Ops).Pop()
		call.Add(gtx.Ops)

		return layout.Dimensions{Size: gtx.Constraints.Max}
	})
}

type Loading struct {
	loadingLast time.Time
	loadingDt   float32
}

func (l *Loading) Layout(gtx layout.Context) layout.Dimensions {
	gtx.Constraints.Max.X = gtx.Constraints.Max.Y

	diff := gtx.Now.Sub(l.loadingLast)
	dt := float32(math.Round(float64(diff/(time.Millisecond*32))) * 0.032)
	l.loadingDt += dt
	if l.loadingDt >= 1 {
		l.loadingDt = 0
	}
	if dt > 0 {
		l.loadingLast = gtx.Now
	}

	width := float32(gtx.Dp(4))

	radius := float32(gtx.Constraints.Max.Y / 5)
	defer op.Affine(f32.Affine2D{}.Offset(f32.Point{
		X: float32(gtx.Constraints.Max.X/2) - (radius / 2) + (width),
		Y: float32(gtx.Constraints.Max.Y/2) - (radius / 2) + (width),
	})).Push(gtx.Ops).Pop()

	rot := f32.Affine2D{}.Rotate(f32.Pt(0, 0), l.loadingDt*math.Pi*2)

	path := clip.Path{}
	path.Begin(gtx.Ops)
	path.Move(rot.Transform(f32.Pt(radius, radius)))
	path.Arc(
		rot.Transform(f32.Pt(-radius, -radius)),
		rot.Transform(f32.Pt(-radius, -radius)),
		float32((math.Pi*2)/8)*7,
	)

	defer clip.Stroke{Path: path.End(), Width: width}.Op().Push(gtx.Ops).Pop()
	paint.ColorOp{Color: color.NRGBA{R: 255, G: 255, B: 255, A: 255}}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	gtx.Execute(op.InvalidateCmd{})

	return layout.Dimensions{Size: gtx.Constraints.Max}
}
