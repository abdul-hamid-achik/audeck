package audio

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/abdulachik/audeck/internal/coreaudio"
)

// Manager wraps CoreAudio interactions and provides a high-level
// interface for the TUI layer to query and control audio devices.
//
// It owns the lifecycle of CoreAudio listeners and translates low-level
// property change events into typed audio.Event values. The TUI layer
// should only interact with audio through this type.
type Manager struct {
	mu       sync.RWMutex
	devices  []Device
	events   chan Event
	listener *coreaudio.Listener

	// cancel stops the internal event processing goroutine.
	cancel context.CancelFunc
}

// NewManager initializes the audio manager.
// It queries CoreAudio for the initial device list and starts
// listening for property changes.
func NewManager() (*Manager, error) {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		events: make(chan Event, 64),
		cancel: cancel,
	}

	// Load initial device list.
	if err := m.refreshDevices(); err != nil {
		cancel()
		return nil, err
	}

	// Start background event processing.
	go m.processEvents(ctx)

	return m, nil
}

// Close shuts down the manager and releases CoreAudio resources.
func (m *Manager) Close() {
	if m.cancel != nil {
		m.cancel()
	}
	m.mu.RLock()
	l := m.listener
	m.mu.RUnlock()
	if l != nil {
		l.Close()
	}
}

// Devices returns a snapshot of the current output device list.
func (m *Manager) Devices() []Device {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]Device, len(m.devices))
	copy(out, m.devices)
	return out
}

// DefaultDevice returns the current default output device.
func (m *Manager) DefaultDevice() (Device, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, d := range m.devices {
		if d.IsDefault {
			return d, true
		}
	}
	return Device{}, false
}

// DeviceByID looks up a device by its ID.
func (m *Manager) DeviceByID(deviceID uint32) (Device, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, d := range m.devices {
		if d.ID == deviceID {
			return d, true
		}
	}
	return Device{}, false
}

// DeviceByUID looks up a device by its UID string.
func (m *Manager) DeviceByUID(uid string) (Device, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, d := range m.devices {
		if d.UID == uid {
			return d, true
		}
	}
	return Device{}, false
}

// SetDefaultDevice changes the system default output device.
func (m *Manager) SetDefaultDevice(deviceID uint32) error {
	if err := coreaudio.SetDefaultOutputDevice(coreaudio.DeviceID(deviceID)); err != nil {
		return err
	}
	// Mark the new default in local state.
	m.mu.Lock()
	for i := range m.devices {
		m.devices[i].IsDefault = m.devices[i].ID == deviceID
	}
	m.mu.Unlock()
	return nil
}

// SetVolume sets the volume for a device (0.0 to 1.0).
// Values are clamped to the valid range. Devices without volume control
// are silently skipped.
func (m *Manager) SetVolume(deviceID uint32, volume float32) error {
	volume = clampVolume(volume)

	if err := coreaudio.SetDeviceVolume(coreaudio.DeviceID(deviceID), volume); err != nil {
		return err
	}

	// Optimistically update local state for responsive UI.
	m.mu.Lock()
	for i := range m.devices {
		if m.devices[i].ID == deviceID {
			m.devices[i].Volume = volume
			break
		}
	}
	m.mu.Unlock()

	return nil
}

// AdjustVolume changes the volume by a relative delta (e.g., +0.05 or -0.05).
func (m *Manager) AdjustVolume(deviceID uint32, delta float32) error {
	m.mu.Lock()
	var current float32
	var found bool
	for _, d := range m.devices {
		if d.ID == deviceID {
			current = d.Volume
			found = true
			break
		}
	}
	if !found {
		m.mu.Unlock()
		return coreaudio.ErrNoDevice
	}
	newVol := clampVolume(current + delta)
	m.mu.Unlock()

	if err := coreaudio.SetDeviceVolume(coreaudio.DeviceID(deviceID), newVol); err != nil {
		return err
	}

	m.mu.Lock()
	for i := range m.devices {
		if m.devices[i].ID == deviceID {
			m.devices[i].Volume = newVol
			break
		}
	}
	m.mu.Unlock()
	return nil
}

// SetMute sets the mute state for a device. Devices without mute control
// are silently skipped.
func (m *Manager) SetMute(deviceID uint32, muted bool) error {
	if err := coreaudio.SetDeviceMute(coreaudio.DeviceID(deviceID), muted); err != nil {
		return err
	}

	// Optimistically update local state.
	m.mu.Lock()
	for i := range m.devices {
		if m.devices[i].ID == deviceID {
			m.devices[i].Muted = muted
			break
		}
	}
	m.mu.Unlock()

	return nil
}

// ToggleMute toggles the mute state for a device.
func (m *Manager) ToggleMute(deviceID uint32) error {
	m.mu.Lock()
	var muted bool
	var found bool
	for _, d := range m.devices {
		if d.ID == deviceID {
			muted = d.Muted
			found = true
			break
		}
	}
	if !found {
		m.mu.Unlock()
		return coreaudio.ErrNoDevice
	}
	newMuted := !muted
	m.mu.Unlock()

	if err := coreaudio.SetDeviceMute(coreaudio.DeviceID(deviceID), newMuted); err != nil {
		return err
	}

	m.mu.Lock()
	for i := range m.devices {
		if m.devices[i].ID == deviceID {
			m.devices[i].Muted = newMuted
			break
		}
	}
	m.mu.Unlock()
	return nil
}

// Subscribe returns a channel that receives audio events.
// Events include device list changes, volume changes, mute changes,
// and default device changes. The channel is closed when the context
// is cancelled or the Manager is closed.
func (m *Manager) Subscribe(ctx context.Context) <-chan Event {
	out := make(chan Event, 32)
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-m.events:
				if !ok {
					return
				}
				select {
				case out <- evt:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}

// ApplyPreferredDevice finds a device by UID and sets it as the default
// for the given scope. Silently returns nil if the device is not found.
func (m *Manager) ApplyPreferredDevice(uid string, scope DeviceScope) error {
	if uid == "" {
		return nil
	}
	d, ok := m.DeviceByUID(uid)
	if !ok {
		return nil
	}
	switch scope {
	case ScopeOutput:
		return m.SetDefaultDevice(d.ID)
	case ScopeInput:
		return m.SetDefaultInputDevice(d.ID)
	}
	return nil
}

// DefaultInputDevice returns the current default input device.
func (m *Manager) DefaultInputDevice() (Device, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, d := range m.devices {
		if d.IsInput && d.IsDefault {
			return d, true
		}
	}
	return Device{}, false
}

// SetDefaultInputDevice changes the system default input device.
func (m *Manager) SetDefaultInputDevice(deviceID uint32) error {
	if err := coreaudio.SetDefaultInputDevice(coreaudio.DeviceID(deviceID)); err != nil {
		return err
	}
	// Mark the new default input in local state.
	m.mu.Lock()
	for i := range m.devices {
		if m.devices[i].IsInput {
			m.devices[i].IsDefault = m.devices[i].ID == deviceID
		}
	}
	m.mu.Unlock()
	return nil
}

// OutputDevices returns a snapshot of only output devices.
func (m *Manager) OutputDevices() []Device {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Device
	for _, d := range m.devices {
		if d.IsOutput {
			out = append(out, d)
		}
	}
	return out
}

// InputDevices returns a snapshot of only input devices.
func (m *Manager) InputDevices() []Device {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []Device
	for _, d := range m.devices {
		if d.IsInput {
			out = append(out, d)
		}
	}
	return out
}

// refreshDevices queries CoreAudio for the current device list and updates
// the internal state. Builds both input and output devices.
func (m *Manager) refreshDevices() error {
	ids, err := coreaudio.GetDeviceIDs()
	if err != nil {
		return err
	}

	defaultOutputID, err := coreaudio.GetDefaultOutputDevice()
	if err != nil && !errors.Is(err, coreaudio.ErrNoDevice) {
		return fmt.Errorf("get default output device: %w", err)
	}

	defaultInputID, err := coreaudio.GetDefaultInputDevice()
	if err != nil && !errors.Is(err, coreaudio.ErrNoDevice) {
		return fmt.Errorf("get default input device: %w", err)
	}

	var devices []Device
	for _, id := range ids {
		alive, err := coreaudio.GetDeviceIsAlive(id)
		if err != nil {
			return fmt.Errorf("check device alive %d: %w", id, err)
		}
		if !alive {
			continue
		}

		isOutput, err := coreaudio.IsOutputDevice(id)
		if err != nil {
			return fmt.Errorf("check output device %d: %w", id, err)
		}
		isInput, err := coreaudio.IsInputDevice(id)
		if err != nil {
			return fmt.Errorf("check input device %d: %w", id, err)
		}
		if !isOutput && !isInput {
			continue
		}

		name, err := coreaudio.GetDeviceName(id)
		if err != nil {
			return fmt.Errorf("get device name %d: %w", id, err)
		}
		uid, err := coreaudio.GetDeviceUID(id)
		if err != nil {
			return fmt.Errorf("get device uid %d: %w", id, err)
		}
		cTransport, err := coreaudio.GetDeviceTransportType(id)
		if err != nil {
			return fmt.Errorf("get device transport %d: %w", id, err)
		}

		// Determine scope for volume/mute property checks.
		scope := coreaudio.ScopeOutput
		if !isOutput && isInput {
			scope = coreaudio.ScopeInput
		}

		hasVolume := coreaudio.HasVolumeControl(id, scope)
		hasMute := coreaudio.HasMuteControl(id, scope)

		var vol float32
		if hasVolume {
			vol, err = coreaudio.GetDeviceVolumeForScope(id, scope)
			if err != nil {
				return fmt.Errorf("get device volume %d: %w", id, err)
			}
		}
		var muted bool
		if hasMute {
			muted, err = coreaudio.GetDeviceMuteForScope(id, scope)
			if err != nil {
				return fmt.Errorf("get device mute %d: %w", id, err)
			}
		}

		// Determine default status based on device type.
		isDefault := false
		if isOutput {
			isDefault = id == defaultOutputID
		} else if isInput {
			isDefault = id == defaultInputID
		}

		devices = append(devices, Device{
			ID:            uint32(id),
			UID:           uid,
			Name:          name,
			TransportType: mapTransportType(cTransport),
			Volume:        vol,
			Muted:         muted,
			IsDefault:     isDefault,
			HasVolume:     hasVolume,
			HasMute:       hasMute,
			IsInput:       isInput,
			IsOutput:      isOutput,
		})
	}

	m.mu.Lock()
	m.devices = devices
	m.mu.Unlock()

	return nil
}

// watchDeviceProperties registers CoreAudio property listeners for a device's
// volume and mute properties on the appropriate scope. Watches all possible
// property addresses (VirtualMainVolume, per-channel VolumeScalar, etc.)
// so we catch events regardless of which property the device exposes.
func (m *Manager) watchDeviceProperties(d Device) {
	if m.listener == nil {
		return
	}
	id := coreaudio.DeviceID(d.ID)
	scope := coreaudio.ScopeOutput
	if !d.IsOutput && d.IsInput {
		scope = coreaudio.ScopeInput
	}
	if d.HasVolume {
		for _, addr := range coreaudio.VolumeWatchAddresses(scope) {
			m.listener.Watch(id, addr) // ignore errors for non-existent properties
		}
	}
	if d.HasMute {
		for _, addr := range coreaudio.MuteWatchAddresses(scope) {
			m.listener.Watch(id, addr)
		}
	}
}

// unwatchDeviceProperties removes CoreAudio property listeners for a device.
func (m *Manager) unwatchDeviceProperties(d Device) {
	if m.listener == nil {
		return
	}
	id := coreaudio.DeviceID(d.ID)
	scope := coreaudio.ScopeOutput
	if !d.IsOutput && d.IsInput {
		scope = coreaudio.ScopeInput
	}
	if d.HasVolume {
		for _, addr := range coreaudio.VolumeWatchAddresses(scope) {
			m.listener.Unwatch(id, addr)
		}
	}
	if d.HasMute {
		for _, addr := range coreaudio.MuteWatchAddresses(scope) {
			m.listener.Unwatch(id, addr)
		}
	}
}

// processEvents reads from the coreaudio listener and translates
// property change events into typed audio.Event values.
func (m *Manager) processEvents(ctx context.Context) {
	defer close(m.events)

	listener := coreaudio.NewListener(64)
	defer listener.Close()

	m.mu.Lock()
	m.listener = listener
	m.mu.Unlock()

	// Watch system-level events.
	listener.Watch(coreaudio.SystemObjectID, coreaudio.PropertyAddress{
		Selector: coreaudio.HardwarePropertyDevices,
		Scope:    coreaudio.ScopeGlobal,
		Element:  coreaudio.ElementMain,
	})
	listener.Watch(coreaudio.SystemObjectID, coreaudio.PropertyAddress{
		Selector: coreaudio.HardwarePropertyDefaultOutputDevice,
		Scope:    coreaudio.ScopeGlobal,
		Element:  coreaudio.ElementMain,
	})
	listener.Watch(coreaudio.SystemObjectID, coreaudio.PropertyAddress{
		Selector: coreaudio.HardwarePropertyDefaultInputDevice,
		Scope:    coreaudio.ScopeGlobal,
		Element:  coreaudio.ElementMain,
	})

	// Watch per-device volume and mute properties.
	m.mu.RLock()
	for _, d := range m.devices {
		m.watchDeviceProperties(d)
	}
	m.mu.RUnlock()

	for {
		select {
		case <-ctx.Done():
			return
		case evt, ok := <-listener.Events():
			if !ok {
				return
			}
			m.handlePropertyEvent(evt)
		}
	}
}

// handlePropertyEvent translates a raw CoreAudio property change into
// typed audio.Event values and updates internal state.
func (m *Manager) handlePropertyEvent(evt coreaudio.PropertyEvent) {
	switch evt.Address.Selector {
	case coreaudio.HardwarePropertyDevices:
		// Snapshot old device set.
		m.mu.RLock()
		oldDevices := make([]Device, len(m.devices))
		copy(oldDevices, m.devices)
		m.mu.RUnlock()

		m.refreshDevices()

		// Snapshot new device set.
		m.mu.RLock()
		newDevices := make([]Device, len(m.devices))
		copy(newDevices, m.devices)
		m.mu.RUnlock()

		// Build sets for diff.
		oldSet := make(map[uint32]Device, len(oldDevices))
		for _, d := range oldDevices {
			oldSet[d.ID] = d
		}
		newSet := make(map[uint32]Device, len(newDevices))
		for _, d := range newDevices {
			newSet[d.ID] = d
		}

		// Unwatch removed devices.
		for id, d := range oldSet {
			if _, ok := newSet[id]; !ok {
				m.unwatchDeviceProperties(d)
			}
		}
		// Watch newly added devices.
		for id, d := range newSet {
			if _, ok := oldSet[id]; !ok {
				m.watchDeviceProperties(d)
			}
		}

		m.emit(DeviceListChanged{Devices: m.Devices()})

	case coreaudio.HardwarePropertyDefaultOutputDevice:
		m.refreshDevices()
		if d, ok := m.DefaultDevice(); ok {
			m.emit(DefaultDeviceChanged{Device: d})
		}

	case coreaudio.HardwarePropertyDefaultInputDevice:
		m.refreshDevices()
		if d, ok := m.DefaultInputDevice(); ok {
			m.emit(DefaultInputDeviceChanged{Device: d})
		}

	case coreaudio.VirtualMainVolume, coreaudio.DevicePropertyVolumeScalar:
		vol, err := coreaudio.GetDeviceVolumeForScope(coreaudio.DeviceID(evt.ObjectID), evt.Address.Scope)
		if err == nil {
			m.mu.Lock()
			for i := range m.devices {
				if m.devices[i].ID == uint32(evt.ObjectID) {
					m.devices[i].Volume = vol
					break
				}
			}
			m.mu.Unlock()
			m.emit(VolumeChanged{DeviceID: uint32(evt.ObjectID), Volume: vol})
		}
	case coreaudio.DevicePropertyMute:
		muted, err := coreaudio.GetDeviceMuteForScope(coreaudio.DeviceID(evt.ObjectID), evt.Address.Scope)
		if err == nil {
			m.mu.Lock()
			for i := range m.devices {
				if m.devices[i].ID == uint32(evt.ObjectID) {
					m.devices[i].Muted = muted
					break
				}
			}
			m.mu.Unlock()
			m.emit(MuteChanged{DeviceID: uint32(evt.ObjectID), Muted: muted})
		}
	}
}

// emit sends an event to subscribers. Non-blocking: drops if buffer is full.
func (m *Manager) emit(evt Event) {
	select {
	case m.events <- evt:
	default:
	}
}

// mapTransportType converts a coreaudio FourCC transport type to the
// audio package's TransportType enum.
func mapTransportType(ct coreaudio.TransportType) TransportType {
	switch ct {
	case coreaudio.TransportBuiltIn:
		return TransportBuiltIn
	case coreaudio.TransportUSB:
		return TransportUSB
	case coreaudio.TransportBluetooth:
		return TransportBluetooth
	case coreaudio.TransportHDMI:
		return TransportHDMI
	case coreaudio.TransportDisplayPort:
		return TransportDisplayPort
	case coreaudio.TransportAirPlay:
		return TransportAirPlay
	case coreaudio.TransportThunderbolt:
		return TransportThunderbolt
	case coreaudio.TransportVirtual:
		return TransportVirtual
	default:
		return TransportUnknown
	}
}

func clampVolume(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
