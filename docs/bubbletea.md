# Bubbletea Documentation Reference

> Core TUI framework for building terminal applications in Go.
> Package: `github.com/charmbracelet/bubbletea`

## Architecture: The Elm Architecture (Model-View-Update)

Bubble Tea implements the Elm Architecture (TEA), a simple yet powerful pattern:

1. **Model** - Your application state
2. **Update** - A function that updates the state based on messages
3. **View** - A function that renders the UI based on the state

```
User Input -> Msg -> Update(model, msg) -> (model, Cmd) -> View(model) -> string -> Terminal
```

## Core Interface

Every Bubble Tea program implements the `Model` interface:

```go
type Model interface {
    // Init is the first function called. Returns an optional initial command.
    Init() Cmd

    // Update handles incoming messages, updates state, and optionally returns commands.
    Update(Msg) (Model, Cmd)

    // View renders the UI as a string. Called after every Update.
    View() string
}
```

## Key Types

### Msg (Message)

```go
type Msg interface{}
```

Messages are events that trigger the Update function. They can come from:
- User input (keyboard, mouse)
- I/O operations (HTTP responses, file reads)
- Timers and ticks
- Window resize events
- Custom application messages

### Cmd (Command)

```go
type Cmd func() Msg
```

Commands are functions that perform I/O and return a message. A nil Cmd means "no operation."

### Program

```go
type Program struct { /* unexported fields */ }

func NewProgram(model Model, opts ...ProgramOption) *Program
func (p *Program) Run() (returnModel Model, returnErr error)
func (p *Program) Send(msg Msg)
func (p *Program) Quit()
func (p *Program) Kill()
func (p *Program) Wait()
```

## Complete Example: Basic Application

```go
package main

import (
    "fmt"
    "os"
    tea "github.com/charmbracelet/bubbletea"
)

type model struct {
    choices  []string
    cursor   int
    selected map[int]struct{}
}

func initialModel() model {
    return model{
        choices:  []string{"Buy carrots", "Buy celery", "Buy kohlrabi"},
        selected: make(map[int]struct{}),
    }
}

func (m model) Init() tea.Cmd {
    return nil // No initial I/O
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "ctrl+c", "q":
            return m, tea.Quit
        case "up", "k":
            if m.cursor > 0 {
                m.cursor--
            }
        case "down", "j":
            if m.cursor < len(m.choices)-1 {
                m.cursor++
            }
        case "enter", " ":
            _, ok := m.selected[m.cursor]
            if ok {
                delete(m.selected, m.cursor)
            } else {
                m.selected[m.cursor] = struct{}{}
            }
        }
    }
    return m, nil
}

func (m model) View() string {
    s := "What should we buy at the market?\n\n"
    for i, choice := range m.choices {
        cursor := " "
        if m.cursor == i {
            cursor = ">"
        }
        checked := " "
        if _, ok := m.selected[i]; ok {
            checked = "x"
        }
        s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, choice)
    }
    s += "\nPress q to quit.\n"
    return s
}

func main() {
    p := tea.NewProgram(initialModel())
    if _, err := p.Run(); err != nil {
        fmt.Printf("Alas, there's been an error: %v", err)
        os.Exit(1)
    }
}
```

## Message Types

### Key Messages

```go
// v1 style
case tea.KeyMsg:
    switch msg.String() {
    case "ctrl+c", "q":
        return m, tea.Quit
    case "up", "k":
        // handle up
    case "enter", " ":
        // handle selection
    }

// v2 style (KeyPressMsg / KeyReleaseMsg)
case tea.KeyPressMsg:
    switch msg.String() {
    case "ctrl+c", "q":
        return m, tea.Quit
    }
```

Type-based comparison (more robust):
```go
case tea.KeyMsg:
    switch msg.Type {
    case tea.KeyEnter:
        m.submit()
    case tea.KeyEsc:
        m.cancel()
    case tea.KeyRunes:
        m.input += string(msg.Runes)
    case tea.KeyBackspace:
        if len(m.input) > 0 {
            m.input = m.input[:len(m.input)-1]
        }
    }
```

### Window Size Messages

```go
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height
    m.ready = true
```

Sent automatically on program start and window resize.

### Mouse Messages

```go
case tea.MouseMsg:
    m.x = msg.X
    m.y = msg.Y
    switch msg.Button {
    case tea.MouseButtonLeft:
        if msg.Action == tea.MouseActionPress {
            // handle left click
        }
    case tea.MouseButtonWheelUp:
        // handle scroll up
    case tea.MouseButtonWheelDown:
        // handle scroll down
    }
```

### Focus/Blur Messages

```go
case tea.FocusMsg:
    m.focused = true
case tea.BlurMsg:
    m.focused = false
```

Requires `tea.WithReportFocus()` program option.

## Commands

### Built-in Commands

```go
tea.Quit        // Exit the program
tea.Suspend     // Suspend the program (ctrl+z)
tea.ClearScreen // Clear the terminal screen
```

### Batch Commands (Concurrent)

```go
func (m model) Init() tea.Cmd {
    return tea.Batch(someCommand, someOtherCommand)
}
```

### Sequence Commands (Sequential)

```go
cmd := tea.Sequence(
    fetchData1,
    fetchData2,
)
```

### Custom Commands (I/O Operations)

```go
type statusMsg int
type errMsg struct{ error }

func checkServer(url string) tea.Cmd {
    return func() tea.Msg {
        c := &http.Client{Timeout: 10 * time.Second}
        res, err := c.Get(url)
        if err != nil {
            return errMsg{err}
        }
        defer res.Body.Close()
        return statusMsg(res.StatusCode)
    }
}
```

### Tick / Every (Timers)

```go
type TickMsg time.Time

// One-shot timer (re-issue to loop)
func doTick() tea.Cmd {
    return tea.Tick(time.Second, func(t time.Time) tea.Msg {
        return TickMsg(t)
    })
}

// System clock-aligned interval
func tickEvery() tea.Cmd {
    return tea.Every(time.Second, func(t time.Time) tea.Msg {
        return TickMsg(t)
    })
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg.(type) {
    case TickMsg:
        // Re-issue to continue ticking
        return m, doTick()
    }
    return m, nil
}
```

### External Process Execution

```go
type editorFinishedMsg struct{ err error }

c := exec.Command("vim", "file.txt")
cmd := tea.ExecProcess(c, func(err error) tea.Msg {
    return editorFinishedMsg{err}
})
```

### Window Title

```go
func (m model) Init() tea.Cmd {
    return tea.SetWindowTitle("My App")
}
```

## Program Options

```go
p := tea.NewProgram(
    model{},
    tea.WithAltScreen(),          // Fullscreen mode
    tea.WithMouseCellMotion(),    // Mouse click/release/wheel/drag
    tea.WithMouseAllMotion(),     // All mouse events including hover
    tea.WithReportFocus(),        // Focus/blur events
    tea.WithInput(os.Stdin),      // Custom input source
    tea.WithOutput(os.Stdout),    // Custom output destination
    tea.WithContext(ctx),         // Context for cancellation
    tea.WithFPS(30),             // Custom FPS (default 60, max 120)
    tea.WithFilter(filterFn),     // Message filter
)
```

### Message Filtering

```go
func filter(m tea.Model, msg tea.Msg) tea.Msg {
    if _, ok := msg.(tea.QuitMsg); !ok {
        return msg
    }
    model := m.(myModel)
    if model.hasChanges {
        return nil // Block quit with unsaved changes
    }
    return msg
}

p := tea.NewProgram(Model{}, tea.WithFilter(filter))
```

## External Control

```go
// Send messages from outside the program
p.Send(externalMsg("Hello from outside!"))

// Print above the TUI (persistent output)
p.Println("Status update: all systems go")
p.Printf("Time: %s", time.Now().Format(time.RFC3339))

// Graceful quit
p.Quit()

// Immediate termination
p.Kill()

// Wait for program to finish
p.Wait()
```

## Debugging

```go
// Log to file (since stdout is occupied by TUI)
if len(os.Getenv("DEBUG")) > 0 {
    f, err := tea.LogToFile("debug.log", "debug")
    if err != nil {
        fmt.Println("fatal:", err)
        os.Exit(1)
    }
    defer f.Close()
}

// Now log.Print writes to debug.log
log.Println("Starting application")
```

## Best Practices

1. **Keep Model simple** - Store only the state needed for rendering
2. **Commands for I/O** - Never do I/O directly in Update; return Commands instead
3. **Type switches for messages** - Use `switch msg := msg.(type)` pattern
4. **Return new model** - Update should return a new model, not mutate in place
5. **Nil for no-ops** - Return nil instead of a Cmd when no I/O is needed
6. **Handle WindowSizeMsg** - Always handle terminal resize for responsive layouts
7. **Use AltScreen** - For fullscreen apps, use `tea.WithAltScreen()`
8. **Batch for concurrent** - Use `tea.Batch()` for parallel operations
9. **Sequence for ordered** - Use `tea.Sequence()` for sequential operations
10. **Debug with log file** - Use `tea.LogToFile()` since stdout is occupied

## v2 Changes (Beta)

In Bubbletea v2:
- `Init()` returns `(Model, Cmd)` instead of just `Cmd`
- `KeyMsg` is split into `KeyPressMsg` and `KeyReleaseMsg`
- `Msg` is a type alias for `uv.Event`
- `View` is a struct with layers, cursor, colors, and display modes
- `BatchMsg` type for batch commands
- New `View` struct with `SetContent(s any)` and `NewView(s any) View`
- `Quit()`, `ClearScreen()`, etc. return `Msg` directly instead of `Cmd`
