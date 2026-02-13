package audio

import "fmt"

// Device represents an audio device with its current state.
type Device struct {
	ID            uint32
	UID           string
	Name          string
	TransportType TransportType
	Volume        float32 // 0.0 to 1.0
	Muted         bool
	IsDefault     bool
	HasVolume     bool // whether volume control is available
	HasMute       bool // whether mute control is available
	IsInput       bool // whether the device has input streams
	IsOutput      bool // whether the device has output streams
}

// TransportType describes how a device is connected to the system.
type TransportType int

const (
	TransportUnknown     TransportType = iota
	TransportBuiltIn                   // internal speakers / mic
	TransportUSB                       // USB audio interface
	TransportBluetooth                 // Bluetooth headphones / speakers
	TransportHDMI                      // HDMI output
	TransportDisplayPort               // DisplayPort output
	TransportAirPlay                   // AirPlay receiver
	TransportThunderbolt               // Thunderbolt audio device
	TransportVirtual                   // virtual / aggregate device
)

// String returns a human-readable label for the transport type.
func (t TransportType) String() string {
	switch t {
	case TransportBuiltIn:
		return "Built-in"
	case TransportUSB:
		return "USB"
	case TransportBluetooth:
		return "Bluetooth"
	case TransportHDMI:
		return "HDMI"
	case TransportDisplayPort:
		return "DisplayPort"
	case TransportAirPlay:
		return "AirPlay"
	case TransportThunderbolt:
		return "Thunderbolt"
	case TransportVirtual:
		return "Virtual"
	default:
		return "Unknown"
	}
}

// VolumePercent returns the volume as an integer percentage (0-100).
func (d Device) VolumePercent() int {
	return int(d.Volume * 100)
}

// DisplayName returns the device name with transport type annotation.
func (d Device) DisplayName() string {
	if d.TransportType == TransportUnknown || d.TransportType == TransportBuiltIn {
		return d.Name
	}
	return fmt.Sprintf("%s (%s)", d.Name, d.TransportType)
}
