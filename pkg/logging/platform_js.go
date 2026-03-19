//go:build js

package logging

import (
	"runtime"

	"github.com/inkeliz/gioismobile"
)

// DetectPlatform returns the current runtime platform.
// On WASM builds, gioismobile detects whether the browser is on
// a mobile device (Android, iOS) vs desktop (macOS, Windows).
// runtime.GOOS is just "js" which doesn't tell you the real OS.
func DetectPlatform() Platform {
	p := Platform{
		OS:       string(gioismobile.OS()),
		Arch:     runtime.GOARCH,
		IsMobile: gioismobile.IsMobile(),
	}
	// Fallback: if gioismobile can't detect, use runtime.GOOS
	if p.OS == "" {
		p.OS = runtime.GOOS
	}
	return p
}
