# ADR-002: LLM-Ready CLI Documentation

## Status

**Proposed** - Enhance Cobra documentation for AI/LLM consumption.

## Context

LLMs (Claude, ChatGPT, etc.) are increasingly used to help developers understand and use CLI tools. Cobra already generates documentation, but it can be optimized for AI consumption by:
- Providing strong examples with concrete I/O
- Using consistent, structured output
- Disabling auto-generated timestamps (reproducible output)
- Organizing commands for easy chunking in vector databases

Reference: [Cobra LLM Documentation Guide](https://cobra.dev/docs/how-to-guides/clis-for-llms/)

## Single Source of Truth Principle

**CRITICAL**: All documentation is extracted from existing Cobra command definitions. No new metadata or annotations are added.

### Cobra Fields → Documentation

| Cobra Field | Documentation Output |
|-------------|---------------------|
| `Use` | Command signature, usage pattern |
| `Short` | One-line description |
| `Long` | Detailed description with context |
| `Example` | Concrete usage examples with expected output |
| `Flags()` | Parameter documentation with types and defaults |
| `GroupID` | Category/section organization |
| `Aliases` | Alternative command names |
| `Args` | Argument validation rules |

**No duplication** - enhance the Cobra fields themselves, then generate docs from them.

## Decision

Enhance all utm-dev commands with LLM-friendly documentation patterns:

1. **Every command gets strong `Example` fields** with concrete input/output
2. **Every command gets detailed `Long` descriptions** explaining what/why
3. **Disable auto-generated tags** for reproducible output
4. **Organize by command groups** for logical chunking

## Implementation Plan

### Phase 1: Documentation Generator

**File: `pkg/generate/docs.go`**

Extract documentation directly from Cobra tree:

```go
package generate

import (
    "github.com/spf13/cobra"
    "github.com/spf13/cobra/doc"
)

// GenerateDocs walks the Cobra command tree and generates documentation
func GenerateDocs(root *cobra.Command, outputDir string, opts Options) error {
    if opts.LLMMode {
        // Disable timestamps for reproducible output
        root.DisableAutoGenTag = true
    }

    if opts.SingleFile {
        return generateConsolidated(root, outputDir)
    }

    return doc.GenMarkdownTree(root, outputDir)
}

// generateConsolidated creates a single LLM-optimized file
func generateConsolidated(root *cobra.Command, outputDir string) error {
    var buf bytes.Buffer

    buf.WriteString("# utm-dev CLI Reference\n\n")

    // Walk command tree - everything comes from existing Cobra fields
    walkForDocs(root, &buf, 0)

    return os.WriteFile(filepath.Join(outputDir, "cli-reference.md"), buf.Bytes(), 0644)
}

func walkForDocs(cmd *cobra.Command, buf *bytes.Buffer, depth int) {
    if cmd.Hidden {
        return
    }

    // All data extracted from existing Cobra fields
    fmt.Fprintf(buf, "## %s\n\n", cmd.CommandPath())
    fmt.Fprintf(buf, "%s\n\n", cmd.Short)  // From cmd.Short

    if cmd.Long != "" {
        fmt.Fprintf(buf, "%s\n\n", cmd.Long)  // From cmd.Long
    }

    fmt.Fprintf(buf, "**Usage:** `%s`\n\n", cmd.UseLine())  // From cmd.Use

    // Flags from cmd.Flags()
    if cmd.HasAvailableLocalFlags() {
        fmt.Fprintf(buf, "**Flags:**\n")
        cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
            fmt.Fprintf(buf, "- `--%s`: %s", f.Name, f.Usage)
            if f.DefValue != "" {
                fmt.Fprintf(buf, " (default: %s)", f.DefValue)
            }
            buf.WriteString("\n")
        })
        buf.WriteString("\n")
    }

    // Examples from cmd.Example
    if cmd.Example != "" {
        fmt.Fprintf(buf, "**Examples:**\n```bash\n%s\n```\n\n", cmd.Example)
    }

    // Recurse into subcommands
    for _, sub := range cmd.Commands() {
        walkForDocs(sub, buf, depth+1)
    }
}
```

### Phase 2: Enhance Existing Command Fields

Improve the source Cobra definitions (not add new ones):

**Pattern for all commands:**

```go
var buildCmd = &cobra.Command{
    Use:   "build [platform] [app-directory]",
    Short: "Build Gio applications for different platforms",
    Long: `Build Gio applications for various platforms with deep linking and native features.

This command compiles your Go/Gio application into platform-specific binaries:
- macOS: .app bundle with WKWebView support
- iOS: .ipa for App Store or TestFlight
- Android: .apk with Chromium WebView
- Windows: .exe with WebView2

The build system is idempotent - it only rebuilds when source files change.`,
    Example: `  # Build for macOS
  utm-dev build macos examples/hybrid-dashboard
  # Output: examples/hybrid-dashboard/.bin/macos/hybrid-dashboard.app

  # Build for Android with deep linking
  utm-dev build android examples/hybrid-dashboard --schemes "myapp://,https://example.com"
  # Output: examples/hybrid-dashboard/.bin/android/hybrid-dashboard.apk

  # Check if rebuild is needed (for CI)
  utm-dev build --check macos examples/hybrid-dashboard
  # Exit code: 0 = up-to-date, 1 = needs rebuild`,
}
```

### Phase 3: Command Audit Checklist

Audit all commands for completeness of existing fields:

| Command | Has Example | Has Long | GroupID | Status |
|---------|-------------|----------|---------|--------|
| build | ✅ | ✅ | build | Done |
| run | ⚠️ | ⚠️ | build | Needs enhancement |
| bundle | ⚠️ | ✅ | build | Needs examples |
| package | ⚠️ | ⚠️ | build | Needs enhancement |
| install | ⚠️ | ⚠️ | sdk | Needs enhancement |
| list | ⚠️ | ✅ | sdk | Needs examples |
| utm * | ✅ | ✅ | vm | Good |
| self * | ⚠️ | ✅ | self | Needs examples |
| icons | ⚠️ | ⚠️ | tools | Needs enhancement |
| generate | ⚠️ | ⚠️ | tools | Needs enhancement |

### Phase 4: Generate Command

**File: `cmd/generate_docs.go`**

```go
var generateDocsCmd = &cobra.Command{
    Use:   "docs [output-dir]",
    Short: "Generate CLI documentation from Cobra definitions",
    Long: `Generate documentation extracted from Cobra command definitions.

All documentation comes from existing command fields:
- Use → command signature
- Short → one-line description
- Long → detailed description
- Example → usage examples
- Flags() → parameter documentation

No additional metadata required - Cobra IS the source of truth.`,
    Example: `  # Generate markdown docs
  utm-dev generate docs

  # Generate single LLM-optimized file
  utm-dev generate docs --single-file

  # Generate without timestamps (for git)
  utm-dev generate docs --no-timestamp`,
    RunE: func(cmd *cobra.Command, args []string) error {
        outputDir := "docs/cli"
        if len(args) > 0 {
            outputDir = args[0]
        }

        singleFile, _ := cmd.Flags().GetBool("single-file")
        noTimestamp, _ := cmd.Flags().GetBool("no-timestamp")

        return generate.GenerateDocs(rootCmd, outputDir, generate.Options{
            LLMMode:    noTimestamp,
            SingleFile: singleFile,
        })
    },
}
```

## Files to Modify

| File | Action | Description |
|------|--------|-------------|
| `pkg/generate/docs.go` | Create | Doc extraction from Cobra tree |
| `cmd/generate.go` | Modify | Add docs subcommand |
| `cmd/build.go` | Modify | Enhance Example field |
| `cmd/run.go` | Modify | Enhance Example and Long fields |
| `cmd/bundle.go` | Modify | Enhance Example field |
| `cmd/package.go` | Modify | Enhance Example and Long fields |
| `cmd/install.go` | Modify | Enhance Example and Long fields |
| `cmd/icons.go` | Modify | Enhance Example and Long fields |

## Verification

1. **Generate docs:**
   ```bash
   utm-dev generate docs docs/cli
   ```

2. **Check no timestamps:**
   ```bash
   grep "Auto generated" docs/cli/*.md
   # Should return nothing
   ```

3. **Verify examples exist:**
   ```bash
   for f in docs/cli/utm-dev_*.md; do
     grep -q "## Examples" "$f" || echo "Missing examples: $f"
   done
   ```

4. **Test with LLM:**
   - Feed generated docs to Claude/ChatGPT
   - Ask: "How do I build an Android app with deep linking?"
   - Verify it can answer from the docs

## Consequences

### Benefits
- **Single source of truth** - Cobra definitions are the only place to update
- LLMs can accurately answer questions about utm-dev
- Better discoverability through search engines
- Consistent documentation across all commands
- Easy to integrate with documentation sites

### Trade-offs
- Need to enhance existing Cobra fields (one-time effort)
- Examples in code need to stay accurate
- Larger Long descriptions in cmd/*.go files

## Related ADRs

- [ADR-003](003-mcp-server-integration.md) - MCP tools extracted from same Cobra definitions
- [ADR-005](005-openapi-spec-export.md) - OpenAPI specs extracted from same Cobra definitions

## References

- [Cobra LLM Documentation Guide](https://cobra.dev/docs/how-to-guides/clis-for-llms/)
- [Cobra doc package](https://pkg.go.dev/github.com/spf13/cobra/doc)
