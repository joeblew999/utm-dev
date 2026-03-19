# ADR-005: OpenAPI Spec Export

## Status

**Proposed** - Generate OpenAPI specifications from CLI commands for API and tool integration.

## Context

Many modern tools expose their functionality through APIs, enabling:
- Integration with other tools and platforms
- Auto-generation of client libraries
- Interactive API documentation (Swagger UI)
- LLM function calling with structured schemas
- Webhook and automation integrations

## Single Source of Truth Principle

**CRITICAL**: OpenAPI specs and LLM function schemas are generated from existing Cobra command definitions.

### Cobra Fields → OpenAPI/Function Schemas

| Cobra Field | OpenAPI | LLM Function |
|-------------|---------|--------------|
| `Use` | Path, operationId | Function name |
| `Short` | Summary | Description |
| `Long` | Description | Extended description |
| `Flags()` | Parameters | Input schema properties |
| `GroupID` | Tags | Category |

### What This Approach CAN Extract

```go
// From cmd/build.go
buildCmd.Flags().Bool("force", false, "Force rebuild even if up-to-date")
```

**Extracts:**
- Name: `force`
- Type: `boolean`
- Default: `false`
- Description: `Force rebuild even if up-to-date`

### What This Approach CANNOT Extract

**Enum values** - Cobra doesn't store these:
```go
// This works for completion but enum values aren't stored
buildCmd.ValidArgsFunction = func(...) { return []string{"macos", "ios"} }
```

**Positional arg descriptions** - Cobra only has validators:
```go
Args: cobra.ExactArgs(2)  // We know there are 2 args, but not what they are
```

### Hybrid Solution

For the ~10% that Cobra doesn't capture, use a small shared schema:

```go
// pkg/schema/definitions.go - ONE place for supplemental metadata
var CommandSchemas = map[string]Schema{
    "build": {
        Args: []Arg{
            {Name: "platform", Enum: Platforms, Desc: "Target platform"},
            {Name: "app_dir", Desc: "Path to application directory"},
        },
    },
}
```

## Decision

Add OpenAPI specification generation to utm-dev:

```bash
# Generate OpenAPI spec (extracts from Cobra)
utm-dev generate openapi

# Generate LLM function schemas
utm-dev generate functions --format anthropic
```

## Implementation Plan

### Phase 1: Shared Schema Definitions

**File: `pkg/schema/schema.go`**

```go
package schema

// Platforms - used by Cobra completion AND schema generation
var Platforms = []string{"macos", "ios", "android", "windows", "linux", "web"}

// Arg describes a positional argument (Cobra doesn't capture this)
type Arg struct {
    Name string
    Type string   // "string", "integer", etc.
    Desc string
    Enum []string // optional
}

// CommandSchema supplements Cobra with what it can't express
type CommandSchema struct {
    Args []Arg // Positional arguments
}

// Schemas - supplemental metadata for commands with positional args
var Schemas = map[string]CommandSchema{
    "build": {
        Args: []Arg{
            {Name: "platform", Type: "string", Desc: "Target platform", Enum: Platforms},
            {Name: "app_dir", Type: "string", Desc: "Path to application directory"},
        },
    },
    "utm_start": {
        Args: []Arg{
            {Name: "vm_name", Type: "string", Desc: "Name of the VM to start"},
        },
    },
    // Only ~5-10 commands need this - most have no positional args
}
```

### Phase 2: OpenAPI Generator

**File: `pkg/generate/openapi.go`**

```go
package generate

import (
    "github.com/joeblew999/utm-dev/pkg/schema"
    "github.com/spf13/cobra"
    "github.com/spf13/pflag"
)

func GenerateOpenAPI(root *cobra.Command) map[string]interface{} {
    spec := map[string]interface{}{
        "openapi": "3.0.3",
        "info": map[string]interface{}{
            "title":   "utm-dev API",
            "version": "1.0.0",
        },
        "paths": make(map[string]interface{}),
    }

    paths := spec["paths"].(map[string]interface{})
    walkForOpenAPI(root, "", paths)

    return spec
}

func walkForOpenAPI(cmd *cobra.Command, prefix string, paths map[string]interface{}) {
    if cmd.Hidden || cmd.RunE == nil {
        for _, sub := range cmd.Commands() {
            walkForOpenAPI(sub, prefix+"/"+cmd.Name(), paths)
        }
        return
    }

    path := prefix + "/" + cmd.Name()
    paths[path] = buildOperation(cmd)

    for _, sub := range cmd.Commands() {
        walkForOpenAPI(sub, path, paths)
    }
}

func buildOperation(cmd *cobra.Command) map[string]interface{} {
    op := map[string]interface{}{
        "summary":     cmd.Short,                      // From Cobra
        "description": cmd.Long,                       // From Cobra
        "operationId": strings.ReplaceAll(cmd.CommandPath(), " ", "_"),
        "tags":        []string{cmd.GroupID},          // From Cobra
        "parameters":  []interface{}{},
    }

    params := []interface{}{}

    // 1. Extract flags from Cobra (90% of params)
    cmd.Flags().VisitAll(func(f *pflag.Flag) {
        param := map[string]interface{}{
            "name":        f.Name,
            "in":          "query",
            "description": f.Usage,
            "required":    false,
            "schema":      flagToSchema(f),
        }
        params = append(params, param)
    })

    // 2. Add positional args from supplemental schema (10%)
    if cmdSchema, ok := schema.Schemas[cmd.Name()]; ok {
        for _, arg := range cmdSchema.Args {
            param := map[string]interface{}{
                "name":        arg.Name,
                "in":          "query",
                "description": arg.Desc,
                "required":    true,
                "schema":      argToSchema(arg),
            }
            params = append(params, param)
        }
    }

    op["parameters"] = params
    return op
}

func flagToSchema(f *pflag.Flag) map[string]interface{} {
    schema := map[string]interface{}{}

    switch f.Value.Type() {
    case "bool":
        schema["type"] = "boolean"
    case "int", "int64":
        schema["type"] = "integer"
    case "stringSlice":
        schema["type"] = "array"
        schema["items"] = map[string]interface{}{"type": "string"}
    default:
        schema["type"] = "string"
    }

    if f.DefValue != "" && f.DefValue != "false" && f.DefValue != "0" {
        schema["default"] = f.DefValue
    }

    return schema
}

func argToSchema(arg schema.Arg) map[string]interface{} {
    s := map[string]interface{}{
        "type": arg.Type,
    }
    if len(arg.Enum) > 0 {
        s["enum"] = arg.Enum
    }
    return s
}
```

### Phase 3: LLM Function Generator

**File: `pkg/generate/functions.go`**

```go
package generate

// GenerateFunctions creates LLM tool/function schemas
func GenerateFunctions(root *cobra.Command, format string) []map[string]interface{} {
    var functions []map[string]interface{}

    walkForFunctions(root, &functions)

    // Format for specific LLM provider
    switch format {
    case "anthropic":
        return formatAnthropic(functions)
    case "openai":
        return formatOpenAI(functions)
    default:
        return functions
    }
}

func formatAnthropic(functions []map[string]interface{}) []map[string]interface{} {
    var tools []map[string]interface{}
    for _, f := range functions {
        tools = append(tools, map[string]interface{}{
            "name":         f["name"],
            "description":  f["description"],
            "input_schema": f["parameters"],
        })
    }
    return tools
}
```

### Phase 4: Generate Commands

**File: `cmd/generate_openapi.go`**

```go
var generateOpenAPICmd = &cobra.Command{
    Use:   "openapi [output-file]",
    Short: "Generate OpenAPI spec from Cobra definitions",
    Example: `  # Generate OpenAPI YAML
  utm-dev generate openapi api/openapi.yaml

  # Generate as JSON
  utm-dev generate openapi --format json`,
    RunE: func(cmd *cobra.Command, args []string) error {
        spec := generate.GenerateOpenAPI(rootCmd)
        // ... output logic
    },
}

var generateFunctionsCmd = &cobra.Command{
    Use:   "functions",
    Short: "Generate LLM function schemas from Cobra definitions",
    Example: `  # Generate Anthropic tool schemas
  utm-dev generate functions --format anthropic

  # Generate OpenAI function schemas
  utm-dev generate functions --format openai`,
    RunE: func(cmd *cobra.Command, args []string) error {
        format, _ := cmd.Flags().GetString("format")
        functions := generate.GenerateFunctions(rootCmd, format)
        // ... output logic
    },
}
```

## What Gets Extracted vs Defined

| Source | Coverage | Examples |
|--------|----------|----------|
| Cobra flags | ~90% | --force, --format, --output |
| Cobra metadata | 100% | name, description, groups |
| Supplemental schema | ~10% | positional args, enums |

**Commands needing supplemental schema:**
- `build` (platform, app_dir args)
- `run` (platform, app_dir args)
- `utm start/stop` (vm_name arg)
- `install` (sdk_name arg)

**Commands fully extracted from Cobra:**
- `list` (flags only)
- `icons` (flags only)
- `self *` (flags only)
- `generate *` (flags only)

## Files to Create/Modify

| File | Action | Description |
|------|--------|-------------|
| `pkg/schema/schema.go` | Create | Shared definitions + supplemental args |
| `pkg/generate/openapi.go` | Create | Cobra → OpenAPI generator |
| `pkg/generate/functions.go` | Create | Cobra → LLM function generator |
| `cmd/generate_openapi.go` | Create | Generate openapi command |
| `cmd/generate_functions.go` | Create | Generate functions command |

## Verification

1. **Generate OpenAPI spec:**
   ```bash
   utm-dev generate openapi api/openapi.yaml
   ```

2. **Generate LLM functions:**
   ```bash
   utm-dev generate functions --format anthropic
   ```

3. **Validate coverage:**
   ```bash
   # All flags should appear in generated schema
   utm-dev build --help | grep -E "^\s+--"
   # Compare with generated openapi.yaml
   ```

## Consequences

### Benefits
- **Mostly single source of truth** - 90% from Cobra
- Small supplemental schema for edge cases
- Schemas always match CLI behavior
- Easy to keep in sync

### Trade-offs
- Need supplemental schema for positional args (~10 commands)
- Enum values need to be defined in shared location
- Not 100% auto-generated (but close)

### Honest Assessment

Pure extraction from Cobra gets you ~90%. The remaining 10% (positional args, enums) needs a small supplemental schema. This is a reasonable trade-off:
- One `pkg/schema/schema.go` file
- ~20 lines of arg definitions
- Used by all generators (docs, MCP, OpenAPI)

## Related ADRs

- [ADR-002](002-llm-ready-documentation.md) - Docs from same sources
- [ADR-003](003-mcp-server-integration.md) - MCP tools from same sources

## References

- [OpenAPI Specification 3.0](https://spec.openapis.org/oas/v3.0.3)
- [OpenAI Function Calling](https://platform.openai.com/docs/guides/function-calling)
- [Anthropic Tool Use](https://docs.anthropic.com/claude/docs/tool-use)
