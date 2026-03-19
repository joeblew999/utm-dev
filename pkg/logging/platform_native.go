//go:build !js

package logging

import "runtime"

// DetectPlatform returns the current runtime platform.
// On native builds (not WASM), runtime.GOOS is definitive.
func DetectPlatform() Platform {
	p := Platform{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}
	switch runtime.GOOS {
	case "android", "ios":
		p.IsMobile = true
	}
	return p
}
