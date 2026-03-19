# ADR-004: Bubbletea TUI Integration

## Status

I THINK A WEB GUI IS BETTER !! 

**Proposed** - Add interactive TUI mode using Charmbracelet's Bubbletea framework.

## Context

Many CLI tools benefit from interactive terminal user interfaces (TUIs) that provide:
- Real-time feedback and progress visualization
- Interactive menus for complex operations
- Better user experience for exploration and discovery
- Dashboard-style views for monitoring

The Charmbracelet ecosystem provides excellent Go libraries for building TUIs:
- **Bubbletea** - The Elm-inspired TUI framework
- **Bubbles** - Pre-built components (spinners, progress bars, tables, etc.)
- **Lipgloss** - Style definitions for terminal rendering
- **Huh** - Form/survey library built on Bubbletea

Reference:
- [Bubbletea](https://github.com/charmbracelet/bubbletea)
- [Bubbles](https://github.com/charmbracelet/bubbles)
- [Lipgloss](https://github.com/charmbracelet/lipgloss)
- [Huh](https://github.com/charmbracelet/huh)

## Decision

Add an optional TUI mode to utm-dev for interactive operations:

```bash
# Start interactive TUI
utm-dev tui

# Or use --interactive flag on specific commands
utm-dev build --interactive
utm-dev utm --interactive
```

This provides a richer experience while maintaining the standard CLI for scripting and automation.

## Implementation Plan

### Phase 1: Add Charmbracelet Dependencies

**File: `go.mod`**

```go
require (
    github.com/charmbracelet/bubbletea v1.x.x
    github.com/charmbracelet/bubbles v0.x.x
    github.com/charmbracelet/lipgloss v1.x.x
    github.com/charmbracelet/huh v0.x.x
)
```

### Phase 2: Create TUI Package Structure

**Directory: `pkg/tui/`**

```
pkg/tui/
├── tui.go           # Main TUI entry point
├── styles.go        # Lipgloss style definitions
├── components/
│   ├── menu.go      # Main menu component
│   ├── build.go     # Build workflow TUI
│   ├── utm.go       # VM management TUI
│   ├── sdk.go       # SDK installation TUI
│   └── progress.go  # Progress display component
└── models/
    ├── state.go     # Application state
    └── messages.go  # Custom message types
```

### Phase 3: Main TUI Command

**File: `cmd/tui.go`**

```go
package cmd

import (
    "github.com/joeblew999/utm-dev/pkg/tui"
    "github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
    Use:   "tui",
    Short: "Start interactive terminal UI",
    Long: `Launch utm-dev in interactive TUI mode.

This provides a visual interface for:
- Building applications for multiple platforms
- Managing UTM virtual machines
- Installing and updating SDKs
- Monitoring build progress

Navigation:
  ↑/↓       Navigate menus
  Enter     Select item
  Esc/q     Go back / Quit
  ?         Show help

Examples:
  # Start main TUI
  utm-dev tui

  # Start directly in VM management
  utm-dev tui --section vm`,
    RunE: func(cmd *cobra.Command, args []string) error {
        section, _ := cmd.Flags().GetString("section")
        return tui.Run(section)
    },
}

func init() {
    tuiCmd.Flags().String("section", "", "Start in specific section (build, vm, sdk)")
    tuiCmd.GroupID = "tools"
    rootCmd.AddCommand(tuiCmd)
}
```

### Phase 4: TUI Components

**File: `pkg/tui/tui.go`**

```go
package tui

import (
    "fmt"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type Model struct {
    currentView string
    menuIndex   int
    width       int
    height      int
    // Sub-models for different views
    buildModel  BuildModel
    vmModel     VMModel
    sdkModel    SDKModel
}

type menuItem struct {
    title       string
    description string
    view        string
}

var mainMenu = []menuItem{
    {"Build", "Build applications for different platforms", "build"},
    {"Virtual Machines", "Manage UTM virtual machines", "vm"},
    {"SDKs", "Install and manage platform SDKs", "sdk"},
    {"Settings", "Configure utm-dev", "settings"},
}

func Run(section string) error {
    m := NewModel()
    if section != "" {
        m.currentView = section
    }

    p := tea.NewProgram(m, tea.WithAltScreen())
    _, err := p.Run()
    return err
}

func NewModel() Model {
    return Model{
        currentView: "menu",
        menuIndex:   0,
    }
}

func (m Model) Init() tea.Cmd {
    return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "q", "ctrl+c":
            return m, tea.Quit
        case "esc":
            if m.currentView != "menu" {
                m.currentView = "menu"
            } else {
                return m, tea.Quit
            }
        case "up", "k":
            if m.menuIndex > 0 {
                m.menuIndex--
            }
        case "down", "j":
            if m.menuIndex < len(mainMenu)-1 {
                m.menuIndex++
            }
        case "enter":
            m.currentView = mainMenu[m.menuIndex].view
        }
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
    }
    return m, nil
}

func (m Model) View() string {
    switch m.currentView {
    case "build":
        return m.buildModel.View()
    case "vm":
        return m.vmModel.View()
    case "sdk":
        return m.sdkModel.View()
    default:
        return m.renderMainMenu()
    }
}

func (m Model) renderMainMenu() string {
    s := titleStyle.Render("utm-dev") + "\n\n"

    for i, item := range mainMenu {
        cursor := "  "
        style := itemStyle
        if i == m.menuIndex {
            cursor = "▸ "
            style = selectedItemStyle
        }
        s += cursor + style.Render(item.title) + "\n"
        s += "    " + descStyle.Render(item.description) + "\n\n"
    }

    s += "\n" + helpStyle.Render("↑/↓: navigate • enter: select • q: quit")
    return s
}
```

**File: `pkg/tui/styles.go`**

```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
    // Colors
    primaryColor   = lipgloss.Color("#7C3AED")  // Purple
    secondaryColor = lipgloss.Color("#10B981")  // Green
    mutedColor     = lipgloss.Color("#6B7280")  // Gray

    // Styles
    titleStyle = lipgloss.NewStyle().
        Bold(true).
        Foreground(primaryColor).
        MarginBottom(1)

    itemStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#FFFFFF"))

    selectedItemStyle = lipgloss.NewStyle().
        Foreground(primaryColor).
        Bold(true)

    descStyle = lipgloss.NewStyle().
        Foreground(mutedColor).
        Italic(true)

    helpStyle = lipgloss.NewStyle().
        Foreground(mutedColor)

    successStyle = lipgloss.NewStyle().
        Foreground(secondaryColor)

    errorStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#EF4444"))
)
```

### Phase 5: Build TUI Component

**File: `pkg/tui/components/build.go`**

```go
package components

import (
    "fmt"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/bubbles/list"
    "github.com/charmbracelet/bubbles/progress"
    "github.com/charmbracelet/bubbles/spinner"
)

type BuildModel struct {
    platforms    []string
    examples     []string
    selected     map[string]bool
    building     bool
    progress     progress.Model
    spinner      spinner.Model
    currentBuild string
    logs         []string
}

func NewBuildModel() BuildModel {
    return BuildModel{
        platforms: []string{"macos", "ios", "android", "windows", "linux"},
        examples:  detectExamples(), // Scan examples/ directory
        selected:  make(map[string]bool),
        progress:  progress.New(progress.WithDefaultGradient()),
        spinner:   spinner.New(spinner.WithSpinner(spinner.Dot)),
    }
}

func (m BuildModel) View() string {
    if m.building {
        return m.renderBuildProgress()
    }
    return m.renderBuildMenu()
}

func (m BuildModel) renderBuildMenu() string {
    s := "Select platforms to build:\n\n"

    for _, p := range m.platforms {
        checkbox := "[ ]"
        if m.selected[p] {
            checkbox = "[✓]"
        }
        s += fmt.Sprintf("  %s %s\n", checkbox, p)
    }

    s += "\nSelect example to build:\n\n"
    for _, e := range m.examples {
        checkbox := "[ ]"
        if m.selected[e] {
            checkbox = "[✓]"
        }
        s += fmt.Sprintf("  %s %s\n", checkbox, e)
    }

    s += "\n[space] toggle • [enter] build • [esc] back"
    return s
}

func (m BuildModel) renderBuildProgress() string {
    s := fmt.Sprintf("%s Building %s...\n\n", m.spinner.View(), m.currentBuild)
    s += m.progress.View() + "\n\n"

    // Show recent logs
    for _, log := range m.logs {
        s += "  " + log + "\n"
    }

    return s
}
```

### Phase 6: VM Management TUI

**File: `pkg/tui/components/utm.go`**

```go
package components

import (
    "github.com/charmbracelet/bubbles/table"
    "github.com/joeblew999/utm-dev/pkg/utm"
)

type VMModel struct {
    table    table.Model
    vms      []utm.VMInfo
    selected int
}

func NewVMModel() VMModel {
    columns := []table.Column{
        {Title: "Name", Width: 20},
        {Title: "Status", Width: 10},
        {Title: "OS", Width: 15},
        {Title: "Memory", Width: 10},
    }

    t := table.New(
        table.WithColumns(columns),
        table.WithFocused(true),
        table.WithHeight(10),
    )

    return VMModel{
        table: t,
        vms:   utm.ListVMs(),
    }
}

func (m VMModel) View() string {
    s := "Virtual Machines\n\n"
    s += m.table.View() + "\n\n"
    s += "[enter] start/stop • [d] delete • [esc] back"
    return s
}
```

### Phase 7: Interactive Forms with Huh

**File: `pkg/tui/forms/setup.go`**

```go
package forms

import (
    "github.com/charmbracelet/huh"
)

func RunSetupWizard() error {
    var (
        platforms []string
        sdks      []string
        confirm   bool
    )

    form := huh.NewForm(
        huh.NewGroup(
            huh.NewMultiSelect[string]().
                Title("Which platforms do you want to target?").
                Options(
                    huh.NewOption("macOS", "macos"),
                    huh.NewOption("iOS", "ios"),
                    huh.NewOption("Android", "android"),
                    huh.NewOption("Windows", "windows"),
                    huh.NewOption("Linux", "linux"),
                ).
                Value(&platforms),
        ),
        huh.NewGroup(
            huh.NewMultiSelect[string]().
                Title("Which SDKs should be installed?").
                Options(
                    huh.NewOption("Android SDK", "android-sdk"),
                    huh.NewOption("Android NDK", "android-ndk"),
                    huh.NewOption("Garble (obfuscation)", "garble"),
                ).
                Value(&sdks),
        ),
        huh.NewGroup(
            huh.NewConfirm().
                Title("Install selected SDKs now?").
                Value(&confirm),
        ),
    )

    if err := form.Run(); err != nil {
        return err
    }

    if confirm {
        // Install selected SDKs
        for _, sdk := range sdks {
            installSDK(sdk)
        }
    }

    return nil
}
```

## TUI Feature Matrix

| Feature | Component | Description |
|---------|-----------|-------------|
| Main Menu | `tui.go` | Navigation hub for all features |
| Build Wizard | `build.go` | Multi-platform build with progress |
| VM Dashboard | `utm.go` | Table view of VMs with controls |
| SDK Manager | `sdk.go` | Install/update SDKs interactively |
| Setup Wizard | `forms/setup.go` | First-run configuration |
| Progress Display | `progress.go` | Build and download progress |
| Log Viewer | `logs.go` | Scrollable build output |

## Files to Create/Modify

| File | Action | Description |
|------|--------|-------------|
| `go.mod` | Modify | Add Charmbracelet dependencies |
| `cmd/tui.go` | Create | TUI command entry point |
| `pkg/tui/tui.go` | Create | Main TUI model and logic |
| `pkg/tui/styles.go` | Create | Lipgloss style definitions |
| `pkg/tui/components/build.go` | Create | Build workflow TUI |
| `pkg/tui/components/utm.go` | Create | VM management TUI |
| `pkg/tui/components/sdk.go` | Create | SDK management TUI |
| `pkg/tui/forms/setup.go` | Create | Setup wizard form |

## Verification

1. **Test TUI launches:**
   ```bash
   utm-dev tui
   # Should show interactive menu
   ```

2. **Test keyboard navigation:**
   - Arrow keys navigate
   - Enter selects
   - Escape goes back
   - q quits

3. **Test build workflow:**
   ```bash
   utm-dev tui --section build
   # Select platform and example
   # Verify build runs with progress
   ```

4. **Test VM management:**
   ```bash
   utm-dev tui --section vm
   # Verify VM list shows
   # Test start/stop controls
   ```

5. **Test responsive layout:**
   - Resize terminal window
   - Verify TUI adapts to size

## Consequences

### Benefits
- Better user experience for interactive workflows
- Visual feedback during long operations
- Easier discovery of features
- Setup wizards for new users
- Dashboard view for monitoring

### Trade-offs
- Additional dependencies (Charmbracelet libraries)
- More code to maintain
- Need to keep TUI in sync with CLI features
- Terminal compatibility considerations

### Accessibility Considerations

- Support standard keyboard navigation
- Use clear visual indicators
- Ensure color scheme has sufficient contrast
- Provide non-TUI alternatives for all features

## References

- [Bubbletea Documentation](https://github.com/charmbracelet/bubbletea)
- [Bubbles Components](https://github.com/charmbracelet/bubbles)
- [Lipgloss Styles](https://github.com/charmbracelet/lipgloss)
- [Huh Forms](https://github.com/charmbracelet/huh)
- [Charm Blog - Building TUIs](https://charm.sh/blog/)
