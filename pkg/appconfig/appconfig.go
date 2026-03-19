package appconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const ConfigFileName = "app.json"

// AppConfig defines the runtime configuration for a webviewer shell app.
// Users create an app.json file with just a URL — no compilation needed.
type AppConfig struct {
	URL    string       `json:"url"`              // Website to load in the webview
	Name   string       `json:"name,omitempty"`   // Window title
	Width  int          `json:"width,omitempty"`  // Window width in dp
	Height int          `json:"height,omitempty"` // Window height in dp
	Update UpdateConfig `json:"update,omitempty"` // Self-update from GitHub releases
}

// UpdateConfig tells the app where to find updates on GitHub.
type UpdateConfig struct {
	Repo  string `json:"repo"`  // GitHub owner/repo (e.g. "joeblew999/utm-dev")
	Asset string `json:"asset"` // Asset name prefix (e.g. "webviewer-shell")
}

// Defaults returns an AppConfig with sensible default values.
func Defaults() *AppConfig {
	return &AppConfig{
		URL:    "https://google.com",
		Name:   "Gio WebViewer",
		Width:  1200,
		Height: 800,
	}
}

// Load reads app.json from the given directory.
func Load(dir string) (*AppConfig, error) {
	configPath := filepath.Join(dir, ConfigFileName)
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", ConfigFileName, err)
	}

	cfg := Defaults()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", ConfigFileName, err)
	}

	return cfg, nil
}

// LoadOrDefault loads app.json from the given directory,
// returning defaults if the file doesn't exist.
func LoadOrDefault(dir string) *AppConfig {
	cfg, err := Load(dir)
	if err != nil {
		return Defaults()
	}
	return cfg
}

// LoadFromExeOrCwd tries to load app.json from the executable's directory first,
// then falls back to the current working directory, then defaults.
// This supports pre-built shell binaries where app.json sits next to the binary.
func LoadFromExeOrCwd() *AppConfig {
	// Try executable directory first
	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		if cfg, err := Load(exeDir); err == nil {
			return cfg
		}
	}

	// Try current working directory
	if cwd, err := os.Getwd(); err == nil {
		if cfg, err := Load(cwd); err == nil {
			return cfg
		}
	}

	return Defaults()
}
