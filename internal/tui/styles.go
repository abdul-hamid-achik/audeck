package tui

import "github.com/charmbracelet/lipgloss"

// Catppuccin Mocha palette.
var (
	colorBase     = lipgloss.Color("#1e1e2e")
	colorMantle   = lipgloss.Color("#181825")
	colorCrust    = lipgloss.Color("#11111b")
	colorSurface0 = lipgloss.Color("#313244")
	colorSurface1 = lipgloss.Color("#45475a")
	colorSurface2 = lipgloss.Color("#585b70")
	colorOverlay0 = lipgloss.Color("#6c7086")
	colorOverlay1 = lipgloss.Color("#7f849c")
	colorText     = lipgloss.Color("#cdd6f4")
	colorSubtext0 = lipgloss.Color("#a6adc8")
	colorSubtext1 = lipgloss.Color("#bac2de")
	colorLavender = lipgloss.Color("#b4befe")
	colorBlue     = lipgloss.Color("#89b4fa")
	colorSapphire = lipgloss.Color("#74c7ec")
	colorTeal     = lipgloss.Color("#94e2d5")
	colorGreen    = lipgloss.Color("#a6e3a1")
	colorYellow   = lipgloss.Color("#f9e2af")
	colorPeach    = lipgloss.Color("#fab387")
	colorMaroon   = lipgloss.Color("#eba0ac")
	colorRed      = lipgloss.Color("#f38ba8")
	colorMauve    = lipgloss.Color("#cba6f7")
	colorPink     = lipgloss.Color("#f5c2e7")
)

// Application styles.
var (
	// Title bar at the top of the application.
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorMauve).
			Background(colorMantle).
			Padding(0, 2).
			MarginBottom(1)

	// Active tab in the tab bar.
	activeTabStyle = lipgloss.NewStyle().
			Foreground(colorBase).
			Background(colorMauve).
			Bold(true).
			Padding(0, 1)

	// Inactive tab in the tab bar.
	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(colorSubtext0).
				Background(colorSurface0).
				Padding(0, 1)

	// The outer container for the device list.
	listContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorSurface2).
				Padding(0, 1)

	// A normal (unselected) device row.
	deviceRowStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 1)

	// The currently highlighted device row.
	selectedDeviceRowStyle = lipgloss.NewStyle().
				Foreground(colorBase).
				Background(colorMauve).
				Bold(true).
				Padding(0, 1)

	// Device name within a normal row.
	deviceNameStyle = lipgloss.NewStyle().
			Foreground(colorText)

	// Device name within the selected row.
	selectedDeviceNameStyle = lipgloss.NewStyle().
				Foreground(colorBase).
				Background(colorMauve).
				Bold(true)

	// Transport type label (e.g., "USB", "BT").
	transportStyle = lipgloss.NewStyle().
			Foreground(colorOverlay1).
			Width(5)

	// Transport type label on selected row.
	selectedTransportStyle = lipgloss.NewStyle().
				Foreground(colorSurface1).
				Background(colorMauve).
				Width(5)

	// Default device indicator.
	defaultIndicatorStyle = lipgloss.NewStyle().
				Foreground(colorGreen).
				Bold(true)

	// Default indicator on the selected row.
	selectedDefaultIndicatorStyle = lipgloss.NewStyle().
					Foreground(colorBase).
					Background(colorMauve).
					Bold(true)

	// Volume bar filled portion.
	volumeFilledStyle = lipgloss.NewStyle().
				Foreground(colorBlue)

	// Volume bar empty portion.
	volumeEmptyStyle = lipgloss.NewStyle().
				Foreground(colorSurface1)

	// Volume bar on selected row - filled.
	selectedVolumeFilledStyle = lipgloss.NewStyle().
					Foreground(colorBase).
					Background(colorMauve)

	// Volume bar on selected row - empty.
	selectedVolumeEmptyStyle = lipgloss.NewStyle().
				Foreground(colorSurface2).
				Background(colorMauve)

	// Muted volume indicator.
	mutedStyle = lipgloss.NewStyle().
			Foreground(colorRed)

	// Muted indicator on selected row.
	selectedMutedStyle = lipgloss.NewStyle().
				Foreground(colorBase).
				Background(colorMauve).
				Bold(true)

	// Footer help text.
	helpStyle = lipgloss.NewStyle().
			Foreground(colorOverlay0).
			MarginTop(1).
			Padding(0, 1)

	// Help key styling.
	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorSubtext0).
			Bold(true)

	// Help description styling.
	helpDescStyle = lipgloss.NewStyle().
			Foreground(colorOverlay0)

	// Error message styling.
	errorStyle = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true).
			Padding(0, 1)

	// Status bar for transient messages (success).
	statusStyle = lipgloss.NewStyle().
			Foreground(colorYellow).
			Padding(0, 1)

	// Status bar for transient error messages.
	statusErrorStyle = lipgloss.NewStyle().
				Foreground(colorRed).
				Padding(0, 1)

	// Scroll indicator arrows.
	scrollIndicatorStyle = lipgloss.NewStyle().
				Foreground(colorOverlay0).
				Align(lipgloss.Center)
)
