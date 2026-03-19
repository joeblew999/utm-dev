# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for utm-dev.

## Core Principle: Single Source of Truth

All ADRs share a common approach: **single source generates all interfaces**.

### Current Direction: API-First (ADR-006)

```
pkg/core/ (Business Logic)
    ↑
pkg/api/ (Huma handlers)
    │
    └─→ OpenAPI spec (auto-generated)
            │
            ├─→ Restish CLI (auto-discovered)
            ├─→ humaclient SDK (auto-generated)
            ├─→ MCP tools (schema from OpenAPI)
            └─→ Swagger UI (documentation)
```

### Previous Direction: Cobra-First (ADR-002/003/005)

```
cmd/*.go (Cobra commands)
    │
    ├─→ pkg/generate/docs.go    → Markdown documentation
    ├─→ pkg/mcp/generator.go    → MCP server tools
    └─→ pkg/generate/openapi.go → OpenAPI / LLM functions
```

**Note**: ADR-006 supersedes ADR-005 by making OpenAPI automatic via Huma.

### What Cobra Gives Us (90%)

| Cobra Field | Extractable |
|-------------|-------------|
| `Use` | Command name, usage pattern |
| `Short` | Brief description |
| `Long` | Detailed description |
| `Example` | Usage examples |
| `Flags()` | Parameters with types, defaults, descriptions |
| `GroupID` | Categories |
| `Aliases` | Alternative names |

### What Cobra Doesn't Give Us (10%)

| Missing | Solution |
|---------|----------|
| Enum values for flags | Shared `pkg/schema/platforms.go` |
| Positional arg descriptions | Shared `pkg/schema/args.go` |
| Complex validation rules | Document in `Long` field |

**Supplemental schema**: One file (`pkg/schema/schema.go`) with ~20 lines covering the edge cases.

## ADR Index

| ADR | Title | Status | Description |
|-----|-------|--------|-------------|
| [002](002-llm-ready-documentation.md) | LLM-Ready Documentation | Proposed | Generate docs from Cobra |
| [003](003-mcp-server-integration.md) | MCP Server Integration | Proposed | Generate MCP tools from Cobra |
| [004](004-bubbletea-tui-integration.md) | Bubbletea TUI Integration | Proposed | Interactive TUI (or Web GUI) |
| [005](005-openapi-spec-export.md) | OpenAPI Spec Export | Superseded | Generate OpenAPI/LLM functions from Cobra |
| [006](006-api-first-cli-architecture.md) | API-First CLI Architecture | **Proposed** | Huma + Restish unified API/CLI |

## Honest Assessment

**Will it really work?**

- **90% yes** - Flags, descriptions, types, defaults all extract cleanly
- **10% manual** - Positional args and enums need a small supplemental schema

**The alternative** (full manual schemas) would mean:
- Duplicating every command definition
- Keeping two sources in sync
- More maintenance, more bugs

**This approach**:
- Cobra IS the schema (mostly)
- Small supplement for edge cases
- One place to update = everywhere updates

## Implementation Priority

1. **ADR-006** (API-First) - Foundation for everything else
2. **ADR-003** (MCP Server) - Uses OpenAPI schemas from ADR-006
3. **ADR-004** (TUI/Web GUI) - Consumes API from ADR-006
4. **ADR-002** (LLM Docs) - Auto-generated from OpenAPI

**Note**: ADR-006 provides the foundation that makes other ADRs simpler to implement.

## Status Definitions

- **Proposed** - Under consideration
- **Accepted** - Decision made, implementation planned
- **Implemented** - Done
- **Deprecated** - Superseded

## Related Files

When implementing:
- `pkg/api/` - Huma API handlers (ADR-006)
- `pkg/core/` - Business logic (ADR-006)
- `pkg/goupclient/` - Auto-generated Go SDK (ADR-006)
- `pkg/mcp/` - MCP server (ADR-003)
- `pkg/schema/schema.go` - Shared supplemental definitions (legacy)

## Source Code References

Local copies of dependencies in `.src/`:
- `.src/huma/` - Huma REST framework
- `.src/humaclient/` - Go client generator
- `.src/restish/` - CLI from OpenAPI
- `.src/gocrud/` - CRUD endpoint generator

## References

- [Cobra documentation](https://cobra.dev/)
- [MCP Protocol](https://spec.modelcontextprotocol.io/)
- [OpenAPI Spec](https://spec.openapis.org/)
