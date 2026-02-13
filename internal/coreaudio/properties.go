//go:build darwin

package coreaudio

// Property scopes.
const (
	ScopeGlobal PropertyScope = 0x676C6F62 // 'glob'
	ScopeInput  PropertyScope = 0x696E7074 // 'inpt'
	ScopeOutput PropertyScope = 0x6F757470 // 'outp'
)

// Property element.
// NOTE: kAudioObjectPropertyElementMaster was deprecated in macOS 12 and
// renamed to kAudioObjectPropertyElementMain. Both have value 0.
const (
	ElementMain PropertyElement = 0
)

// System object (singleton, ID = 1).
const SystemObjectID AudioObjectID = 1

// Hardware property selectors (on SystemObjectID).
const (
	HardwarePropertyDevices             PropertySelector = 0x64657623 // 'dev#'
	HardwarePropertyDefaultOutputDevice PropertySelector = 0x644F7574 // 'dOut'
	HardwarePropertyDefaultInputDevice  PropertySelector = 0x64496E20 // 'dIn '
)

// Device property selectors (on a DeviceID).
const (
	DevicePropertyName          PropertySelector = 0x6C6E616D // 'lnam' (kAudioObjectPropertyName)
	DevicePropertyDeviceUID     PropertySelector = 0x75696420 // 'uid '
	DevicePropertyTransportType PropertySelector = 0x7472616E // 'tran'
	DevicePropertyIsAlive       PropertySelector = 0x6C69766E // 'livn'
	DevicePropertyVolumeScalar  PropertySelector = 0x766F6C6D // 'volm'
	DevicePropertyMute          PropertySelector = 0x6D757465 // 'mute'
	DevicePropertyStreams       PropertySelector = 0x73746D23 // 'stm#'
)

// AudioHardwareService property selectors (via AudioToolbox).
const (
	VirtualMainVolume PropertySelector = 0x766D7663 // 'vmvc'
)

// Transport types.
const (
	TransportBuiltIn     TransportType = 0x626C746E // 'bltn'
	TransportUSB         TransportType = 0x75736220 // 'usb '
	TransportBluetooth   TransportType = 0x626C7565 // 'blue'
	TransportHDMI        TransportType = 0x68646D69 // 'hdmi'
	TransportDisplayPort TransportType = 0x64707274 // 'dprt'
	TransportAirPlay     TransportType = 0x61697270 // 'airp'
	TransportThunderbolt TransportType = 0x7468756E // 'thun'
	TransportVirtual     TransportType = 0x76697274 // 'virt'
	TransportUnknown     TransportType = 0
)
