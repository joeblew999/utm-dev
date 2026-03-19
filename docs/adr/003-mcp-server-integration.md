# ADR-003: MCP Server Integration

## Status

**Proposed** - Transform utm-dev into an MCP server for AI assistant integration.

## Context

The Model Context Protocol (MCP) is an open protocol that standardizes how applications provide context to LLMs. The [official Go SDK](https://github.com/modelcontextprotocol/go-sdk) provides **full schema support** using Go struct tags and the `github.com/google/jsonschema-go` package.

By implementing MCP, utm-dev can be directly controlled by AI assistants like Claude.

## Go SDK Schema Support

The MCP Go SDK uses [google/jsonschema-go](https://github.com/google/jsonschema-go) for automatic schema generation:

```go
// jsonschema tag = description
// No omitempty = required field
// Enums = use Go typed constants

type Platform string

const (
    PlatformMacOS   Platform = "macos"
    PlatformIOS     Platform = "ios"
    PlatformAndroid Platform = "android"
)

type BuildInput struct {
    Platform Platform `json:"platform" jsonschema:"Target platform to build for"`
    AppDir   string   `json:"app_dir" jsonschema:"Path to application directory"`
    Force    bool     `json:"force,omitempty" jsonschema:"Force rebuild even if up-to-date"`
}

// SDK auto-generates JSON Schema and validates inputs!
mcp.AddTool(server, &mcp.Tool{Name: "build"}, HandleBuild)
```

**This means we DON'T need to extract from Cobra** - we define proper Go types with tags.

## Decision

Add MCP server mode using the official Go SDK with struct-based schemas:

```bash
utm-dev mcp serve --stdio
```

## Implementation Plan

### Phase 1: Add Dependencies

**File: `go.mod`**

```go
require (
    github.com/modelcontextprotocol/go-sdk v0.x.x
    github.com/golang-jwt/jwt/v5 v5.x.x  // For auth if needed
)
```

### Phase 2: Define Tool Input/Output Types

**File: `pkg/mcp/types.go`**

```go
package mcp

// Platform enum - typed constants for JSON Schema enum support
type Platform string

const (
    PlatformMacOS   Platform = "macos"
    PlatformIOS     Platform = "ios"
    PlatformAndroid Platform = "android"
    PlatformWindows Platform = "windows"
    PlatformLinux   Platform = "linux"
    PlatformWeb     Platform = "web"
)

// VMStatus enum
type VMStatus string

const (
    VMStatusRunning   VMStatus = "running"
    VMStatusStopped   VMStatus = "stopped"
    VMStatusSuspended VMStatus = "suspended"
)

// BuildInput - no omitempty = required, with omitempty = optional
type BuildInput struct {
    Platform Platform `json:"platform" jsonschema:"Target platform to build for"`
    AppDir   string   `json:"app_dir" jsonschema:"Path to application directory"`
    Force    bool     `json:"force,omitempty" jsonschema:"Force rebuild even if up-to-date"`
    Schemes  string   `json:"schemes,omitempty" jsonschema:"Deep linking URI schemes (comma-separated)"`
}

type BuildOutput struct {
    Success    bool   `json:"success"`
    OutputPath string `json:"output_path" jsonschema:"Path to built artifact"`
    Cached     bool   `json:"cached" jsonschema:"Whether build was cached"`
}

// UTMStartInput
type UTMStartInput struct {
    VMName string `json:"vm_name" jsonschema:"Name of the VM to start"`
}

type UTMStartOutput struct {
    Success bool     `json:"success"`
    Status  VMStatus `json:"status" jsonschema:"VM status after operation"`
}

// UTMListOutput
type UTMListOutput struct {
    VMs []VMInfo `json:"vms"`
}

type VMInfo struct {
    Name   string   `json:"name"`
    Status VMStatus `json:"status" jsonschema:"Current VM status"`
}

// IconsInput
type IconsInput struct {
    AppDir string `json:"app_dir" jsonschema:"Path to application directory"`
}

type IconsOutput struct {
    Success   bool     `json:"success"`
    Generated []string `json:"generated" jsonschema:"List of generated icon files"`
}

// InstallInput
type InstallInput struct {
    SDKName string `json:"sdk_name" jsonschema:"Name of the SDK to install"`
    Force   bool   `json:"force,omitempty" jsonschema:"Force reinstall"`
}

type InstallOutput struct {
    Success     bool   `json:"success"`
    InstallPath string `json:"install_path" jsonschema:"Installation path"`
    Version     string `json:"version" jsonschema:"Installed version"`
}
```

### Phase 3: Implement Tool Handlers

**File: `pkg/mcp/handlers.go`**

```go
package mcp

import (
    "context"

    "github.com/joeblew999/utm-dev/pkg/builder"
    "github.com/joeblew999/utm-dev/pkg/utm"
)

// HandleBuild executes the build command
func HandleBuild(ctx context.Context, input BuildInput) (BuildOutput, error) {
    // Call existing build logic
    result, err := builder.Build(input.Platform, input.AppDir, builder.Options{
        Force:   input.Force,
        Schemes: input.Schemes,
    })
    if err != nil {
        return BuildOutput{Success: false}, err
    }

    return BuildOutput{
        Success:    true,
        OutputPath: result.OutputPath,
        Cached:     result.Cached,
    }, nil
}

// HandleUTMStart starts a VM
func HandleUTMStart(ctx context.Context, input UTMStartInput) (UTMStartOutput, error) {
    err := utm.StartVM(input.VMName)
    if err != nil {
        return UTMStartOutput{Success: false}, err
    }

    status, _ := utm.GetVMStatus(input.VMName)
    return UTMStartOutput{
        Success: true,
        Status:  status,
    }, nil
}

// HandleUTMList lists all VMs
func HandleUTMList(ctx context.Context, input struct{}) (UTMListOutput, error) {
    vms, err := utm.ListVMs()
    if err != nil {
        return UTMListOutput{}, err
    }

    var vmInfos []VMInfo
    for _, vm := range vms {
        vmInfos = append(vmInfos, VMInfo{
            Name:   vm.Name,
            Status: vm.Status,
        })
    }

    return UTMListOutput{VMs: vmInfos}, nil
}

// HandleIcons generates icons
func HandleIcons(ctx context.Context, input IconsInput) (IconsOutput, error) {
    generated, err := icons.Generate(input.AppDir)
    if err != nil {
        return IconsOutput{Success: false}, err
    }

    return IconsOutput{
        Success:   true,
        Generated: generated,
    }, nil
}
```

### Phase 4: MCP Server Setup

**File: `pkg/mcp/server.go`**

```go
package mcp

import (
    "github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewServer creates an MCP server with all utm-dev tools
func NewServer() *mcp.Server {
    server := mcp.NewServer(&mcp.Implementation{
        Name:    "utm-dev",
        Version: "1.0.0",
    }, nil)

    // Register tools - SDK auto-generates schemas from struct tags!
    mcp.AddTool(server, &mcp.Tool{
        Name:        "build",
        Description: "Build a Gio application for a target platform",
    }, HandleBuild)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "utm_start",
        Description: "Start a UTM virtual machine",
    }, HandleUTMStart)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "utm_stop",
        Description: "Stop a UTM virtual machine",
    }, HandleUTMStop)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "utm_list",
        Description: "List all UTM virtual machines",
    }, HandleUTMList)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "icons",
        Description: "Generate platform-specific icons for an application",
    }, HandleIcons)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "install",
        Description: "Install an SDK (Android SDK, NDK, etc.)",
    }, HandleInstall)

    return server
}

// ServeStdio runs the MCP server on stdio
func ServeStdio(server *mcp.Server) error {
    return mcp.ServeStdio(server)
}
```

### Phase 5: Cobra Command

**File: `cmd/mcp.go`**

```go
package cmd

import (
    "github.com/joeblew999/utm-dev/pkg/mcp"
    "github.com/spf13/cobra"
)

var mcpCmd = &cobra.Command{
    Use:   "mcp",
    Short: "Run utm-dev as an MCP server",
    Long: `Start utm-dev as a Model Context Protocol server.

Tools are defined with full JSON Schema support via Go struct tags.
The SDK automatically validates inputs and generates documentation.`,
}

var mcpServeCmd = &cobra.Command{
    Use:   "serve",
    Short: "Start the MCP server",
    Example: `  # Start MCP server on stdio (for Claude Desktop)
  utm-dev mcp serve --stdio

  # With authentication
  utm-dev mcp serve --stdio --auth`,
    RunE: func(cmd *cobra.Command, args []string) error {
        server := mcp.NewServer()
        return mcp.ServeStdio(server)
    },
}

func init() {
    mcpServeCmd.Flags().Bool("stdio", true, "Use stdio transport")
    mcpCmd.AddCommand(mcpServeCmd)
    mcpCmd.GroupID = "tools"
    rootCmd.AddCommand(mcpCmd)
}
```

## Claude Desktop Integration

**File: `~/Library/Application Support/Claude/claude_desktop_config.json`**

```json
{
  "mcpServers": {
    "utm-dev": {
      "command": "utm-dev",
      "args": ["mcp", "serve", "--stdio"]
    }
  }
}
```

## Tools Summary

| Tool | Input Type | Output Type | Description |
|------|------------|-------------|-------------|
| `build` | BuildInput | BuildOutput | Build app for platform |
| `utm_start` | UTMStartInput | UTMStartOutput | Start a VM |
| `utm_stop` | UTMStopInput | UTMStopOutput | Stop a VM |
| `utm_list` | - | UTMListOutput | List all VMs |
| `icons` | IconsInput | IconsOutput | Generate icons |
| `install` | InstallInput | InstallOutput | Install SDK |

## Files to Create

| File | Description |
|------|-------------|
| `pkg/mcp/types.go` | Input/Output struct definitions with jsonschema tags |
| `pkg/mcp/handlers.go` | Tool handler functions |
| `pkg/mcp/server.go` | MCP server setup |
| `cmd/mcp.go` | Cobra command |

## Benefits of This Approach

1. **Full schema support** - Enums, required fields, descriptions all via struct tags
2. **Automatic validation** - SDK validates inputs against schema
3. **Type safety** - Go compiler catches type mismatches
4. **No extraction needed** - Define types once, SDK does the rest
5. **Output schemas** - Can validate tool outputs too

## Consequences

### Benefits
- Clean, type-safe tool definitions
- SDK handles all JSON Schema generation
- Inputs validated automatically
- Native Go patterns

### Trade-offs
- Types defined separately from Cobra commands
- Need to keep handlers in sync with CLI logic

## References

- [Official MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [MCP Go Package Docs](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp)
- [google/jsonschema-go](https://github.com/google/jsonschema-go)
- [MCP Tools Specification](https://modelcontextprotocol.io/specification/draft/server/tools)

Sources:
- [MCP Go SDK GitHub](https://github.com/modelcontextprotocol/go-sdk)
- [MCP Go SDK Package](https://pkg.go.dev/github.com/modelcontextprotocol/go-sdk/mcp)
- [MCP Tools Spec](https://modelcontextprotocol.io/specification/draft/server/tools)
