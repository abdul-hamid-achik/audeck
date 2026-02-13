//go:build darwin

package coreaudio

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// ErrNoDevice is returned when the default device is not set (ID == 0).
var ErrNoDevice = errors.New("coreaudio: no device available")

// defaultDeviceAddress builds the PropertyAddress for a default device selector.
func defaultDeviceAddress(selector PropertySelector) PropertyAddress {
	return PropertyAddress{
		Selector: selector,
		Scope:    ScopeGlobal,
		Element:  ElementMain,
	}
}

// GetDefaultOutputDevice returns the ID of the current default output device.
// Returns ErrNoDevice if no output device is available.
func GetDefaultOutputDevice() (DeviceID, error) {
	return getDefaultDevice(HardwarePropertyDefaultOutputDevice)
}

// SetDefaultOutputDevice sets the system default output device.
// Returns an error if the device ID is invalid or the operation is not permitted.
func SetDefaultOutputDevice(id DeviceID) error {
	return setDefaultDevice(HardwarePropertyDefaultOutputDevice, id)
}

// GetDefaultInputDevice returns the ID of the current default input device.
// Returns ErrNoDevice if no input device is available.
func GetDefaultInputDevice() (DeviceID, error) {
	return getDefaultDevice(HardwarePropertyDefaultInputDevice)
}

// SetDefaultInputDevice sets the system default input device.
// Returns an error if the device ID is invalid or the operation is not permitted.
func SetDefaultInputDevice(id DeviceID) error {
	return setDefaultDevice(HardwarePropertyDefaultInputDevice, id)
}

// getDefaultDevice reads a default device property from the system object.
func getDefaultDevice(selector PropertySelector) (DeviceID, error) {
	addr := defaultDeviceAddress(selector)

	data, err := GetPropertyData(SystemObjectID, addr)
	if err != nil {
		return 0, fmt.Errorf("coreaudio: get default device: %w", err)
	}
	if len(data) < 4 {
		return 0, ErrNoDevice
	}

	id := DeviceID(binary.LittleEndian.Uint32(data[:4]))
	if id == 0 {
		return 0, ErrNoDevice
	}
	return id, nil
}

// setDefaultDevice writes a default device property on the system object.
func setDefaultDevice(selector PropertySelector, id DeviceID) error {
	if id == 0 {
		return ErrNoDevice
	}

	addr := defaultDeviceAddress(selector)

	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(id))

	if err := SetPropertyData(SystemObjectID, addr, data); err != nil {
		return fmt.Errorf("coreaudio: set default device: %w", err)
	}
	return nil
}
