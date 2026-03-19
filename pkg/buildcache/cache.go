// Package buildcache provides build state tracking for idempotent builds
package buildcache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// BuildState tracks the state of a build for idempotency checks
type BuildState struct {
	Project      string    `json:"project"`
	Platform     string    `json:"platform"`
	OutputPath   string    `json:"output_path"`
	SourceHash   string    `json:"source_hash"`
	LastBuild    time.Time `json:"last_build"`
	BuildSuccess bool      `json:"build_success"`
}

// Cache manages build state
type Cache struct {
	path   string
	states map[string]*BuildState
}

// NewCache creates or loads a build cache
func NewCache(cacheFile string) (*Cache, error) {
	cache := &Cache{
		path:   cacheFile,
		states: make(map[string]*BuildState),
	}

	// Try to load existing cache
	if _, err := os.Stat(cacheFile); err == nil {
		if err := cache.load(); err != nil {
			// If cache is corrupted, start fresh
			cache.states = make(map[string]*BuildState)
		}
	}

	return cache, nil
}

// load reads the cache from disk
func (c *Cache) load() error {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return err
	}

	var states map[string]*BuildState
	if err := json.Unmarshal(data, &states); err != nil {
		return err
	}

	c.states = states
	return nil
}

// Save writes the cache to disk
func (c *Cache) Save() error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(c.path), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	data, err := json.MarshalIndent(c.states, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0644)
}

// GetState returns the build state for a project/platform combination
func (c *Cache) GetState(project, platform string) *BuildState {
	key := c.key(project, platform)
	return c.states[key]
}

// SetState records a build state
func (c *Cache) SetState(state *BuildState) {
	key := c.key(state.Project, state.Platform)
	c.states[key] = state
}

// key generates a cache key
func (c *Cache) key(project, platform string) string {
	return fmt.Sprintf("%s:%s", project, platform)
}

// NeedsRebuild checks if a rebuild is needed based on source changes
func (c *Cache) NeedsRebuild(project, platform, projectPath, outputPath string) (bool, string) {
	state := c.GetState(project, platform)
	if state == nil {
		return true, "no previous build found"
	}

	// Check if output exists
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return true, "output doesn't exist"
	}

	// Check if previous build failed
	if !state.BuildSuccess {
		return true, "previous build failed"
	}

	// Calculate current source hash
	currentHash, err := hashDirectory(projectPath)
	if err != nil {
		return true, fmt.Sprintf("can't hash sources: %v", err)
	}

	// Compare hashes
	if currentHash != state.SourceHash {
		return true, "sources changed"
	}

	// Check if output is older than sources (safety check)
	outputInfo, _ := os.Stat(outputPath)
	if outputInfo != nil && outputInfo.ModTime().Before(state.LastBuild) {
		return true, "output was modified after build"
	}

	return false, ""
}

// RecordBuild records a successful build
func (c *Cache) RecordBuild(project, platform, projectPath, outputPath string, success bool) error {
	sourceHash, err := hashDirectory(projectPath)
	if err != nil {
		sourceHash = "" // Continue even if hashing fails
	}

	state := &BuildState{
		Project:      project,
		Platform:     platform,
		OutputPath:   outputPath,
		SourceHash:   sourceHash,
		LastBuild:    time.Now(),
		BuildSuccess: success,
	}

	c.SetState(state)
	return c.Save()
}

// hashDirectory creates a hash of relevant source files in a directory
func hashDirectory(path string) (string, error) {
	h := sha256.New()

	// Walk the directory and hash relevant files
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and build artifacts
		if info.IsDir() {
			name := info.Name()
			// Skip build directories
			if name == ".bin" || name == ".build" || name == ".dist" || name == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only hash source files
		ext := filepath.Ext(filePath)
		if ext != ".go" && ext != ".mod" && ext != ".sum" && ext != ".png" && ext != ".jpg" {
			return nil
		}

		// Hash file path (relative) and modification time
		relPath, _ := filepath.Rel(path, filePath)
		h.Write([]byte(relPath))
		h.Write([]byte(info.ModTime().String()))

		// For small files, hash content too
		if info.Size() < 1024*1024 { // 1MB limit
			f, err := os.Open(filePath)
			if err != nil {
				return nil // Skip files we can't read
			}
			defer f.Close()
			io.Copy(h, f)
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// GetDefaultCachePath returns the default cache file path
func GetDefaultCachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".utm-dev", "build-cache.json")
}
