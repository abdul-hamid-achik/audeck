//go:build darwin

package coreaudio

import (
	"encoding/binary"
	"math"
	"unsafe"
)

// GetDeviceIDs returns all audio device IDs known to the system.
func GetDeviceIDs() ([]DeviceID, error) {
	addr := PropertyAddress{
		Selector: HardwarePropertyDevices,
		Scope:    ScopeGlobal,
		Element:  ElementMain,
	}
	data, err := GetPropertyData(SystemObjectID, addr)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	count := len(data) / int(unsafe.Sizeof(AudioObjectID(0)))
	ids := make([]DeviceID, count)
	for i := range count {
		ids[i] = DeviceID(binary.LittleEndian.Uint32(data[i*4:]))
	}
	return ids, nil
}

// GetDeviceName returns the display name of a device.
func GetDeviceName(deviceID DeviceID) (string, error) {
	addr := PropertyAddress{
		Selector: DevicePropertyName,
		Scope:    ScopeGlobal,
		Element:  ElementMain,
	}
	data, err := GetPropertyData(deviceID, addr)
	if err != nil {
		return "", err
	}
	return cfStringFromPropertyData(data), nil
}

// GetDeviceUID returns the persistent UID string for a device.
func GetDeviceUID(deviceID DeviceID) (string, error) {
	addr := PropertyAddress{
		Selector: DevicePropertyDeviceUID,
		Scope:    ScopeGlobal,
		Element:  ElementMain,
	}
	data, err := GetPropertyData(deviceID, addr)
	if err != nil {
		return "", err
	}
	return cfStringFromPropertyData(data), nil
}

// GetDeviceTransportType returns how the device is connected.
func GetDeviceTransportType(deviceID DeviceID) (TransportType, error) {
	addr := PropertyAddress{
		Selector: DevicePropertyTransportType,
		Scope:    ScopeGlobal,
		Element:  ElementMain,
	}
	data, err := GetPropertyData(deviceID, addr)
	if err != nil {
		return TransportUnknown, err
	}
	if len(data) < 4 {
		return TransportUnknown, nil
	}
	return TransportType(binary.LittleEndian.Uint32(data)), nil
}

// IsOutputDevice returns true if the device has output streams.
func IsOutputDevice(deviceID DeviceID) (bool, error) {
	return hasStreamsInScope(deviceID, ScopeOutput)
}

// IsInputDevice returns true if the device has input streams.
func IsInputDevice(deviceID DeviceID) (bool, error) {
	return hasStreamsInScope(deviceID, ScopeInput)
}

// hasStreamsInScope checks whether a device has any streams in the given scope
// by querying the data size of the streams property. A non-zero size means
// at least one stream exists.
func hasStreamsInScope(deviceID DeviceID, scope PropertyScope) (bool, error) {
	addr := PropertyAddress{
		Selector: DevicePropertyStreams,
		Scope:    scope,
		Element:  ElementMain,
	}
	size, err := GetPropertyDataSize(deviceID, addr)
	if err != nil {
		// If the property doesn't exist for this scope, the device
		// simply doesn't support it -- not an error.
		if isUnknownPropertyError(err) {
			return false, nil
		}
		return false, err
	}
	return size > 0, nil
}

// isUnknownPropertyError returns true if the error is an UnknownPropertyError.
func isUnknownPropertyError(err error) bool {
	e, ok := err.(*Error)
	return ok && e.Code == UnknownPropertyError
}

// GetDeviceIsAlive returns whether the device is still connected and functional.
func GetDeviceIsAlive(deviceID DeviceID) (bool, error) {
	addr := PropertyAddress{
		Selector: DevicePropertyIsAlive,
		Scope:    ScopeGlobal,
		Element:  ElementMain,
	}
	data, err := GetPropertyData(deviceID, addr)
	if err != nil {
		return false, err
	}
	if len(data) < 4 {
		return false, nil
	}
	return binary.LittleEndian.Uint32(data) != 0, nil
}

// volumeAddresses returns the ordered list of property addresses to probe
// for volume control: VirtualMainVolume, then per-channel VolumeScalar on
// element 0 (master), then channels 1 and 2.
func volumeAddresses(scope PropertyScope) []PropertyAddress {
	return []PropertyAddress{
		{Selector: VirtualMainVolume, Scope: scope, Element: ElementMain},
		{Selector: DevicePropertyVolumeScalar, Scope: scope, Element: ElementMain},
		{Selector: DevicePropertyVolumeScalar, Scope: scope, Element: 1},
	}
}

// muteAddresses returns the ordered list of property addresses to probe
// for mute control.
func muteAddresses(scope PropertyScope) []PropertyAddress {
	return []PropertyAddress{
		{Selector: DevicePropertyMute, Scope: scope, Element: ElementMain},
		{Selector: DevicePropertyMute, Scope: scope, Element: 1},
	}
}

// HasVolumeControl returns true if the device has any volume control
// property on the given scope.
func HasVolumeControl(deviceID DeviceID, scope PropertyScope) bool {
	for _, addr := range volumeAddresses(scope) {
		if HasProperty(deviceID, addr) {
			return true
		}
	}
	return false
}

// HasMuteControl returns true if the device has any mute control
// property on the given scope.
func HasMuteControl(deviceID DeviceID, scope PropertyScope) bool {
	for _, addr := range muteAddresses(scope) {
		if HasProperty(deviceID, addr) {
			return true
		}
	}
	return false
}

// VolumeWatchAddresses returns all property addresses that should be
// watched for volume change events on the given scope.
func VolumeWatchAddresses(scope PropertyScope) []PropertyAddress {
	return volumeAddresses(scope)
}

// MuteWatchAddresses returns all property addresses that should be
// watched for mute change events on the given scope.
func MuteWatchAddresses(scope PropertyScope) []PropertyAddress {
	return muteAddresses(scope)
}

// getFloat32Property reads a float32 value from the first available
// property address in the list.
func getFloat32Property(deviceID DeviceID, addrs []PropertyAddress) (float32, error) {
	for _, addr := range addrs {
		if !HasProperty(deviceID, addr) {
			continue
		}
		data, err := GetPropertyData(deviceID, addr)
		if err != nil {
			continue
		}
		if len(data) < 4 {
			continue
		}
		return math.Float32frombits(binary.LittleEndian.Uint32(data)), nil
	}
	return 0, &Error{Code: UnknownPropertyError, Context: "GetProperty"}
}

// getBoolProperty reads a UInt32-as-bool value from the first available
// property address in the list.
func getBoolProperty(deviceID DeviceID, addrs []PropertyAddress) (bool, error) {
	for _, addr := range addrs {
		if !HasProperty(deviceID, addr) {
			continue
		}
		data, err := GetPropertyData(deviceID, addr)
		if err != nil {
			continue
		}
		if len(data) < 4 {
			continue
		}
		return binary.LittleEndian.Uint32(data) != 0, nil
	}
	return false, &Error{Code: UnknownPropertyError, Context: "GetProperty"}
}

// GetDeviceVolume returns the scalar volume (0.0-1.0) for a device on the
// output scope.
func GetDeviceVolume(deviceID DeviceID) (float32, error) {
	return GetDeviceVolumeForScope(deviceID, ScopeOutput)
}

// GetDeviceVolumeForScope returns the scalar volume (0.0-1.0) for a device
// on the given scope. It tries VirtualMainVolume first, then falls back to
// per-channel VolumeScalar.
func GetDeviceVolumeForScope(deviceID DeviceID, scope PropertyScope) (float32, error) {
	return getFloat32Property(deviceID, volumeAddresses(scope))
}

// SetDeviceVolume sets the scalar volume (0.0-1.0) for a device on the
// output scope.
func SetDeviceVolume(deviceID DeviceID, volume float32) error {
	return SetDeviceVolumeForScope(deviceID, ScopeOutput, volume)
}

// SetDeviceVolumeForScope sets the scalar volume (0.0-1.0) for a device
// on the given scope. It tries VirtualMainVolume first, then master
// VolumeScalar, then sets all per-channel VolumeScalar values.
func SetDeviceVolumeForScope(deviceID DeviceID, scope PropertyScope, volume float32) error {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, math.Float32bits(volume))

	// Try aggregate volume controls first.
	for _, addr := range []PropertyAddress{
		{Selector: VirtualMainVolume, Scope: scope, Element: ElementMain},
		{Selector: DevicePropertyVolumeScalar, Scope: scope, Element: ElementMain},
	} {
		if HasProperty(deviceID, addr) {
			return SetPropertyData(deviceID, addr, data)
		}
	}

	// Fall back to setting per-channel volume on all available channels.
	set := false
	for ch := PropertyElement(1); ch <= 2; ch++ {
		addr := PropertyAddress{Selector: DevicePropertyVolumeScalar, Scope: scope, Element: ch}
		if HasProperty(deviceID, addr) {
			if err := SetPropertyData(deviceID, addr, data); err == nil {
				set = true
			}
		}
	}
	if set {
		return nil
	}
	return &Error{Code: UnknownPropertyError, Context: "SetDeviceVolume"}
}

// GetDeviceMute returns whether the device output is muted.
func GetDeviceMute(deviceID DeviceID) (bool, error) {
	return GetDeviceMuteForScope(deviceID, ScopeOutput)
}

// GetDeviceMuteForScope returns whether the device is muted on the given
// scope. Tries the main element first, then falls back to channel 1.
func GetDeviceMuteForScope(deviceID DeviceID, scope PropertyScope) (bool, error) {
	return getBoolProperty(deviceID, muteAddresses(scope))
}

// SetDeviceMute sets the mute state on the device output.
func SetDeviceMute(deviceID DeviceID, muted bool) error {
	return SetDeviceMuteForScope(deviceID, ScopeOutput, muted)
}

// SetDeviceMuteForScope sets the mute state on the given scope.
// Tries the main element first, then falls back to per-channel.
func SetDeviceMuteForScope(deviceID DeviceID, scope PropertyScope, muted bool) error {
	data := make([]byte, 4)
	if muted {
		binary.LittleEndian.PutUint32(data, 1)
	}

	for _, addr := range muteAddresses(scope) {
		if HasProperty(deviceID, addr) {
			return SetPropertyData(deviceID, addr, data)
		}
	}
	return &Error{Code: UnknownPropertyError, Context: "SetDeviceMute"}
}
