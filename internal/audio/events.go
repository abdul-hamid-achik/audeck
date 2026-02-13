package audio

// Event is the interface for all audio system events.
// These events are sent from the Manager to subscribers (the TUI layer)
// when CoreAudio property changes are detected.
type Event interface {
	audioEvent() // marker method, unexported to seal the interface
}

// DeviceScope distinguishes between output and input device contexts.
type DeviceScope int

const (
	ScopeOutput DeviceScope = iota
	ScopeInput
)

// DeviceListChanged fires when audio devices are added or removed.
type DeviceListChanged struct {
	Devices []Device
}

// DefaultDeviceChanged fires when the default output device changes.
type DefaultDeviceChanged struct {
	Device Device
}

// DefaultInputDeviceChanged fires when the default input device changes.
type DefaultInputDeviceChanged struct {
	Device Device
}

// VolumeChanged fires when a device's volume level changes.
type VolumeChanged struct {
	DeviceID uint32
	Volume   float32
}

// MuteChanged fires when a device's mute state changes.
type MuteChanged struct {
	DeviceID uint32
	Muted    bool
}

func (DeviceListChanged) audioEvent()         {}
func (DefaultDeviceChanged) audioEvent()      {}
func (DefaultInputDeviceChanged) audioEvent() {}
func (VolumeChanged) audioEvent()             {}
func (MuteChanged) audioEvent()               {}
