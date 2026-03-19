package logging

// Platform describes the runtime platform.
type Platform struct {
	OS       string `json:"os"`       // runtime.GOOS or gioismobile.OS() for WASM
	Arch     string `json:"arch"`     // runtime.GOARCH (arm64, amd64, wasm)
	IsMobile bool   `json:"isMobile"` // true for android, ios, or mobile browser (via gioismobile)
}

// IsDesktop returns true for macOS, Windows, Linux.
func (p Platform) IsDesktop() bool {
	switch p.OS {
	case "darwin", "windows", "linux":
		return true
	}
	return false
}

// CanSelfUpdate returns true if the platform supports self-update
// (desktop only — mobile uses app stores).
func (p Platform) CanSelfUpdate() bool {
	return p.IsDesktop()
}

// DisplayName returns a human-friendly platform name.
func (p Platform) DisplayName() string {
	switch p.OS {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	case "android":
		return "Android"
	case "ios":
		return "iOS"
	case "js":
		if p.IsMobile {
			return "Mobile Web"
		}
		return "Web"
	default:
		return p.OS
	}
}
