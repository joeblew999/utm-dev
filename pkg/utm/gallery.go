package utm

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed vm-gallery.json
var vmGalleryJSON []byte

// VMGallery represents the VM gallery configuration
type VMGallery struct {
	Meta GalleryMeta        `json:"meta"`
	VMs  map[string]VMEntry `json:"vms"`
}

// GalleryMeta contains metadata about the gallery
type GalleryMeta struct {
	SchemaVersion string       `json:"schemaVersion"`
	Paths         GalleryPaths `json:"paths"`
	UTMApp        UTMAppConfig `json:"utmApp"`
}

// UTMAppConfig contains UTM application download info
type UTMAppConfig struct {
	Version  string `json:"version"`
	URL      string `json:"url"`
	Checksum string `json:"checksum,omitempty"`
	MinMacOS string `json:"minMacOS,omitempty"`
}

// GalleryPaths defines default paths (can be overridden)
type GalleryPaths struct {
	App   string `json:"app"`
	VMs   string `json:"vms"`
	ISO   string `json:"iso"`
	Share string `json:"share"`
}

// VMEntry represents a VM in the gallery
type VMEntry struct {
	// Display name for the VM
	Name string `json:"name"`

	// Description of the VM
	Description string `json:"description,omitempty"`

	// Architecture (arm64, amd64)
	Arch string `json:"arch"`

	// OS type (windows, linux, macos)
	OS string `json:"os"`

	// ISO download configuration (set for ISO-based installs)
	ISO ISOConfig `json:"iso,omitempty"`

	// Box download configuration (set for pre-built .utm box installs)
	// When Box is set, ISO-based create is skipped — box is imported directly.
	Box *BoxConfig `json:"box,omitempty"`

	// UTM template configuration
	Template TemplateConfig `json:"template"`

	// Tags for filtering
	Tags []string `json:"tags,omitempty"`
}

// IsBoxBased returns true when this VM uses a pre-built box instead of ISO install
func (v *VMEntry) IsBoxBased() bool {
	return v.Box != nil && v.Box.Name != ""
}

// ISOConfig contains ISO download information
type ISOConfig struct {
	// URL to download the ISO
	URL string `json:"url"`

	// Expected SHA256 checksum
	Checksum string `json:"checksum,omitempty"`

	// Filename to save as
	Filename string `json:"filename"`

	// Size in bytes (for display)
	Size int64 `json:"size,omitempty"`
}

// BoxConfig contains a pre-built .utm box download (HCP Vagrant registry format)
// The box is a tar archive containing a .utm bundle — no OS install needed.
type BoxConfig struct {
	// HCP Vagrant registry namespace/name e.g. "utm/windows-11"
	Name string `json:"name"`

	// Architecture e.g. "arm64"
	Arch string `json:"arch,omitempty"`

	// Expected SHA256 checksum of the .box file (empty = skip verify)
	Checksum string `json:"checksum,omitempty"`

	// Approximate size in bytes (for display)
	Size int64 `json:"size,omitempty"`
}

// TemplateConfig contains UTM VM template settings
type TemplateConfig struct {
	// UTM template type (windows-arm64, linux-arm64, etc.)
	Type string `json:"type"`

	// RAM in MB
	RAM int `json:"ram"`

	// Disk size in MB
	Disk int `json:"disk"`

	// Number of CPU cores
	CPU int `json:"cpu"`

	// Enable GPU acceleration
	GPU bool `json:"gpu,omitempty"`

	// Enable shared directory
	SharedDir bool `json:"sharedDir"`
}

// LoadGallery loads the VM gallery from embedded JSON
func LoadGallery() (*VMGallery, error) {
	var gallery VMGallery
	if err := json.Unmarshal(vmGalleryJSON, &gallery); err != nil {
		return nil, fmt.Errorf("failed to parse VM gallery: %w", err)
	}
	return &gallery, nil
}

// GetVM returns a VM entry by its key
func (g *VMGallery) GetVM(key string) (*VMEntry, bool) {
	vm, ok := g.VMs[key]
	if !ok {
		return nil, false
	}
	return &vm, true
}

// ListVMs returns all VM keys in the gallery
func (g *VMGallery) ListVMs() []string {
	keys := make([]string, 0, len(g.VMs))
	for k := range g.VMs {
		keys = append(keys, k)
	}
	return keys
}

// FilterByOS returns VMs matching the given OS
func (g *VMGallery) FilterByOS(os string) map[string]VMEntry {
	result := make(map[string]VMEntry)
	for k, v := range g.VMs {
		if v.OS == os {
			result[k] = v
		}
	}
	return result
}

// FilterByArch returns VMs matching the given architecture
func (g *VMGallery) FilterByArch(arch string) map[string]VMEntry {
	result := make(map[string]VMEntry)
	for k, v := range g.VMs {
		if v.Arch == arch {
			result[k] = v
		}
	}
	return result
}

// FilterByTag returns VMs that have the given tag
func (g *VMGallery) FilterByTag(tag string) map[string]VMEntry {
	result := make(map[string]VMEntry)
	for k, v := range g.VMs {
		for _, t := range v.Tags {
			if t == tag {
				result[k] = v
				break
			}
		}
	}
	return result
}
