# Bubbles Documentation Reference

> Reusable TUI components for Bubble Tea applications.
> Package: `github.com/charmbracelet/bubbles`

## Overview

Bubbles provides a collection of pre-built, interactive components that follow the Bubble Tea Model-View-Update pattern. Each component is a `tea.Model` that can be embedded in your application model.

## Installation

```bash
go get github.com/charmbracelet/bubbles
```

## Component Pattern

All Bubbles components follow the same pattern:

1. Create the component model (e.g., `spinner.New()`)
2. Embed it in your application model
3. Initialize it in `Init()` if needed
4. Forward messages to it in `Update()`
5. Render it in `View()`

```go
type model struct {
    component component.Model  // Embed the component
}

func (m model) Init() tea.Cmd {
    return m.component.Init()  // or component-specific init like spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    m.component, cmd = m.component.Update(msg)
    return m, cmd
}

func (m model) View() string {
    return m.component.View()
}
```

---

## Spinner

Animated loading indicators with multiple built-in styles.

```go
import "github.com/charmbracelet/bubbles/spinner"

// Create spinner
s := spinner.New(
    spinner.WithSpinner(spinner.Dot),
    spinner.WithStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("205"))),
)

// Available styles:
// spinner.Line, spinner.Dot, spinner.MiniDot, spinner.Jump
// spinner.Pulse, spinner.Points, spinner.Globe, spinner.Moon
// spinner.Monkey, spinner.Meter, spinner.Hamburger, spinner.Ellipsis
```

**Init:** Return `m.spinner.Tick`
**Update:** Forward `spinner.TickMsg` to spinner
**View:** `m.spinner.View()` returns the current frame

```go
func (m model) Init() tea.Cmd {
    return m.spinner.Tick
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case spinner.TickMsg:
        var cmd tea.Cmd
        m.spinner, cmd = m.spinner.Update(msg)
        return m, cmd
    }
    return m, nil
}

func (m model) View() string {
    return fmt.Sprintf("%s Loading...\n", m.spinner.View())
}
```

---

## Text Input (Single Line)

Single-line text input with placeholder, character limit, suggestions, and password masking.

```go
import "github.com/charmbracelet/bubbles/textinput"

ti := textinput.New()
ti.Placeholder = "Enter your name"
ti.Focus()
ti.CharLimit = 50
ti.Width = 30
ti.PromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
ti.TextStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

// Auto-completion
ti.ShowSuggestions = true
ti.SetSuggestions([]string{"Alice", "Bob", "Charlie"})

// Password mode
ti.EchoMode = textinput.EchoPassword
ti.EchoCharacter = '*'
```

**Init:** Return `textinput.Blink`
**Key methods:** `ti.Value()`, `ti.SetValue(s)`, `ti.Focus()`, `ti.Blur()`

```go
func (m model) Init() tea.Cmd {
    return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.Type {
        case tea.KeyEnter:
            return m, tea.Quit
        case tea.KeyCtrlC, tea.KeyEsc:
            return m, tea.Quit
        }
    }
    m.textInput, cmd = m.textInput.Update(msg)
    return m, cmd
}

func (m model) View() string {
    return fmt.Sprintf("Name:\n\n%s\n\n(enter to submit)", m.textInput.View())
}
```

---

## Text Area (Multi-Line)

Multi-line text editor with line numbers, word wrap, and scrolling.

```go
import "github.com/charmbracelet/bubbles/textarea"

ta := textarea.New()
ta.Placeholder = "Write your story..."
ta.Focus()
ta.SetWidth(60)
ta.SetHeight(10)
ta.ShowLineNumbers = true
ta.CharLimit = 1000
ta.MaxHeight = 20
ta.Prompt = "| "
```

**Init:** Return `textarea.Blink`
**Key methods:** `ta.Value()`, `ta.SetValue(s)`, `ta.Focus()`, `ta.Blur()`, `ta.Length()`, `ta.LineCount()`

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd
    var cmd tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.Type {
        case tea.KeyEsc:
            if m.textarea.Focused() {
                m.textarea.Blur()
            }
        case tea.KeyCtrlC:
            return m, tea.Quit
        default:
            if !m.textarea.Focused() {
                cmd = m.textarea.Focus()
                cmds = append(cmds, cmd)
            }
        }
    }

    m.textarea, cmd = m.textarea.Update(msg)
    cmds = append(cmds, cmd)
    return m, tea.Batch(cmds...)
}
```

---

## Table

Interactive table with column definitions, row selection, and keyboard navigation.

```go
import "github.com/charmbracelet/bubbles/table"

columns := []table.Column{
    {Title: "Rank", Width: 6},
    {Title: "City", Width: 15},
    {Title: "Country", Width: 15},
    {Title: "Population", Width: 12},
}

rows := []table.Row{
    {"1", "Tokyo", "Japan", "37,400,068"},
    {"2", "Delhi", "India", "28,514,000"},
    {"3", "Shanghai", "China", "25,582,000"},
}

t := table.New(
    table.WithColumns(columns),
    table.WithRows(rows),
    table.WithFocused(true),
    table.WithHeight(7),
)

// Custom styles
s := table.DefaultStyles()
s.Header = s.Header.
    BorderStyle(lipgloss.NormalBorder()).
    BorderForeground(lipgloss.Color("240")).
    BorderBottom(true).
    Bold(false)
s.Selected = s.Selected.
    Foreground(lipgloss.Color("229")).
    Background(lipgloss.Color("57")).
    Bold(false)
t.SetStyles(s)
```

**Key methods:** `t.SelectedRow()`, `t.SetRows(rows)`, `t.SetColumns(cols)`, `t.Focus()`, `t.Blur()`
**Navigation:** Up/Down arrows, Page Up/Down, Home/End

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmd tea.Cmd
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "enter":
            return m, tea.Printf("Selected: %s", m.table.SelectedRow()[1])
        case "q":
            return m, tea.Quit
        }
    }
    m.table, cmd = m.table.Update(msg)
    return m, cmd
}
```

---

## List

Feature-rich list with fuzzy filtering, pagination, help, and status messages.

```go
import "github.com/charmbracelet/bubbles/list"

// Item interface
type item struct {
    title, desc string
}
func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

// Create list
items := []list.Item{
    item{title: "Go", desc: "Compiled language by Google"},
    item{title: "Rust", desc: "Systems programming language"},
}

l := list.New(items, list.NewDefaultDelegate(), 0, 0)
l.Title = "My List"
l.SetShowStatusBar(true)
l.SetFilteringEnabled(true)
l.SetShowHelp(true)
```

**Handle window resize:**
```go
case tea.WindowSizeMsg:
    h, v := docStyle.GetFrameSize()
    m.list.SetSize(msg.Width-h, msg.Height-v)
```

**Key methods:** `l.SelectedItem()`, `l.SetItems(items)`, `l.SetSize(w, h)`, `l.FilterInput.Value()`

### Custom Item Delegate Styling

```go
d := list.NewDefaultDelegate()
c := lipgloss.Color("#6f03fc")
d.Styles.SelectedTitle = d.Styles.SelectedTitle.Foreground(c).BorderLeftForeground(c)
d.Styles.SelectedDesc = d.Styles.SelectedTitle

l := list.New(items, d, width, height)
```

---

## Viewport

Scrollable content area for displaying large text.

```go
import "github.com/charmbracelet/bubbles/viewport"

// Create on WindowSizeMsg
case tea.WindowSizeMsg:
    headerHeight := 1
    footerHeight := 1
    verticalMargin := headerHeight + footerHeight

    if !m.ready {
        m.viewport = viewport.New(msg.Width, msg.Height-verticalMargin)
        m.viewport.YPosition = headerHeight
        m.viewport.SetContent(m.content)
        m.viewport.MouseWheelEnabled = true
        m.viewport.MouseWheelDelta = 3
        m.ready = true
    } else {
        m.viewport.Width = msg.Width
        m.viewport.Height = msg.Height - verticalMargin
    }
```

**Key methods:** `vp.SetContent(s)`, `vp.ScrollPercent()`, `vp.GotoTop()`, `vp.GotoBottom()`
**Navigation:** Up/Down, Page Up/Down, Mouse wheel

```go
func (m model) View() string {
    if !m.ready {
        return "\n  Initializing..."
    }
    header := titleStyle.Render("Viewport Demo")
    footer := infoStyle.Render(fmt.Sprintf("%3.f%%", m.viewport.ScrollPercent()*100))
    return fmt.Sprintf("%s\n%s\n%s", header, m.viewport.View(), footer)
}
```

---

## Progress Bar

Visual progress indicator with solid and gradient fills.

```go
import "github.com/charmbracelet/bubbles/progress"

// Gradient fill
p := progress.New(
    progress.WithDefaultGradient(),
    progress.WithWidth(40),
)

// Solid fill
p := progress.New(
    progress.WithSolidFill("#7D56F4"),
    progress.WithWidth(40),
)
```

**Rendering options:**
```go
// Static rendering (no animation)
p.ViewAs(0.5) // 50% complete

// Animated rendering (returns a Cmd)
cmd := p.SetPercent(0.75)  // Smooth spring animation to 75%
```

**Handle animation frames:**
```go
case progress.FrameMsg:
    progressModel, cmd := m.progress.Update(msg)
    m.progress = progressModel.(progress.Model)
    return m, cmd
```

---

## Paginator

Page navigation for sliced data.

```go
import "github.com/charmbracelet/bubbles/paginator"

p := paginator.New()
p.Type = paginator.Dots      // or paginator.Arabic for "1/5"
p.PerPage = 10
p.SetTotalPages(len(items))
p.ActiveDot = "●"           // bullet character
p.InactiveDot = "○"          // empty circle
```

**Get current page items:**
```go
start, end := m.paginator.GetSliceBounds(len(m.items))
currentItems := m.items[start:end]
```

**Navigation:** h/l or Left/Right arrows

---

## Timer

Countdown timer with start/stop/toggle controls.

```go
import "github.com/charmbracelet/bubbles/timer"

// Basic timer
t := timer.New(10 * time.Second)

// With custom tick interval
t := timer.NewWithInterval(10*time.Second, 100*time.Millisecond)
```

**Key methods:** `t.Toggle()`, `t.Start()`, `t.Stop()`, `t.Running()`
**Messages:** `timer.TickMsg`, `timer.TimeoutMsg`, `timer.StartStopMsg`

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case timer.TickMsg:
        var cmd tea.Cmd
        m.timer, cmd = m.timer.Update(msg)
        return m, cmd
    case timer.TimeoutMsg:
        return m, tea.Quit  // Timer finished
    case tea.KeyMsg:
        if msg.String() == " " {
            return m, m.timer.Toggle()
        }
    }
    return m, nil
}
```

---

## Stopwatch

Counts up from zero with start/stop/reset.

```go
import "github.com/charmbracelet/bubbles/stopwatch"

sw := stopwatch.New()
```

**Init:** Return `m.stopwatch.Init()`
**Key methods:** `sw.Toggle()`, `sw.Reset()`, `sw.Running()`
**Messages:** `stopwatch.TickMsg`, `stopwatch.StartStopMsg`, `stopwatch.ResetMsg`

---

## Help

Displays keyboard shortcuts in short or full format.

```go
import (
    "github.com/charmbracelet/bubbles/help"
    "github.com/charmbracelet/bubbles/key"
)

// Define key bindings
type keyMap struct {
    Up   key.Binding
    Down key.Binding
    Help key.Binding
    Quit key.Binding
}

// Implement help.KeyMap interface
func (k keyMap) ShortHelp() []key.Binding {
    return []key.Binding{k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
    return [][]key.Binding{
        {k.Up, k.Down},
        {k.Help, k.Quit},
    }
}

// Create bindings
keys := keyMap{
    Up:   key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("up/k", "move up")),
    Down: key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("down/j", "move down")),
    Help: key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "toggle help")),
    Quit: key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
}

h := help.New()
```

**Rendering:** `h.View(keys)` renders short or full help based on `h.ShowAll`
**Key matching:** `key.Matches(msg, keys.Quit)` checks if a key press matches a binding
**Enable/Disable:** `keys.Delete.SetEnabled(false)` / `keys.Delete.Enabled()`

---

## File Picker

File system browser with directory navigation and filtering.

```go
import "github.com/charmbracelet/bubbles/filepicker"

fp := filepicker.New()
fp.CurrentDirectory = "."
fp.AllowedTypes = []string{".go", ".mod", ".sum", ".md"}
fp.ShowPermissions = true
fp.ShowSize = true
fp.ShowHidden = false
fp.DirAllowed = false
fp.FileAllowed = true
fp.Height = 20
```

**Init:** Return `m.filepicker.Init()`
**Check selection:**
```go
if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
    m.selectedFile = path
}

if didSelect, path := m.filepicker.DidSelectDisabledFile(msg); didSelect {
    m.err = fmt.Errorf("file type not allowed: %s", path)
}
```

---

## Key Bindings (bubbles/key)

Utility for defining and matching key bindings.

```go
import "github.com/charmbracelet/bubbles/key"

binding := key.NewBinding(
    key.WithKeys("ctrl+s", "s"),      // Keys that trigger this binding
    key.WithHelp("ctrl+s/s", "save"), // Help text
)

// Check if a key matches
if key.Matches(msg, binding) {
    // handle save
}

// Enable/disable
binding.SetEnabled(false)
binding.Enabled() // false
```

---

## Relevant Components for audeck

For a TUI audio application, the most relevant components are:

| Component | Use Case in audeck |
|-----------|-------------------|
| **Viewport** | Scrollable waveform display, log output |
| **List** | Track/file browser, effect chain listing |
| **Table** | Audio device settings, track properties |
| **Progress** | Playback position, loading indicator |
| **Spinner** | Loading/processing indicator |
| **Text Input** | File name entry, search, parameter values |
| **Help** | Keyboard shortcut reference |
| **Key Bindings** | Consistent keybinding management |
| **Timer/Stopwatch** | Recording duration, playback time display |
| **Paginator** | Multi-page settings or track lists |
