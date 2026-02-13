//go:build darwin

package coreaudio

import "fmt"

// OSStatus is the CoreAudio/macOS error code type.
type OSStatus int32

// Common CoreAudio error codes.
const (
	NoError                   OSStatus = 0
	NotRunningError           OSStatus = 0x73746F70 // 'stop'
	UnspecifiedError          OSStatus = 0x77686174 // 'what'
	UnknownPropertyError      OSStatus = 0x77686F3F // 'who?'
	BadPropertySizeError      OSStatus = 0x2173697A // '!siz'
	IllegalOperationError     OSStatus = 0x6E6F7065 // 'nope'
	BadObjectError            OSStatus = 0x216F626A // '!obj'
	BadDeviceError            OSStatus = 0x21646576 // '!dev'
	BadStreamError            OSStatus = 0x21737472 // '!str'
	UnsupportedOperationError OSStatus = 0x756E6F70 // 'unop'
	DevicePermissionsError    OSStatus = 0x21686F67 // '!hog'
)

// Error represents a CoreAudio error with its OSStatus code.
type Error struct {
	Code    OSStatus
	Context string
}

func (e *Error) Error() string {
	if name, ok := statusNames[e.Code]; ok {
		return fmt.Sprintf("coreaudio: %s: %s (OSStatus %d)", e.Context, name, e.Code)
	}
	return fmt.Sprintf("coreaudio: %s: unknown error (OSStatus %d)", e.Context, e.Code)
}

var statusNames = map[OSStatus]string{
	NoError:                   "no error",
	NotRunningError:           "not running",
	UnspecifiedError:          "unspecified error",
	UnknownPropertyError:      "unknown property",
	BadPropertySizeError:      "bad property size",
	IllegalOperationError:     "illegal operation",
	BadObjectError:            "bad object",
	BadDeviceError:            "bad device",
	BadStreamError:            "bad stream",
	UnsupportedOperationError: "unsupported operation",
	DevicePermissionsError:    "device permissions (hog mode)",
}

// checkStatus converts a non-zero OSStatus to an error.
func checkStatus(status OSStatus, context string) error {
	if status == NoError {
		return nil
	}
	return &Error{Code: status, Context: context}
}
