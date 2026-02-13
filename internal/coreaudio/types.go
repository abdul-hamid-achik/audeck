//go:build darwin

package coreaudio

// AudioObjectID is the unique identifier for a CoreAudio object.
// Maps to C type: AudioObjectID (UInt32).
type AudioObjectID uint32

// DeviceID is an audio device identifier. Alias for clarity.
type DeviceID = AudioObjectID

// StreamID is an audio stream identifier.
type StreamID = AudioObjectID

// PropertySelector identifies which property to query/set.
type PropertySelector uint32

// PropertyScope identifies the scope of a property (input/output/global).
type PropertyScope uint32

// PropertyElement identifies the element of a property.
type PropertyElement uint32

// PropertyAddress identifies a specific property on an audio object.
type PropertyAddress struct {
	Selector PropertySelector
	Scope    PropertyScope
	Element  PropertyElement
}

// TransportType identifies how a device is connected.
type TransportType uint32
