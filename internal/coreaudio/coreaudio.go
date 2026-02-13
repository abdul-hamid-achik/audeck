//go:build darwin

package coreaudio

/*
#cgo darwin LDFLAGS: -framework CoreAudio -framework AudioToolbox -framework CoreFoundation
#include <CoreAudio/CoreAudio.h>
#include <AudioToolbox/AudioServices.h>
#include <CoreFoundation/CoreFoundation.h>

// Compatibility for macOS < 12 where ElementMain was named ElementMaster.
#ifndef kAudioObjectPropertyElementMain
#define kAudioObjectPropertyElementMain kAudioObjectPropertyElementMaster
#endif
*/
import "C"
import "unsafe"

// toCAddress converts a Go PropertyAddress to a C AudioObjectPropertyAddress.
func toCAddress(addr PropertyAddress) C.AudioObjectPropertyAddress {
	return C.AudioObjectPropertyAddress{
		mSelector: C.AudioObjectPropertySelector(addr.Selector),
		mScope:    C.AudioObjectPropertyScope(addr.Scope),
		mElement:  C.AudioObjectPropertyElement(addr.Element),
	}
}

// GetPropertyDataSize returns the byte size of a property's data.
func GetPropertyDataSize(objectID AudioObjectID, addr PropertyAddress) (uint32, error) {
	cAddr := toCAddress(addr)
	var dataSize C.UInt32
	status := C.AudioObjectGetPropertyDataSize(
		C.AudioObjectID(objectID),
		&cAddr,
		0,
		nil,
		&dataSize,
	)
	if err := checkStatus(OSStatus(status), "GetPropertyDataSize"); err != nil {
		return 0, err
	}
	return uint32(dataSize), nil
}

// GetPropertyData reads property data into a freshly allocated byte buffer.
func GetPropertyData(objectID AudioObjectID, addr PropertyAddress) ([]byte, error) {
	size, err := GetPropertyDataSize(objectID, addr)
	if err != nil {
		return nil, err
	}
	if size == 0 {
		return nil, nil
	}
	buf := make([]byte, size)
	cAddr := toCAddress(addr)
	cSize := C.UInt32(size)
	status := C.AudioObjectGetPropertyData(
		C.AudioObjectID(objectID),
		&cAddr,
		0,
		nil,
		&cSize,
		unsafe.Pointer(&buf[0]),
	)
	if err := checkStatus(OSStatus(status), "GetPropertyData"); err != nil {
		return nil, err
	}
	return buf[:cSize], nil
}

// SetPropertyData writes property data.
func SetPropertyData(objectID AudioObjectID, addr PropertyAddress, data []byte) error {
	if len(data) == 0 {
		return nil
	}
	cAddr := toCAddress(addr)
	status := C.AudioObjectSetPropertyData(
		C.AudioObjectID(objectID),
		&cAddr,
		0,
		nil,
		C.UInt32(len(data)),
		unsafe.Pointer(&data[0]),
	)
	return checkStatus(OSStatus(status), "SetPropertyData")
}

// HasProperty checks if an audio object has a specific property.
func HasProperty(objectID AudioObjectID, addr PropertyAddress) bool {
	cAddr := toCAddress(addr)
	return C.AudioObjectHasProperty(C.AudioObjectID(objectID), &cAddr) != 0
}

// IsPropertySettable checks if a property can be written.
func IsPropertySettable(objectID AudioObjectID, addr PropertyAddress) (bool, error) {
	cAddr := toCAddress(addr)
	var settable C.Boolean
	status := C.AudioObjectIsPropertySettable(
		C.AudioObjectID(objectID),
		&cAddr,
		&settable,
	)
	if err := checkStatus(OSStatus(status), "IsPropertySettable"); err != nil {
		return false, err
	}
	return settable != 0, nil
}
