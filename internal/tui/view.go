package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/abdulachik/audeck/internal/audio"
)

// View renders the complete UI.
func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	// Build sections.
	title := m.renderTitle()
	tabs := m.renderTabs()
	deviceList := m.renderDeviceList()
	status := m.renderStatus()
	help := m.renderHelp()

	// Compose the full layout.
	sections := []string{title, tabs, deviceList}
	if status != "" {
		sections = append(sections, status)
	}
	sections = append(sections, help)
	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Center in the terminal.
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m Model) renderTitle() string {
	title := titleStyle.Render("  audeck  Audio Device Manager")

	// Right-align device count.
	devices := m.activeDevices()
	count := helpDescStyle.Render(fmt.Sprintf("%d devices", len(devices)))
	gap := ""
	titleW := lipgloss.Width(title)
	countW := lipgloss.Width(count)
	listWidth := m.listWidth()
	if remaining := listWidth - titleW - countW; remaining > 0 {
		gap = strings.Repeat(" ", remaining)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, title, gap, count)
}

func (m Model) renderTabs() string {
	outputLabel := " Output "
	inputLabel := " Input "

	var outputTab, inputTab string
	if m.activeTab == tabOutput {
		outputTab = activeTabStyle.Render(outputLabel)
		inputTab = inactiveTabStyle.Render(inputLabel)
	} else {
		outputTab = inactiveTabStyle.Render(outputLabel)
		inputTab = activeTabStyle.Render(inputLabel)
	}

	return lipgloss.JoinHorizontal(lipgloss.Bottom, outputTab, " ", inputTab)
}

func (m Model) renderDeviceList() string {
	devices := m.activeDevices()
	if len(devices) == 0 {
		label := "output"
		if m.activeTab == tabInput {
			label = "input"
		}
		empty := errorStyle.Render(fmt.Sprintf("No %s devices found", label))
		return listContainerStyle.Width(m.listWidth()).Render(empty)
	}

	visible := m.visibleDeviceCount()
	start := m.scrollOffset
	end := start + visible
	if end > len(devices) {
		end = len(devices)
	}

	var rows []string

	// Scroll up indicator.
	if start > 0 {
		indicator := scrollIndicatorStyle.Width(m.listWidth() - 4).Render("▲ more")
		rows = append(rows, indicator)
	}

	for i := start; i < end; i++ {
		selected := i == m.cursor
		rows = append(rows, m.renderDeviceRow(devices[i], selected))
	}

	// Scroll down indicator.
	if end < len(devices) {
		indicator := scrollIndicatorStyle.Width(m.listWidth() - 4).Render("▼ more")
		rows = append(rows, indicator)
	}

	list := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return listContainerStyle.Width(m.listWidth()).Render(list)
}

func (m Model) renderDeviceRow(d device, selected bool) string {
	rowWidth := m.listWidth() - 4 // total row width (fits in container content area)
	w := rowWidth - 2             // content width (subtract row Padding(0,1) = 2 chars)

	// Transport icon.
	icon := transportIcon(d.TransportType)
	var iconStr string
	if selected {
		iconStr = selectedTransportStyle.Render(icon)
	} else {
		iconStr = transportStyle.Render(icon)
	}

	// Default indicator.
	var defaultStr string
	if d.IsDefault {
		if selected {
			defaultStr = selectedDefaultIndicatorStyle.Render("*")
		} else {
			defaultStr = defaultIndicatorStyle.Render("*")
		}
	} else {
		defaultStr = " "
	}

	// Device name.
	var nameStr string
	if selected {
		nameStr = selectedDeviceNameStyle.Render(d.Name)
	} else {
		nameStr = deviceNameStyle.Render(d.Name)
	}

	// Volume bar or N/A.
	var rightPart string
	if !d.HasVolume {
		na := "N/A"
		if selected {
			rightPart = selectedDeviceNameStyle.Render(na)
		} else {
			rightPart = helpDescStyle.Render(na)
		}
	} else {
		volBarWidth := 20
		volStr := renderVolumeBar(d.Volume, d.Muted, volBarWidth, selected)

		var pctStr string
		if d.Muted {
			if selected {
				pctStr = selectedMutedStyle.Render("MUTE")
			} else {
				pctStr = mutedStyle.Render("MUTE")
			}
		} else {
			pct := fmt.Sprintf("%3d%%", int(math.Round(float64(d.Volume)*100)))
			if selected {
				pctStr = selectedDeviceNameStyle.Render(pct)
			} else {
				pctStr = helpDescStyle.Render(pct)
			}
		}
		rightPart = fmt.Sprintf("%s %s", volStr, pctStr)
	}

	// Compose the row: [default] [icon] [name ... gap ... volume pct]
	leftPart := fmt.Sprintf("%s %s %s", defaultStr, iconStr, nameStr)

	leftW := lipgloss.Width(leftPart)
	rightW := lipgloss.Width(rightPart)
	gap := ""
	if remaining := w - leftW - rightW; remaining > 0 {
		gap = strings.Repeat(" ", remaining)
	}

	row := leftPart + gap + rightPart

	if selected {
		return selectedDeviceRowStyle.Width(rowWidth).Render(row)
	}
	return deviceRowStyle.Width(rowWidth).Render(row)
}

func renderVolumeBar(volume float32, muted bool, width int, selected bool) string {
	if muted {
		var bar string
		if selected {
			bar = selectedVolumeEmptyStyle.Render(strings.Repeat("-", width))
		} else {
			bar = volumeEmptyStyle.Render(strings.Repeat("-", width))
		}
		return bar
	}

	filled := int(math.Round(float64(volume) * float64(width)))
	empty := width - filled

	var filledStr, emptyStr string
	if selected {
		filledStr = selectedVolumeFilledStyle.Render(strings.Repeat("|", filled))
		emptyStr = selectedVolumeEmptyStyle.Render(strings.Repeat("-", empty))
	} else {
		filledStr = volumeFilledStyle.Render(strings.Repeat("|", filled))
		emptyStr = volumeEmptyStyle.Render(strings.Repeat("-", empty))
	}

	return filledStr + emptyStr
}

func (m Model) renderStatus() string {
	if m.statusText == "" {
		return ""
	}
	if m.statusIsError {
		return statusErrorStyle.Render(m.statusText)
	}
	return statusStyle.Render(m.statusText)
}

func (m Model) renderHelp() string {
	bindings := []struct {
		key  string
		desc string
	}{
		{"j/k", "navigate"},
		{"tab/1/2", "switch tab"},
		{"enter", "set default"},
		{"h/l", "volume"},
		{"+/-", "fine vol"},
		{"m", "mute"},
		{"q", "quit"},
	}

	var parts []string
	for _, b := range bindings {
		parts = append(parts,
			helpKeyStyle.Render(b.key)+" "+helpDescStyle.Render(b.desc),
		)
	}

	return helpStyle.Render(strings.Join(parts, "    "))
}

func (m Model) listWidth() int {
	maxWidth := 72
	if m.width > 0 && m.width-4 < maxWidth {
		return m.width - 4
	}
	return maxWidth
}

func transportIcon(t audio.TransportType) string {
	switch t {
	case audio.TransportBuiltIn:
		return " SYS"
	case audio.TransportUSB:
		return " USB"
	case audio.TransportBluetooth:
		return "  BT"
	case audio.TransportHDMI:
		return "HDMI"
	case audio.TransportDisplayPort:
		return "  DP"
	case audio.TransportAirPlay:
		return " AIR"
	case audio.TransportThunderbolt:
		return "  TB"
	case audio.TransportVirtual:
		return " VRT"
	default:
		return "  ?"
	}
}
