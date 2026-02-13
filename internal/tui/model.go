package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/abdulachik/audeck/internal/audio"
)

// device holds the display state for a single audio device.
type device struct {
	ID            uint32
	Name          string
	TransportType audio.TransportType
	Volume        float32 // 0.0 to 1.0
	Muted         bool
	IsDefault     bool
	HasVolume     bool
	HasMute       bool
	IsInput       bool
	IsOutput      bool
}

const (
	tabOutput = 0
	tabInput  = 1
)

// Model is the top-level Bubble Tea model for audeck.
type Model struct {
	manager *audio.Manager

	outputDevices []device
	inputDevices  []device
	activeTab     int // 0=output, 1=input
	cursor        int
	scrollOffset  int

	width  int
	height int
	ready  bool

	statusText    string // transient status/error message
	statusIsError bool   // true if error, false if success

	// Event subscription from Manager.
	events    <-chan audio.Event
	cancelSub context.CancelFunc
}

// NewModel creates a Model backed by the given audio Manager.
func NewModel(manager *audio.Manager) Model {
	ctx, cancel := context.WithCancel(context.Background())
	events := manager.Subscribe(ctx)
	return Model{
		manager:   manager,
		cursor:    0,
		activeTab: tabOutput,
		events:    events,
		cancelSub: cancel,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(refreshDevicesCmd(m.manager), listenEventsCmd(m.events))
}

// Message types for the TUI event loop.

type devicesRefreshedMsg struct {
	outputDevices []device
	inputDevices  []device
}

// audioEventMsg wraps an audio.Event from the Manager subscription.
type audioEventMsg struct {
	event audio.Event
}

type errMsg struct {
	err error
}

type statusMsg struct {
	text string
}

type clearStatusMsg struct{}

// activeDevices returns the device list for the active tab.
func (m *Model) activeDevices() []device {
	if m.activeTab == tabInput {
		return m.inputDevices
	}
	return m.outputDevices
}

// setActiveDevices sets the device list for the active tab.
func (m *Model) setActiveDevices(devices []device) {
	if m.activeTab == tabInput {
		m.inputDevices = devices
	} else {
		m.outputDevices = devices
	}
}

// refreshDevicesCmd returns a Cmd that queries the manager for both output
// and input device lists.
func refreshDevicesCmd(mgr *audio.Manager) tea.Cmd {
	return func() tea.Msg {
		if mgr == nil {
			return devicesRefreshedMsg{}
		}
		outputs := mgr.OutputDevices()
		inputs := mgr.InputDevices()
		tuiOutputs := make([]device, len(outputs))
		for i, d := range outputs {
			tuiOutputs[i] = mapDevice(d)
		}
		tuiInputs := make([]device, len(inputs))
		for i, d := range inputs {
			tuiInputs[i] = mapDevice(d)
		}
		return devicesRefreshedMsg{outputDevices: tuiOutputs, inputDevices: tuiInputs}
	}
}

// listenEventsCmd returns a Cmd that blocks on the event channel and
// returns the next audio event as an audioEventMsg.
func listenEventsCmd(events <-chan audio.Event) tea.Cmd {
	return func() tea.Msg {
		evt, ok := <-events
		if !ok {
			return nil
		}
		return audioEventMsg{event: evt}
	}
}

// mapDevice converts an audio.Device to a tui device for display.
func mapDevice(d audio.Device) device {
	return device{
		ID:            d.ID,
		Name:          d.Name,
		TransportType: d.TransportType,
		Volume:        d.Volume,
		Muted:         d.Muted,
		IsDefault:     d.IsDefault,
		HasVolume:     d.HasVolume,
		HasMute:       d.HasMute,
		IsInput:       d.IsInput,
		IsOutput:      d.IsOutput,
	}
}
