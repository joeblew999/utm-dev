package gitignore

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GitIgnore represents a .gitignore file manager
type GitIgnore struct {
	ProjectPath string
	FilePath    string
	Exists      bool
	Lines       []string
}

// New creates a new GitIgnore manager for the given project directory
func New(projectPath string) *GitIgnore {
	gitignorePath := filepath.Join(projectPath, ".gitignore")

	// Check if .gitignore already exists
	exists := false
	if _, err := os.Stat(gitignorePath); err == nil {
		exists = true
	}

	return &GitIgnore{
		ProjectPath: projectPath,
		FilePath:    gitignorePath,
		Exists:      exists,
		Lines:       []string{},
	}
}

// Load reads the existing .gitignore file if it exists
func (g *GitIgnore) Load() error {
	if !g.Exists {
		return nil // Nothing to load
	}

	file, err := os.Open(g.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open .gitignore: %w", err)
	}
	defer file.Close()

	g.Lines = []string{}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		g.Lines = append(g.Lines, scanner.Text())
	}

	return scanner.Err()
}

// HasPattern checks if a pattern already exists in the .gitignore
func (g *GitIgnore) HasPattern(pattern string) bool {
	cleanPattern := strings.TrimSpace(pattern)

	for _, line := range g.Lines {
		if strings.TrimSpace(line) == cleanPattern {
			return true
		}
	}

	return false
}

// HasManagedSection checks if utm-dev managed section exists
func (g *GitIgnore) HasManagedSection() bool {
	for _, line := range g.Lines {
		if strings.Contains(line, "# utm-dev managed") {
			return true
		}
	}
	return false
}

// Info returns basic information about the .gitignore file
func (g *GitIgnore) Info() map[string]interface{} {
	info := map[string]interface{}{
		"exists":          g.Exists,
		"path":            g.FilePath,
		"lines":           len(g.Lines),
		"managed_section": g.HasManagedSection(),
	}

	if g.Exists {
		// Count different types of lines
		comments := 0
		patterns := 0
		empty := 0

		for _, line := range g.Lines {
			trimmed := strings.TrimSpace(line)
			switch {
			case trimmed == "":
				empty++
			case strings.HasPrefix(trimmed, "#"):
				comments++
			default:
				patterns++
			}
		}

		info["comments"] = comments
		info["patterns"] = patterns
		info["empty_lines"] = empty
	}

	return info
}

// String returns a summary of the .gitignore status
func (g *GitIgnore) String() string {
	if !g.Exists {
		return fmt.Sprintf(".gitignore: not found at %s", g.FilePath)
	}

	info := g.Info()
	return fmt.Sprintf(".gitignore: %d lines (%d patterns, %d comments) at %s",
		info["lines"], info["patterns"], info["comments"], g.FilePath)
}
