# Lipgloss Documentation Reference

> Style definitions for terminal layouts in Go, similar to CSS.
> Package: `github.com/charmbracelet/lipgloss`

## Overview

Lip Gloss provides an expressive, declarative approach to terminal styling. It works like CSS for the terminal, supporting colors, borders, padding, margins, alignment, and layout composition.

## Core Concepts

### Creating Styles

```go
import "github.com/charmbracelet/lipgloss"

style := lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.Color("#FAFAFA")).
    Background(lipgloss.Color("#7D56F4")).
    PaddingTop(2).
    PaddingLeft(4).
    Width(22)

output := style.Render("Hello, kitty")
fmt.Println(output)
```

Styles are **immutable value types** -- each method returns a new copy.

### Rendering Text

```go
// Render with a style
style.Render("Hello, kitty")

// Set string on the style itself
styledString := lipgloss.NewStyle().
    SetString("Hello").
    Bold(true)
fmt.Println(styledString) // Uses Stringer interface
```

## Color System

### ANSI 16 Colors (4-bit)
```go
lipgloss.Color("5")  // magenta
lipgloss.Color("9")  // red
lipgloss.Color("12") // light blue
```

### ANSI 256 Colors (8-bit)
```go
lipgloss.Color("86")  // aqua
lipgloss.Color("201") // hot pink
lipgloss.Color("202") // orange
```

### True Color (24-bit)
```go
lipgloss.Color("#0000FF") // blue
lipgloss.Color("#04B575") // green
lipgloss.Color("#3C3C3C") // dark gray
```

### Adaptive Colors (Light/Dark Background)
```go
lipgloss.AdaptiveColor{Light: "236", Dark: "248"}
```

### Complete Colors (Per Profile)
```go
lipgloss.CompleteColor{
    TrueColor: "#0000FF",
    ANSI256:   "86",
    ANSI:      "5",
}
```

### Complete Adaptive Colors
```go
lipgloss.CompleteAdaptiveColor{
    Light: lipgloss.CompleteColor{TrueColor: "#d7ffae", ANSI256: "193", ANSI: "11"},
    Dark:  lipgloss.CompleteColor{TrueColor: "#d75fee", ANSI256: "163", ANSI: "5"},
}
```

## Inline Text Formatting

```go
style := lipgloss.NewStyle().
    Bold(true).
    Italic(true).
    Underline(true).
    Strikethrough(true).
    Faint(true).
    Blink(true).
    Reverse(true)
```

## Block-Level Formatting

### Padding

```go
// Individual sides
style := lipgloss.NewStyle().
    PaddingTop(2).
    PaddingRight(4).
    PaddingBottom(2).
    PaddingLeft(4)

// Shorthand (CSS-like: top, right, bottom, left)
lipgloss.NewStyle().Padding(2)          // all sides
lipgloss.NewStyle().Padding(2, 4)       // vertical, horizontal
lipgloss.NewStyle().Padding(1, 4, 2)    // top, horizontal, bottom
lipgloss.NewStyle().Padding(2, 4, 3, 1) // top, right, bottom, left
```

### Margins

```go
// Individual sides
style := lipgloss.NewStyle().
    MarginTop(2).
    MarginRight(4).
    MarginBottom(2).
    MarginLeft(4)

// Shorthand (same as padding)
lipgloss.NewStyle().Margin(2)
lipgloss.NewStyle().Margin(2, 4)
lipgloss.NewStyle().Margin(1, 4, 2)
lipgloss.NewStyle().Margin(2, 4, 3, 1)
```

### Width and Height

```go
style := lipgloss.NewStyle().
    SetString("What's for lunch?").
    Width(24).
    Height(32).
    Foreground(lipgloss.Color("63"))
```

### Text Alignment

```go
style := lipgloss.NewStyle().
    Width(24).
    Align(lipgloss.Left)    // or lipgloss.Right, lipgloss.Center
```

### Rendering Constraints

```go
// Force single line
someStyle.Inline(true).Render("yadda yadda")

// Max width/height
someStyle.MaxWidth(5).MaxHeight(5).Render("yadda yadda")
```

## Borders

### Predefined Border Styles

```go
lipgloss.NormalBorder()   // Standard box drawing
lipgloss.RoundedBorder()  // Rounded corners
lipgloss.ThickBorder()    // Thick lines
lipgloss.DoubleBorder()   // Double lines
lipgloss.BlockBorder()    // Block characters
lipgloss.HiddenBorder()   // Hidden (for spacing)
lipgloss.ASCIIBorder()    // ASCII characters
lipgloss.MarkdownBorder() // Markdown table style
```

### Applying Borders

```go
style := lipgloss.NewStyle().
    BorderStyle(lipgloss.NormalBorder()).
    BorderForeground(lipgloss.Color("63"))

// Selective edges
style := lipgloss.NewStyle().
    BorderStyle(lipgloss.RoundedBorder()).
    BorderForeground(lipgloss.Color("228")).
    BorderBackground(lipgloss.Color("63")).
    BorderTop(true).
    BorderLeft(true)

// Shorthand: Border(style, top, right, bottom, left)
lipgloss.NewStyle().Border(lipgloss.ThickBorder(), true, false) // top+bottom only
lipgloss.NewStyle().Border(lipgloss.DoubleBorder(), true, false, false, true) // top+left
```

### Custom Borders

```go
customBorder := lipgloss.Border{
    Top:         "._.:*:",
    Bottom:      "._.:*:",
    Left:        "|*",
    Right:       "|*",
    TopLeft:     "*",
    TopRight:    "*",
    BottomLeft:  "*",
    BottomRight: "*",
}

style := lipgloss.NewStyle().
    BorderStyle(customBorder).
    BorderForeground(lipgloss.Color("205"))
```

## Layout Composition

### Joining Blocks

```go
// Horizontally (align along bottom edge)
lipgloss.JoinHorizontal(lipgloss.Bottom, block1, block2, block3)

// Horizontally (align 20% from top)
lipgloss.JoinHorizontal(0.2, block1, block2, block3)

// Vertically (center aligned)
lipgloss.JoinVertical(lipgloss.Center, block1, block2, block3)

// Vertically (right aligned)
lipgloss.JoinVertical(lipgloss.Right, block1, block2, block3)
```

### Placing Text in Whitespace

```go
// Center horizontally in 80-cell space
block := lipgloss.PlaceHorizontal(80, lipgloss.Center, paragraph)

// Place at bottom of 30-line tall space
block := lipgloss.PlaceVertical(30, lipgloss.Bottom, paragraph)

// Place in bottom-right corner of 30x80 space
block := lipgloss.Place(80, 30, lipgloss.Right, lipgloss.Bottom, paragraph)

// With styled whitespace
block := lipgloss.Place(80, 20, lipgloss.Center, lipgloss.Center, text,
    lipgloss.WithWhitespaceBackground(lipgloss.Color("236")),
    lipgloss.WithWhitespaceChars("."),
)
```

### Measuring Dimensions

```go
width := lipgloss.Width(renderedBlock)
height := lipgloss.Height(renderedBlock)
w, h := lipgloss.Size(renderedBlock)
```

## Style Inheritance and Composition

### Copying Styles

```go
style := lipgloss.NewStyle().Foreground(lipgloss.Color("219"))
copiedStyle := style                  // True copy (value type)
wildStyle := style.Blink(true)        // New copy with blink
```

### Inheriting Styles

```go
baseStyle := lipgloss.NewStyle().
    Foreground(lipgloss.Color("229")).
    Background(lipgloss.Color("63"))

// Only unset rules are inherited
childStyle := lipgloss.NewStyle().
    Foreground(lipgloss.Color("201")). // Won't be overridden
    Inherit(baseStyle)                  // Gets background from base
```

### Unsetting Rules

```go
style := lipgloss.NewStyle().
    Bold(true).
    UnsetBold().
    Background(lipgloss.Color("227")).
    UnsetBackground()
```

### Transform Function

```go
style := lipgloss.NewStyle().
    Transform(func(s string) string {
        return ">> " + s + " <<"
    })
```

## Tables (lipgloss/table)

```go
import (
    "github.com/charmbracelet/lipgloss"
    "github.com/charmbracelet/lipgloss/table"
)

rows := [][]string{
    {"Chinese", "Ni Hao", "Ni Hao"},
    {"Japanese", "Konnichiwa", "Yaa"},
    {"Spanish", "Hola", "Que tal?"},
}

purple := lipgloss.Color("99")
headerStyle := lipgloss.NewStyle().Foreground(purple).Bold(true).Align(lipgloss.Center)
cellStyle := lipgloss.NewStyle().Padding(0, 1).Width(14)

t := table.New().
    Border(lipgloss.NormalBorder()).
    BorderStyle(lipgloss.NewStyle().Foreground(purple)).
    StyleFunc(func(row, col int) lipgloss.Style {
        switch {
        case row == table.HeaderRow:
            return headerStyle
        case row%2 == 0:
            return cellStyle.Foreground(lipgloss.Color("245"))
        default:
            return cellStyle.Foreground(lipgloss.Color("241"))
        }
    }).
    Headers("LANGUAGE", "FORMAL", "INFORMAL").
    Rows(rows...)

// Add individual rows
t.Row("English", "Hello", "Hey")

// With width constraints
t.Width(80)

fmt.Println(t)
```

### Markdown and ASCII Tables

```go
// Markdown style
table.New().Border(lipgloss.MarkdownBorder()).BorderTop(false).BorderBottom(false)

// ASCII style
table.New().Border(lipgloss.ASCIIBorder())
```

## Lists (lipgloss/list)

```go
import "github.com/charmbracelet/lipgloss/list"

// Simple list
l := list.New("A", "B", "C")

// Nested list
l := list.New(
    "Produce",
    list.New("Apples", "Bananas", "Carrots"),
    "Dairy",
    list.New("Milk", "Cheese"),
)

// Styled with Roman numerals
l := list.New("Glossier", "Nyx", "Mac").
    Enumerator(list.Roman).
    EnumeratorStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("99")).MarginRight(1)).
    ItemStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("212")))

// Custom enumerator
l.Enumerator(func(items list.Items, i int) string {
    if items.At(i).Value() == "Goose" {
        return "Honk ->"
    }
    return "Quack ->"
})

// Dynamic building
l := list.New()
for i := 0; i < n; i++ {
    l.Item("Item")
}
```

## Trees (lipgloss/tree)

```go
import "github.com/charmbracelet/lipgloss/tree"

// Simple tree
t := tree.Root(".").Child("A", "B", "C")

// Nested tree
t := tree.Root(".").
    Child("README.md").
    Child(
        tree.Root("src").Child("main.go", "utils.go"),
    ).
    Child(
        tree.Root("tests").Child("main_test.go"),
    )

// Styled tree
t := tree.Root("Project").
    Child("Config", "Source", "Tests").
    Enumerator(tree.RoundedEnumerator).
    EnumeratorStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("63")).MarginRight(1)).
    RootStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("35")).Bold(true)).
    ItemStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("212")))
```

## Custom Renderers

For server-client scenarios (e.g., SSH):

```go
func myHandler(sess ssh.Session) {
    renderer := lipgloss.NewRenderer(sess)
    style := renderer.NewStyle().
        Background(lipgloss.AdaptiveColor{Light: "63", Dark: "228"})
    io.WriteString(sess, style.Render("Hello"))
}
```

## Tab Width

```go
style := lipgloss.NewStyle()
style = style.TabWidth(2)                       // 2 spaces per tab
style = style.TabWidth(0)                       // No tab conversion
style = style.TabWidth(lipgloss.NoTabConversion) // Disable conversion
```

## Best Practices for audeck

1. **Use AdaptiveColor** for colors that need to work on both light and dark terminals
2. **Compose layouts with Join functions** rather than manual string concatenation
3. **Use Place functions** for centering and positioning within available space
4. **Measure rendered output** with `Width()` and `Height()` before composing layouts
5. **Create style constants** at package level for reuse across views
6. **Use border styles** consistently for visual hierarchy
7. **Handle WindowSizeMsg** to make layouts responsive to terminal size
8. **Use lipgloss/table** for structured data display (audio tracks, settings)
