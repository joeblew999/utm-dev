package service

import (
	"fmt"

	"github.com/joeblew999/utm-dev/pkg/config"
	"github.com/joeblew999/utm-dev/pkg/constants"
	"github.com/joeblew999/utm-dev/pkg/gitignore"
	"github.com/joeblew999/utm-dev/pkg/icons"
	"github.com/joeblew999/utm-dev/pkg/project"
	"github.com/joeblew999/utm-dev/pkg/workspace"
)

// ServiceConfig configures service behavior
type ServiceConfig struct {
	Mode         string // "cli" or "service"
	AutoMaintain bool   // Enable automatic maintenance checks
	AutoFix      bool   // Automatically fix issues found
	Verbose      bool   // Show maintenance actions
}

// GioService provides high-level operations for Gio projects
type GioService struct {
	config ServiceConfig
}

// NewGioService creates a new service instance with default config
func NewGioService() *GioService {
	return &GioService{
		config: ServiceConfig{
			Mode:         "cli",
			AutoMaintain: false,
			AutoFix:      false,
			Verbose:      false,
		},
	}
}

// NewGioServiceWithConfig creates a new service instance with custom config
func NewGioServiceWithConfig(config ServiceConfig) *GioService {
	return &GioService{config: config}
}

// ProjectRequest represents a request to work with a project
type ProjectRequest struct {
	ProjectPath string `json:"project_path"`
	Platform    string `json:"platform,omitempty"`
}

// CreateExampleRequest represents a request to create a new example
type CreateExampleRequest struct {
	ExampleName          string `json:"example_name"`
	UpdateWorkspace      bool   `json:"update_workspace"`
	ForceWorkspaceUpdate bool   `json:"force_workspace_update"`
}

// WorkspaceRequest represents a request to manage workspace
type WorkspaceRequest struct {
	ModulePath string `json:"module_path"`
	StartPath  string `json:"start_path,omitempty"`
	Force      bool   `json:"force"`
}

// ProjectResponse represents a response from project operations
type ProjectResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	ProjectID string `json:"project_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

// LoadProject loads and validates a Gio project
func (s *GioService) LoadProject(req ProjectRequest) (*project.GioProject, error) {
	proj, err := project.NewGioProject(req.ProjectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load project: %w", err)
	}

	if err := proj.Validate(); err != nil {
		return nil, fmt.Errorf("invalid project: %w", err)
	}

	return proj, nil
}

// GenerateIcons generates icons for a project and platform
func (s *GioService) GenerateIcons(req ProjectRequest) (*ProjectResponse, error) {
	if req.Platform == "" {
		return nil, fmt.Errorf("platform is required")
	}

	// Perform health checks if enabled
	if s.config.AutoMaintain {
		actions := s.performHealthChecks(req.ProjectPath)
		if s.config.Verbose && len(actions) > 0 {
			fmt.Printf("🔧 Maintenance checks performed:\n")
			for _, action := range actions {
				fmt.Printf("  - %s\n", action)
			}
		}
	}

	proj, err := s.LoadProject(req)
	if err != nil {
		return &ProjectResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Generate icons using the project-aware interface
	err = icons.GenerateForProject(icons.ProjectConfig{
		ProjectPath: req.ProjectPath,
		Platform:    req.Platform,
	})

	if err != nil {
		return &ProjectResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to generate icons: %v", err),
		}, nil
	}

	return &ProjectResponse{
		Success:   true,
		Message:   fmt.Sprintf("Generated %s icons for project %s", req.Platform, proj.Name),
		ProjectID: proj.Name,
	}, nil
}

// GenerateTestIcon creates a test source icon for a project
func (s *GioService) GenerateTestIcon(req ProjectRequest) (*ProjectResponse, error) {
	proj, err := s.LoadProject(req)
	if err != nil {
		return &ProjectResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	paths := proj.Paths()
	if err := icons.GenerateTestIcon(paths.GetSourceIcon()); err != nil {
		return &ProjectResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to generate test icon: %v", err),
		}, nil
	}

	return &ProjectResponse{
		Success:   true,
		Message:   fmt.Sprintf("Generated test icon for project %s", proj.Name),
		ProjectID: proj.Name,
	}, nil
}

// CreateExample creates a new example project and optionally adds it to workspace
func (s *GioService) CreateExample(req CreateExampleRequest) (*ProjectResponse, error) {
	if req.ExampleName == "" {
		return nil, fmt.Errorf("example name is required")
	}

	// TODO: Create example directory structure
	// TODO: Initialize go.mod
	// TODO: Create basic main.go

	// Calculate relative path for workspace
	workspacePath := fmt.Sprintf("./modules/utm-dev/examples/%s", req.ExampleName)

	// Find and update workspace if requested
	if req.UpdateWorkspace {
		ws, err := workspace.FindWorkspace("")
		if err != nil {
			return &ProjectResponse{
				Success: false,
				Error:   fmt.Sprintf("failed to find workspace: %v", err),
			}, nil
		}

		if ws.Exists && !ws.HasModule(workspacePath) {
			if req.ForceWorkspaceUpdate {
				err = ws.AddModule(workspacePath, true)
				if err != nil {
					return &ProjectResponse{
						Success: false,
						Error:   fmt.Sprintf("failed to add module to workspace: %v", err),
					}, nil
				}
			} else {
				return &ProjectResponse{
					Success: false,
					Error:   fmt.Sprintf("module %s not in workspace (use --force to add)", workspacePath),
				}, nil
			}
		}
	}

	return &ProjectResponse{
		Success:   true,
		Message:   fmt.Sprintf("Created example %s", req.ExampleName),
		ProjectID: req.ExampleName,
	}, nil
}

// EnsureInWorkspace ensures a project is included in the workspace
func (s *GioService) EnsureInWorkspace(req WorkspaceRequest) (*ProjectResponse, error) {
	ws, err := workspace.FindWorkspace(req.StartPath)
	if err != nil {
		return &ProjectResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to find workspace: %v", err),
		}, nil
	}

	if !ws.Exists {
		return &ProjectResponse{
			Success: false,
			Error:   "no go.work file found",
		}, nil
	}

	if ws.HasModule(req.ModulePath) {
		return &ProjectResponse{
			Success: true,
			Message: fmt.Sprintf("module %s already in workspace", req.ModulePath),
		}, nil
	}

	if !req.Force {
		return &ProjectResponse{
			Success: false,
			Error:   fmt.Sprintf("module %s not in workspace (use force=true to add)", req.ModulePath),
		}, nil
	}

	err = ws.AddModule(req.ModulePath, true)
	if err != nil {
		return &ProjectResponse{
			Success: false,
			Error:   fmt.Sprintf("failed to add module to workspace: %v", err),
		}, nil
	}

	return &ProjectResponse{
		Success: true,
		Message: fmt.Sprintf("added module %s to workspace", req.ModulePath),
	}, nil
}

// ListProjects could be extended to manage multiple projects
func (s *GioService) ListProjects() ([]string, error) {
	// Placeholder for future multi-project support
	return []string{}, nil
}

// performHealthChecks runs maintenance checks and optionally fixes issues
func (s *GioService) performHealthChecks(projectPath string) []string {
	if !s.config.AutoMaintain {
		return []string{}
	}

	var actions []string

	// Ensure directories exist
	if err := config.EnsureDirectories(); err != nil {
		if s.config.Verbose {
			actions = append(actions, fmt.Sprintf("Failed to ensure directories: %v", err))
		}
	} else if s.config.Verbose {
		actions = append(actions, "Ensured cache and SDK directories exist")
	}

	// Check gitignore patterns
	gi := gitignore.New(projectPath)
	if err := gi.Load(); err == nil {
		missingPatterns := constants.CommonGitIgnorePatterns()
		for _, pattern := range missingPatterns {
			if !gi.HasPattern(pattern) {
				if s.config.Verbose {
					actions = append(actions, fmt.Sprintf("Missing .gitignore pattern: %s", pattern))
				}
				// TODO: Add pattern if AutoFix is enabled
			}
		}
	}

	// Check workspace sync
	if ws, err := workspace.FindWorkspace(projectPath); err == nil && ws.Exists {
		// TODO: Check if current project is in workspace
		// TODO: Suggest adding if missing and AutoFix is enabled
		if s.config.Verbose {
			actions = append(actions, "Checked workspace sync")
		}
	}

	// Check directory health
	if s.config.Verbose {
		dirInfo := config.GetDirectoryInfo()
		if dirInfo.CacheExists || dirInfo.SDKExists {
			actions = append(actions, fmt.Sprintf("Directory status - Cache: %t, SDK: %t",
				dirInfo.CacheExists, dirInfo.SDKExists))
		}
	}

	return actions
}
