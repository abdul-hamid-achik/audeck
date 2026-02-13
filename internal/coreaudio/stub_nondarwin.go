//go:build !darwin

package coreaudio

import "errors"

var errUnsupported = errors.New("coreaudio: unsupported platform (requires darwin)")

// ErrNoDevice is returned when the default device is not set (ID == 0).
var ErrNoDevice = errors.New("coreaudio: no device available")

// AudioObjectID is the unique identifier for an audio object.
type AudioObjectID uint32

// DeviceID is an audio device identifier.
type DeviceID = AudioObjectID

// StreamID is an audio stream identifier.
type StreamID = AudioObjectID

// PropertySelector identifies which property to query/set.
type PropertySelector uint32

// PropertyScope identifies a property scope.
type PropertyScope uint32

// PropertyElement identifies a property element.
type PropertyElement uint32

// PropertyAddress identifies a specific property on an audio object.
type PropertyAddress struct {
	Selector PropertySelector
	Scope    PropertyScope
	Element  PropertyElement
}

// TransportType identifies how a device is connected.
type TransportType uint32

// OSStatus is the CoreAudio/macOS error code type.
type OSStatus int32

// Error represents a CoreAudio error with its OSStatus code.
type Error struct {
	Code    OSStatus
	Context string
}

func (e *Error) Error() string {
	return "coreaudio: " + e.Context + ": unsupported platform"
}

// PropertyEvent is emitted when a watched CoreAudio property changes.
type PropertyEvent struct {
	ObjectID AudioObjectID
	Address  PropertyAddress
}

// Listener receives CoreAudio property change notifications (stub).
type Listener struct {
	ch chan PropertyEvent
}

func NewListener(bufSize int) *Listener {
	return &Listener{ch: make(chan PropertyEvent, bufSize)}
}

func (l *Listener) Events() <-chan PropertyEvent { return l.ch }
func (l *Listener) Watch(AudioObjectID, PropertyAddress) error {
	return errUnsupported
}
func (l *Listener) Unwatch(AudioObjectID, PropertyAddress) error {
	return errUnsupported
}
func (l *Listener) Close() error {
	close(l.ch)
	return nil
}

// Property scope constants.
const (
	ScopeGlobal PropertyScope = 0x676C6F62
	ScopeInput  PropertyScope = 0x696E7074
	ScopeOutput PropertyScope = 0x6F757470
)

const ElementMain PropertyElement = 0

const SystemObjectID AudioObjectID = 1

// Hardware property selectors.
const (
	HardwarePropertyDevices             PropertySelector = 0x64657623
	HardwarePropertyDefaultOutputDevice PropertySelector = 0x644F7574
	HardwarePropertyDefaultInputDevice  PropertySelector = 0x64496E20
)

// Device property selectors.
const (
	DevicePropertyName          PropertySelector = 0x6C6E616D
	DevicePropertyDeviceUID     PropertySelector = 0x75696420
	DevicePropertyTransportType PropertySelector = 0x7472616E
	DevicePropertyIsAlive       PropertySelector = 0x6C69766E
	DevicePropertyVolumeScalar  PropertySelector = 0x766F6C6D
	DevicePropertyMute          PropertySelector = 0x6D757465
	DevicePropertyStreams       PropertySelector = 0x73746D23
)

// AudioHardwareService property selectors.
const (
	VirtualMainVolume PropertySelector = 0x766D7663
)

// Transport type constants.
const (
	TransportBuiltIn     TransportType = 0x626C746E
	TransportUSB         TransportType = 0x75736220
	TransportBluetooth   TransportType = 0x626C7565
	TransportHDMI        TransportType = 0x68646D69
	TransportDisplayPort TransportType = 0x64707274
	TransportAirPlay     TransportType = 0x61697270
	TransportThunderbolt TransportType = 0x7468756E
	TransportVirtual     TransportType = 0x76697274
	TransportUnknown     TransportType = 0
)

// Common CoreAudio error codes.
const (
	NoError                   OSStatus = 0
	NotRunningError           OSStatus = 0x73746F70
	UnspecifiedError          OSStatus = 0x77686174
	UnknownPropertyError      OSStatus = 0x77686F3F
	BadPropertySizeError      OSStatus = 0x2173697A
	IllegalOperationError     OSStatus = 0x6E6F7065
	BadObjectError            OSStatus = 0x216F626A
	BadDeviceError            OSStatus = 0x21646576
	BadStreamError            OSStatus = 0x21737472
	UnsupportedOperationError OSStatus = 0x756E6F70
	DevicePermissionsError    OSStatus = 0x21686F67
)

// Device query stubs.
func GetDeviceIDs() ([]DeviceID, error)                     { return nil, errUnsupported }
func GetDeviceName(DeviceID) (string, error)                { return "", errUnsupported }
func GetDeviceUID(DeviceID) (string, error)                 { return "", errUnsupported }
func GetDeviceTransportType(DeviceID) (TransportType, error) { return TransportUnknown, errUnsupported }
func GetDeviceIsAlive(DeviceID) (bool, error)               { return false, errUnsupported }
func IsOutputDevice(DeviceID) (bool, error)                 { return false, errUnsupported }
func IsInputDevice(DeviceID) (bool, error)                  { return false, errUnsupported }
func HasProperty(AudioObjectID, PropertyAddress) bool       { return false }

// Default device stubs.
func GetDefaultOutputDevice() (DeviceID, error)  { return 0, errUnsupported }
func SetDefaultOutputDevice(DeviceID) error      { return errUnsupported }
func GetDefaultInputDevice() (DeviceID, error)   { return 0, errUnsupported }
func SetDefaultInputDevice(DeviceID) error       { return errUnsupported }

// Volume stubs.
func HasVolumeControl(DeviceID, PropertyScope) bool                      { return false }
func GetDeviceVolume(DeviceID) (float32, error)                          { return 0, errUnsupported }
func SetDeviceVolume(DeviceID, float32) error                            { return errUnsupported }
func GetDeviceVolumeForScope(DeviceID, PropertyScope) (float32, error)   { return 0, errUnsupported }
func SetDeviceVolumeForScope(DeviceID, PropertyScope, float32) error     { return errUnsupported }
func VolumeWatchAddresses(PropertyScope) []PropertyAddress               { return nil }

// Mute stubs.
func HasMuteControl(DeviceID, PropertyScope) bool                    { return false }
func GetDeviceMute(DeviceID) (bool, error)                           { return false, errUnsupported }
func SetDeviceMute(DeviceID, bool) error                             { return errUnsupported }
func GetDeviceMuteForScope(DeviceID, PropertyScope) (bool, error)    { return false, errUnsupported }
func SetDeviceMuteForScope(DeviceID, PropertyScope, bool) error      { return errUnsupported }
func MuteWatchAddresses(PropertyScope) []PropertyAddress             { return nil }

// Property stubs.
func GetPropertyDataSize(AudioObjectID, PropertyAddress) (uint32, error) { return 0, errUnsupported }
func GetPropertyData(AudioObjectID, PropertyAddress) ([]byte, error)     { return nil, errUnsupported }
func SetPropertyData(AudioObjectID, PropertyAddress, []byte) error       { return errUnsupported }
func IsPropertySettable(AudioObjectID, PropertyAddress) (bool, error)    { return false, errUnsupported }
