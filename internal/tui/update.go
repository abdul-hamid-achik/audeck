package tui

import (
	"errors"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/abdulachik/audeck/internal/audio"
)

// Update handles incoming messages and returns an updated model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, refreshDevicesCmd(m.manager)

	case devicesRefreshedMsg:
		m.outputDevices = msg.outputDevices
		m.inputDevices = msg.inputDevices
		// Clamp cursor to valid range after device list changes.
		devices := m.activeDevices()
		if m.cursor >= len(devices) {
			m.cursor = max(0, len(devices)-1)
		}
		m.adjustScroll()
		return m, nil

	case audioEventMsg:
		m.handleAudioEvent(msg.event)
		return m, listenEventsCmd(m.events)

	case errMsg:
		if msg.err != nil {
			m.statusText = msg.err.Error()
			m.statusIsError = true
			return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
				return clearStatusMsg{}
			})
		}
		return m, nil

	case statusMsg:
		m.statusText = msg.text
		m.statusIsError = false
		return m, tea.Tick(3*time.Second, func(time.Time) tea.Msg {
			return clearStatusMsg{}
		})

	case clearStatusMsg:
		m.statusText = ""
		return m, nil

	case tea.MouseMsg:
		return m.handleMouse(msg)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// handleAudioEvent processes events from the Manager subscription and
// updates the TUI model accordingly.
func (m *Model) handleAudioEvent(evt audio.Event) {
	switch e := evt.(type) {
	case audio.DeviceListChanged:
		var outputs, inputs []device
		for _, d := range e.Devices {
			td := mapDevice(d)
			if d.IsOutput {
				outputs = append(outputs, td)
			}
			if d.IsInput {
				inputs = append(inputs, td)
			}
		}
		m.outputDevices = outputs
		m.inputDevices = inputs
		devices := m.activeDevices()
		if m.cursor >= len(devices) {
			m.cursor = max(0, len(devices)-1)
		}
		m.adjustScroll()

	case audio.DefaultDeviceChanged:
		for i := range m.outputDevices {
			m.outputDevices[i].IsDefault = m.outputDevices[i].ID == e.Device.ID
		}

	case audio.DefaultInputDeviceChanged:
		for i := range m.inputDevices {
			m.inputDevices[i].IsDefault = m.inputDevices[i].ID == e.Device.ID
		}

	case audio.VolumeChanged:
		for i := range m.outputDevices {
			if m.outputDevices[i].ID == e.DeviceID {
				m.outputDevices[i].Volume = e.Volume
				break
			}
		}
		for i := range m.inputDevices {
			if m.inputDevices[i].ID == e.DeviceID {
				m.inputDevices[i].Volume = e.Volume
				break
			}
		}

	case audio.MuteChanged:
		for i := range m.outputDevices {
			if m.outputDevices[i].ID == e.DeviceID {
				m.outputDevices[i].Muted = e.Muted
				break
			}
		}
		for i := range m.inputDevices {
			if m.inputDevices[i].ID == e.DeviceID {
				m.inputDevices[i].Muted = e.Muted
				break
			}
		}
	}
}

// selectedDevice returns the currently selected device, or nil if none.
func (m Model) selectedDevice() *device {
	devices := m.activeDevices()
	if m.cursor >= 0 && m.cursor < len(devices) {
		return &devices[m.cursor]
	}
	return nil
}

// visibleDeviceCount returns how many device rows fit in the viewport.
func (m Model) visibleDeviceCount() int {
	// Reserve lines for: title(2) + tabs(2) + border(2) + help(2) + status(1) + padding(2)
	available := m.height - 11
	if available < 3 {
		return 3
	}
	return available
}

// adjustScroll ensures the cursor is within the visible scroll window.
func (m *Model) adjustScroll() {
	visible := m.visibleDeviceCount()
	devices := m.activeDevices()
	if m.cursor < m.scrollOffset {
		m.scrollOffset = m.cursor
	}
	if m.cursor >= m.scrollOffset+visible {
		m.scrollOffset = m.cursor - visible + 1
	}
	// Clamp scrollOffset so we don't scroll past the end.
	if maxOffset := len(devices) - visible; maxOffset > 0 && m.scrollOffset > maxOffset {
		m.scrollOffset = maxOffset
	}
	if m.scrollOffset < 0 {
		m.scrollOffset = 0
	}
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		if m.cancelSub != nil {
			m.cancelSub()
		}
		return m, tea.Quit

	// Tab switching
	case "tab":
		if m.activeTab == tabOutput {
			m.activeTab = tabInput
		} else {
			m.activeTab = tabOutput
		}
		m.cursor = 0
		m.scrollOffset = 0
	case "1":
		if m.activeTab != tabOutput {
			m.activeTab = tabOutput
			m.cursor = 0
			m.scrollOffset = 0
		}
	case "2":
		if m.activeTab != tabInput {
			m.activeTab = tabInput
			m.cursor = 0
			m.scrollOffset = 0
		}

	// Navigation
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.adjustScroll()
		}
	case "down", "j":
		devices := m.activeDevices()
		if m.cursor < len(devices)-1 {
			m.cursor++
			m.adjustScroll()
		}
	case "home", "g":
		m.cursor = 0
		m.adjustScroll()
	case "end", "G":
		devices := m.activeDevices()
		if len(devices) > 0 {
			m.cursor = len(devices) - 1
			m.adjustScroll()
		}

	// Set default device
	case "enter":
		if d := m.selectedDevice(); d != nil {
			mgr := m.manager
			deviceID := d.ID
			deviceName := d.Name
			isInputTab := m.activeTab == tabInput
			return m, func() tea.Msg {
				if mgr == nil {
					return errMsg{err: errors.New("audio manager unavailable")}
				}
				var err error
				if isInputTab {
					err = mgr.SetDefaultInputDevice(deviceID)
				} else {
					err = mgr.SetDefaultDevice(deviceID)
				}
				if err != nil {
					return errMsg{err: err}
				}
				return statusMsg{text: fmt.Sprintf("Default set to %s", deviceName)}
			}
		}

	// Volume up (5% step)
	case "right", "l":
		if d := m.selectedDevice(); d != nil && d.HasVolume {
			return m, m.adjustVolumeCmd(d.ID, 0.05)
		}

	// Volume down (5% step)
	case "left", "h":
		if d := m.selectedDevice(); d != nil && d.HasVolume {
			return m, m.adjustVolumeCmd(d.ID, -0.05)
		}

	// Fine volume up (1% step)
	case "+", "=":
		if d := m.selectedDevice(); d != nil && d.HasVolume {
			return m, m.adjustVolumeCmd(d.ID, 0.01)
		}

	// Fine volume down (1% step)
	case "-", "_":
		if d := m.selectedDevice(); d != nil && d.HasVolume {
			return m, m.adjustVolumeCmd(d.ID, -0.01)
		}

	// Mute toggle
	case "m":
		if d := m.selectedDevice(); d != nil && d.HasMute {
			mgr := m.manager
			deviceID := d.ID
			return m, func() tea.Msg {
				if mgr == nil {
					return errMsg{err: errors.New("audio manager unavailable")}
				}
				if err := mgr.ToggleMute(deviceID); err != nil {
					return errMsg{err: err}
				}
				return nil
			}
		}
	}

	return m, nil
}

// handleMouse processes mouse events for scroll wheel navigation.
func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if m.cursor > 0 {
			m.cursor--
			m.adjustScroll()
		}
	case tea.MouseButtonWheelDown:
		devices := m.activeDevices()
		if m.cursor < len(devices)-1 {
			m.cursor++
			m.adjustScroll()
		}
	}
	return m, nil
}

// adjustVolumeCmd returns a Cmd that adjusts the volume of the given device.
func (m Model) adjustVolumeCmd(deviceID uint32, delta float32) tea.Cmd {
	mgr := m.manager
	return func() tea.Msg {
		if mgr == nil {
			return errMsg{err: errors.New("audio manager unavailable")}
		}
		if err := mgr.AdjustVolume(deviceID, delta); err != nil {
			return errMsg{err: err}
		}
		return nil
	}
}
