# ADR-006: API-First CLI Architecture

## Status

**Proposed** - Use Huma + Restish for unified API/CLI with auto-generated clients.

## Context

Traditional CLI tools like utm-dev face a common problem: as they grow, they need multiple interfaces:

1. **CLI** - For developers in terminals
2. **REST API** - For automation, CI/CD, remote control
3. **SDK** - For programmatic access from other Go code
4. **MCP Server** - For AI assistant integration
5. **TUI/Web GUI** - For visual interaction

The traditional approach duplicates logic across each interface, leading to:
- Code duplication and drift
- Inconsistent behavior between interfaces
- High maintenance burden
- Documentation that gets out of sync

## Decision

Adopt an **API-first architecture** using the Huma ecosystem:

1. **Huma** - REST API framework with auto-generated OpenAPI
2. **Restish** - Auto-generates CLI from OpenAPI spec
3. **humaclient** - Auto-generates Go SDK from Huma API
4. **gocrud** - Auto-generates CRUD endpoints for data models
5. **Huma SSE** - Built-in streaming for real-time updates

```
┌─────────────────────────────────────────────────────────────────┐
│                       pkg/core/                                  │
│              (Business Logic - Single Source)                    │
└─────────────────────────────────────────────────────────────────┘
                              ↑
┌─────────────────────────────────────────────────────────────────┐
│                      Huma API Server                             │
│         (REST endpoints + SSE streaming + OpenAPI)               │
└─────────────────────────────────────────────────────────────────┘
                              │
              ┌───────────────┼───────────────┐
              │               │               │
              ▼               ▼               ▼
       ┌──────────┐    ┌──────────┐    ┌──────────┐
       │ OpenAPI  │    │   SSE    │    │  gocrud  │
       │   Spec   │    │ Streams  │    │  CRUD    │
       └────┬─────┘    └──────────┘    └──────────┘
            │
    ┌───────┼───────┬───────────────┐
    │       │       │               │
    ▼       ▼       ▼               ▼
┌───────┐ ┌─────┐ ┌─────┐    ┌───────────┐
│Restish│ │huma │ │ MCP │    │ Swagger   │
│  CLI  │ │client│ │Server│   │    UI     │
└───────┘ └─────┘ └─────┘    └───────────┘
```

## Why This Approach

### Single Source of Truth

Write business logic once in `pkg/core/`. Everything else is generated:

| Component | Source | Generated From |
|-----------|--------|----------------|
| REST API | Huma handlers | pkg/core/ calls |
| OpenAPI Spec | Auto | Huma type definitions |
| CLI | Auto | OpenAPI via Restish |
| Go SDK | Auto | Huma API via humaclient |
| MCP Tools | Semi-auto | OpenAPI schemas |
| Documentation | Auto | OpenAPI spec |

### Comparison with Traditional Approach

**Traditional (Current utm-dev)**:
```
cmd/build.go      → calls → pkg/builder/
cmd/utm.go        → calls → pkg/utm/
cmd/install.go    → calls → pkg/installer/
cmd/icons.go      → calls → pkg/icons/
... (each command manually defined)
```

**API-First**:
```
pkg/api/handlers.go → calls → pkg/core/*
                ↓
         OpenAPI spec
                ↓
    Restish auto-discovers CLI
```

### Benefits

1. **Zero CLI maintenance** - Restish reads OpenAPI, CLI updates automatically
2. **Guaranteed consistency** - CLI, API, SDK all from same source
3. **Free documentation** - OpenAPI generates Swagger UI
4. **Remote-ready** - Same API works locally and over network
5. **AI-ready** - OpenAPI schemas feed MCP tool definitions
6. **Type-safe SDK** - humaclient generates validated Go client

## Implementation Plan

### Phase 1: Core API Setup

**File: `pkg/api/server.go`**

```go
package api

import (
    "net/http"

    "github.com/danielgtaylor/huma/v2"
    "github.com/danielgtaylor/huma/v2/adapters/humachi"
    "github.com/go-chi/chi/v5"
)

func NewServer() *Server {
    router := chi.NewMux()

    api := humachi.New(router, huma.DefaultConfig("utm-dev", "1.0.0"))

    // Register all handlers
    RegisterBuildHandlers(api)
    RegisterUTMHandlers(api)
    RegisterInstallHandlers(api)
    RegisterIconsHandlers(api)

    return &Server{
        router: router,
        api:    api,
    }
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    s.router.ServeHTTP(w, r)
}
```

### Phase 2: Define API Types

**File: `pkg/api/types.go`**

```go
package api

// Platform enum for builds
type Platform string

const (
    PlatformMacOS   Platform = "macos"
    PlatformIOS     Platform = "ios"
    PlatformAndroid Platform = "android"
    PlatformWindows Platform = "windows"
    PlatformLinux   Platform = "linux"
    PlatformWeb     Platform = "web"
)

// BuildInput - request body for build operations
type BuildInput struct {
    Platform Platform `json:"platform" enum:"macos,ios,android,windows,linux,web" doc:"Target platform"`
    AppDir   string   `json:"app_dir" doc:"Path to application directory"`
    Force    bool     `json:"force,omitempty" doc:"Force rebuild even if up-to-date"`
}

// BuildOutput - response body for build operations
type BuildOutput struct {
    Body struct {
        Success    bool   `json:"success" doc:"Whether build succeeded"`
        OutputPath string `json:"output_path" doc:"Path to built artifact"`
        Cached     bool   `json:"cached" doc:"Whether result was from cache"`
        Duration   string `json:"duration" doc:"Build duration"`
    }
}

// UTM types
type VMStatus string

const (
    VMStatusRunning   VMStatus = "running"
    VMStatusStopped   VMStatus = "stopped"
    VMStatusSuspended VMStatus = "suspended"
)

type VMInfo struct {
    Name   string   `json:"name" doc:"VM name"`
    Status VMStatus `json:"status" doc:"Current status"`
    OS     string   `json:"os,omitempty" doc:"Operating system"`
    Memory int      `json:"memory,omitempty" doc:"Memory in MB"`
}

type UTMListOutput struct {
    Body []VMInfo
}

type UTMStartInput struct {
    VMName string `path:"vmName" doc:"Name of VM to start"`
}
```

### Phase 3: Implement Handlers

**File: `pkg/api/build_handlers.go`**

```go
package api

import (
    "context"

    "github.com/danielgtaylor/huma/v2"
    "github.com/joeblew999/utm-dev/pkg/core"
)

func RegisterBuildHandlers(api huma.API) {
    huma.Register(api, huma.Operation{
        OperationID: "build",
        Method:      http.MethodPost,
        Path:        "/build",
        Summary:     "Build application for target platform",
        Description: "Builds a Gio application for the specified platform. Uses build cache for incremental builds.",
        Tags:        []string{"build"},
    }, HandleBuild)

    huma.Register(api, huma.Operation{
        OperationID: "build-check",
        Method:      http.MethodGet,
        Path:        "/build/check",
        Summary:     "Check if rebuild is needed",
        Tags:        []string{"build"},
    }, HandleBuildCheck)
}

func HandleBuild(ctx context.Context, input *BuildInput) (*BuildOutput, error) {
    result, err := core.Build(ctx, core.BuildOptions{
        Platform: string(input.Platform),
        AppDir:   input.AppDir,
        Force:    input.Force,
    })
    if err != nil {
        return nil, huma.Error500InternalServerError("build failed", err)
    }

    return &BuildOutput{
        Body: struct {
            Success    bool   `json:"success"`
            OutputPath string `json:"output_path"`
            Cached     bool   `json:"cached"`
            Duration   string `json:"duration"`
        }{
            Success:    true,
            OutputPath: result.OutputPath,
            Cached:     result.Cached,
            Duration:   result.Duration.String(),
        },
    }, nil
}
```

### Phase 4: SSE Streaming for Long Operations

**File: `pkg/api/build_stream.go`**

```go
package api

import (
    "context"
    "net/http"

    "github.com/danielgtaylor/huma/v2"
    "github.com/danielgtaylor/huma/v2/sse"
)

// SSE event types
type BuildProgress struct {
    Phase   string  `json:"phase" doc:"Current build phase"`
    Percent float64 `json:"percent" doc:"Completion percentage"`
    Message string  `json:"message" doc:"Status message"`
}

type BuildLog struct {
    Level   string `json:"level" doc:"Log level: info, warn, error"`
    Message string `json:"message" doc:"Log message"`
}

type BuildComplete struct {
    Success    bool   `json:"success"`
    OutputPath string `json:"output_path,omitempty"`
    Error      string `json:"error,omitempty"`
}

func RegisterBuildStreamHandlers(api huma.API) {
    sse.Register(api, huma.Operation{
        OperationID: "build-stream",
        Method:      http.MethodGet,
        Path:        "/build/stream",
        Summary:     "Stream build progress",
        Description: "Returns Server-Sent Events with build progress, logs, and completion status.",
        Tags:        []string{"build", "streaming"},
    }, map[string]any{
        "progress": BuildProgress{},
        "log":      BuildLog{},
        "complete": BuildComplete{},
    }, HandleBuildStream)
}

func HandleBuildStream(ctx context.Context, input *BuildInput, send sse.Sender) {
    // Start build with progress callback
    progressCh := make(chan core.Progress)
    logCh := make(chan core.LogEntry)

    go core.BuildWithProgress(ctx, core.BuildOptions{
        Platform: string(input.Platform),
        AppDir:   input.AppDir,
    }, progressCh, logCh)

    for {
        select {
        case p, ok := <-progressCh:
            if !ok {
                return
            }
            send.Data(BuildProgress{
                Phase:   p.Phase,
                Percent: p.Percent,
                Message: p.Message,
            })

        case l, ok := <-logCh:
            if !ok {
                return
            }
            send.Data(BuildLog{
                Level:   l.Level,
                Message: l.Message,
            })

        case <-ctx.Done():
            send.Data(BuildComplete{
                Success: false,
                Error:   "cancelled",
            })
            return
        }
    }
}
```

### Phase 5: Client Generation

**File: `pkg/api/client_gen.go`**

```go
package api

import (
    "github.com/danielgtaylor/humaclient"
)

func init() {
    // Enable client generation with: GENERATE_CLIENT=1 go run .
    humaclient.RegisterWithOptions(api, humaclient.Options{
        PackageName:     "goupclient",
        ClientName:      "Client",
        OutputDirectory: "./pkg/goupclient",
    })
}
```

This generates `pkg/goupclient/client.go`:

```go
// Auto-generated - do not edit

package goupclient

type Client struct { ... }

func New(baseURL string) *Client { ... }

// Build builds an application for the target platform
func (c *Client) Build(ctx context.Context, input BuildInput) (*BuildOutput, error) { ... }

// BuildStream streams build progress via SSE
func (c *Client) BuildStream(ctx context.Context, input BuildInput) (<-chan BuildEvent, error) { ... }

// UTMList lists all virtual machines
func (c *Client) UTMList(ctx context.Context) ([]VMInfo, error) { ... }

// UTMStart starts a virtual machine
func (c *Client) UTMStart(ctx context.Context, vmName string) error { ... }
```

### Phase 6: Restish CLI Integration

**File: `cmd/root.go`** (simplified)

```go
package cmd

import (
    "os"
    "os/exec"

    "github.com/joeblew999/utm-dev/pkg/api"
)

func main() {
    // Check if running as API server
    if os.Getenv("GOUP_SERVER") != "" || containsArg("serve") {
        server := api.NewServer()
        server.ListenAndServe(":8080")
        return
    }

    // Otherwise, proxy to restish with pre-configured API
    runRestish(os.Args[1:])
}

func runRestish(args []string) {
    // Ensure API is configured
    ensureAPIConfigured()

    // Prepend "utm-dev" as the API name
    fullArgs := append([]string{"utm-dev"}, args...)

    cmd := exec.Command("restish", fullArgs...)
    cmd.Stdin = os.Stdin
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    cmd.Run()
}

func ensureAPIConfigured() {
    // Check if ~/.restish/apis.json has utm-dev configured
    // If not, add it pointing to local socket or default URL
}
```

**User Experience**:

```bash
# Start API server (background or separate terminal)
utm-dev serve

# CLI commands auto-discovered from OpenAPI
utm-dev build --platform macos --app-dir examples/webviewer
utm-dev utm list
utm-dev utm start Win11
utm-dev install android-ndk

# Or use restish directly
restish utm-dev build --platform macos --app-dir examples/webviewer
```

### Phase 7: Embedded Mode (No Server)

For local-only usage without running a server:

**File: `cmd/embedded.go`**

```go
package cmd

import (
    "net/http/httptest"

    "github.com/joeblew999/utm-dev/pkg/api"
)

// For local CLI usage, embed the API server in-process
func runEmbedded(args []string) {
    server := api.NewServer()
    ts := httptest.NewServer(server)
    defer ts.Close()

    // Configure restish to use test server
    os.Setenv("GOUP_API_URL", ts.URL)
    runRestish(args)
}
```

This allows the CLI to work without an external server by spinning up an in-process API.

## Migration Path

### Phase 1: Parallel Implementation
1. Create `pkg/api/` with Huma handlers
2. Handlers call existing `pkg/*/` logic
3. Existing Cobra commands continue to work

### Phase 2: Client Generation
1. Add humaclient registration
2. Generate `pkg/goupclient/`
3. Test SDK against running server

### Phase 3: Restish Integration
1. Add `serve` command to start API server
2. Configure Restish API definition
3. Test CLI via Restish

### Phase 4: Deprecate Cobra Commands
1. Mark existing `cmd/*.go` as deprecated
2. Point users to new API-based CLI
3. Eventually remove Cobra commands

### Phase 5: Full Migration
1. Remove old Cobra commands
2. Simplify `cmd/root.go` to server + restish proxy
3. Update documentation

## API Endpoints Summary

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/build` | POST | Build app for platform |
| `/build/check` | GET | Check if rebuild needed |
| `/build/stream` | GET | SSE stream of build progress |
| `/utm` | GET | List all VMs |
| `/utm/{name}` | GET | Get VM details |
| `/utm/{name}/start` | POST | Start VM |
| `/utm/{name}/stop` | POST | Stop VM |
| `/utm/{name}/screenshot` | GET | Capture VM screenshot |
| `/install/{sdk}` | POST | Install SDK |
| `/install` | GET | List installed SDKs |
| `/icons` | POST | Generate icons |
| `/self/version` | GET | Get utm-dev version |
| `/self/doctor` | GET | Run diagnostics |

## Files to Create

| File | Description |
|------|-------------|
| `pkg/api/server.go` | Huma server setup |
| `pkg/api/types.go` | Request/response types |
| `pkg/api/build_handlers.go` | Build operation handlers |
| `pkg/api/build_stream.go` | SSE streaming for builds |
| `pkg/api/utm_handlers.go` | UTM/VM handlers |
| `pkg/api/install_handlers.go` | SDK installation handlers |
| `pkg/api/icons_handlers.go` | Icon generation handlers |
| `pkg/api/self_handlers.go` | Self-management handlers |
| `pkg/api/client_gen.go` | humaclient registration |
| `pkg/core/build.go` | Refactored build logic |
| `pkg/core/utm.go` | Refactored UTM logic |
| `cmd/serve.go` | Server command |

## Consequences

### Benefits

1. **Single source of truth** - API defines all operations
2. **Auto-generated CLI** - No Cobra command maintenance
3. **Auto-generated SDK** - Type-safe Go client for free
4. **Built-in streaming** - SSE for progress without extra work
5. **API documentation** - Swagger UI from OpenAPI
6. **Remote-ready** - Same tool works locally and over network
7. **MCP-ready** - OpenAPI schemas inform MCP tool definitions
8. **Testability** - HTTP handlers are easy to test

### Trade-offs

1. **Dependency on Huma ecosystem** - Tied to Daniel G. Taylor's tools
2. **Server requirement** - Need running server (or embedded mode)
3. **Migration effort** - Existing code needs refactoring
4. **Learning curve** - Team needs to learn Huma patterns

### Risks

1. **Huma maintenance** - Project could become unmaintained
   - Mitigation: Huma is well-maintained, has corporate users
2. **Restish adoption** - Users may not want another tool
   - Mitigation: Embed restish or provide thin wrapper
3. **Performance** - HTTP overhead for local operations
   - Mitigation: Embedded mode with in-process server

## Related ADRs

- [ADR-003](003-mcp-server-integration.md) - MCP can use OpenAPI schemas
- [ADR-004](004-bubbletea-tui-integration.md) - TUI/Web can use same API
- [ADR-005](005-openapi-spec-export.md) - Superseded by this ADR (OpenAPI is automatic)

## References

- [Huma Framework](https://huma.rocks/)
- [Huma GitHub](https://github.com/danielgtaylor/huma)
- [Huma SSE Package](https://pkg.go.dev/github.com/danielgtaylor/huma/v2/sse)
- [humaclient](https://github.com/danielgtaylor/humaclient)
- [Restish](https://rest.sh/)
- [Restish GitHub](https://github.com/rest-sh/restish)
- [gocrud](https://github.com/ckoliber/gocrud)

## Source Code References

Local copies for development reference:

- `.src/huma/` - Huma framework source
- `.src/huma/sse/` - SSE implementation
- `.src/humaclient/` - Client generator
- `.src/restish/` - CLI generator
- `.src/gocrud/` - CRUD generator
