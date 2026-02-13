# Audeck Architecture

## Overview

Audeck is a macOS TUI audio control panel built with Go. It uses CoreAudio via cgo
for hardware interaction and Bubble Tea for the terminal UI. The architecture is
organized into three main layers with clear boundaries.

```
+--------------------------------------------------+
|                    cmd/audeck                     |
|               (main, CLI entry point)             |
+--------------------------------------------------+
           |                          |
           v                          v
+---------------------+  +-------------------------+
|   internal/tui      |  |   internal/audio        |
|   (Bubble Tea UI)   |  |   (business logic)      |
|                     |  |                         |
| - Model/Update/View |  | - DeviceManager         |
| - Device list       |  | - Volume/mute control   |
| - Volume slider     |  | - Event subscription    |
| - Keybindings       |  | - Device switching      |
+---------------------+  +-------------------------+
           |                     |
           | tea.Msg             | calls
           +----+----------------+
                |
                v
+--------------------------------------------------+
|            internal/coreaudio                     |
|        (cgo bindings to CoreAudio)                |
|                                                   |
| - AudioObject property get/set                    |
| - Device enumeration                              |
| - Listener bridge (C callbacks -> Go channels)    |
| - OSStatus error mapping                          |
| - CFString conversion                             |
+--------------------------------------------------+
                |
                | cgo / LDFLAGS
                v
+--------------------------------------------------+
|     macOS CoreAudio + AudioToolbox Frameworks     |
+--------------------------------------------------+
```

## Package Structure

```
audeck/
  cmd/
    audeck/
      main.go              # CLI entry, initializes tea.Program
  internal/
    coreaudio/
      coreaudio.go         # Core cgo bindings, #include headers, property get/set
      types.go             # Go type definitions (AudioObjectID, etc.)
      properties.go        # Property address constants and helpers
      errors.go            # OSStatus -> error mapping
      listener.go          # Listener type, Watch/Close, registration tracking
      listener_gateway.go  # C gateway function (static inline in preamble)
      listener_export.go   # //export goPropertyListenerCallback (no C defs)
      device.go            # Device-specific queries (name, UID, volume)
      cfstring.go          # CFStringRef -> Go string conversion
    audio/
      manager.go           # DeviceManager: high-level audio operations
      device.go            # Device type with friendly API
      events.go            # Event types (DeviceChanged, VolumeChanged, etc.)
    tui/
      model.go             # Bubble Tea Model
      update.go            # Update function (key handling, messages)
      view.go              # View function (rendering)
      styles.go            # Lipgloss styles
      keys.go              # Key bindings
      components.go        # Reusable UI components (slider, device row)
  go.mod
  go.sum
```

---

## Layer 1: internal/coreaudio (cgo Bindings)

This is the lowest layer. It wraps the macOS CoreAudio C API with minimal
abstraction. No business logic lives here -- just type-safe access to the C API.

### Go Type Definitions (`types.go`)

```go
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
```

### Property Constants (`properties.go`)

```go
package coreaudio

// Defined via cgo references to C constants.
// Only the constants actually needed by audeck are exposed.

// Property scopes
const (
    ScopeGlobal      PropertyScope = 0x676C6F62 // 'glob'
    ScopeInput       PropertyScope = 0x696E7074 // 'inpt'
    ScopeOutput      PropertyScope = 0x6F757470 // 'outp'
)

// Property element
// NOTE: kAudioObjectPropertyElementMaster was deprecated in macOS 12 and
// renamed to kAudioObjectPropertyElementMain. Both have value 0.
// In our C preamble we add a compat define:
//   #ifndef kAudioObjectPropertyElementMain
//   #define kAudioObjectPropertyElementMain kAudioObjectPropertyElementMaster
//   #endif
const (
    ElementMain PropertyElement = 0 // kAudioObjectPropertyElementMain
)

// System object (singleton, ID = 1)
const SystemObjectID AudioObjectID = 1

// Hardware property selectors (on SystemObjectID)
const (
    HardwarePropertyDevices              PropertySelector = 0x64657623 // 'dev#'
    HardwarePropertyDefaultOutputDevice  PropertySelector = 0x646F7574 // 'dout'
    HardwarePropertyDefaultInputDevice   PropertySelector = 0x64696E20 // 'din '
)

// Device property selectors (on a DeviceID)
const (
    DevicePropertyName          PropertySelector = 0x6C6E616D // 'lnam' (kAudioObjectPropertyName)
    DevicePropertyDeviceUID     PropertySelector = 0x75696420 // 'uid '
    DevicePropertyTransportType PropertySelector = 0x7472616E // 'tran'
    DevicePropertyIsAlive       PropertySelector = 0x6C697665 // 'livn' -- via kAudioDevicePropertyDeviceIsAlive
    DevicePropertyVolumeScalar  PropertySelector = 0x766F6C6D // 'volm' -- on the output scope
    DevicePropertyMute          PropertySelector = 0x6D757465 // 'mute'
)

// AudioHardwareService property selectors (via AudioToolbox)
const (
    VirtualMainVolume PropertySelector = 0x766D7663 // 'vmvc'
)

// Transport types
const (
    TransportBuiltIn    TransportType = 0x626C746E // 'bltn'
    TransportUSB        TransportType = 0x75736220 // 'usb '
    TransportBluetooth  TransportType = 0x626C7565 // 'blue'
    TransportHDMI       TransportType = 0x68646D69 // 'hdmi'
    TransportDisplayPort TransportType = 0x64707274 // 'dprt'
    TransportAirPlay    TransportType = 0x61697270 // 'airp'
    TransportThunderbolt TransportType = 0x7468756E // 'thun'
    TransportVirtual    TransportType = 0x76697274 // 'virt'
    TransportUnknown    TransportType = 0
)
```

### Error Handling (`errors.go`)

CoreAudio returns `OSStatus` (int32) values. We map them to a concrete Go error type.

```go
package coreaudio

import "fmt"

// OSStatus is the CoreAudio/macOS error code type.
type OSStatus int32

// Common CoreAudio error codes.
const (
    NoError                  OSStatus = 0
    NotRunningError          OSStatus = 0x73746F70 // 'stop'
    UnspecifiedError         OSStatus = 0x77686174 // 'what'
    UnknownPropertyError     OSStatus = 0x77686F3F // 'who?'
    BadPropertySizeError     OSStatus = 0x2173697A // '!siz'
    IllegalOperationError    OSStatus = 0x6E6F7065 // 'nope'
    BadObjectError           OSStatus = 0x216F626A // '!obj'
    BadDeviceError           OSStatus = 0x21646576 // '!dev'
    BadStreamError           OSStatus = 0x21737472 // '!str'
    UnsupportedOperationError OSStatus = 0x756E6F70 // 'unop'
    DevicePermissionsError   OSStatus = 0x21686F67 // '!hog'
)

// Error represents a CoreAudio error with its OSStatus code.
type Error struct {
    Code    OSStatus
    Context string // e.g., "GetPropertyData" or "SetPropertyData"
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
```

### Core cgo Bindings (`coreaudio.go`)

```go
package coreaudio

/*
#cgo darwin LDFLAGS: -framework CoreAudio -framework AudioToolbox -framework CoreFoundation
#include <CoreAudio/CoreAudio.h>
#include <AudioToolbox/AudioServices.h>
#include <CoreFoundation/CoreFoundation.h>
#include "coreaudio.h"
*/
import "C"
import (
    "unsafe"
)

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

// GetPropertyData reads property data into the provided byte buffer.
func GetPropertyData(objectID AudioObjectID, addr PropertyAddress) ([]byte, error) {
    size, err := GetPropertyDataSize(objectID, addr)
    if err != nil {
        return nil, err
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
```

### Listener Bridge Pattern (Three-File Split)

This is the most architecturally critical piece. CoreAudio fires property change
notifications on arbitrary threads via C callbacks. We need to bridge these into
Go channels safely.

**Strategy: cgo.Handle + exported Go function + C gateway (three-file split)**

We use `runtime/cgo.Handle` (Go 1.17+) to safely pass Go state through C void
pointers. The C gateway is defined as a `static inline` function in a Go file
preamble, and the `//export` directive lives in a separate file (cgo requires
that files with `//export` must not have C definitions in the preamble).

**Why three files?** cgo has a rule: if a Go file uses `//export`, its C preamble
can only contain declarations (`extern`), not definitions. Splitting into three
files avoids linker errors while keeping all code in `.go` files (no separate
`.c`/`.h` files needed).

#### File 1: `listener_gateway.go` (C gateway, no //export)

```go
package coreaudio

/*
#include <CoreAudio/CoreAudio.h>
#include <stdint.h>

// Forward declaration of the Go callback.
extern void goPropertyListenerCallback(
    unsigned int objectID,
    unsigned int numAddresses,
    const AudioObjectPropertyAddress* addresses,
    uintptr_t handle
);

// C gateway function registered with CoreAudio as the property listener.
// Must be static inline since we take its address for AudioObjectAddPropertyListener.
static OSStatus propertyListenerGateway(
    AudioObjectID               inObjectID,
    UInt32                      inNumberAddresses,
    const AudioObjectPropertyAddress inAddresses[],
    void*                       inClientData
) {
    goPropertyListenerCallback(
        (unsigned int)inObjectID,
        (unsigned int)inNumberAddresses,
        inAddresses,
        (uintptr_t)inClientData
    );
    return 0;
}
*/
import "C"
```

#### File 2: `listener_export.go` (//export, no C definitions in preamble)

```go
package coreaudio

/*
#include <CoreAudio/CoreAudio.h>
*/
import "C"
import (
    "runtime/cgo"
    "unsafe"
)

//export goPropertyListenerCallback
func goPropertyListenerCallback(
    objectID C.uint,
    numAddresses C.uint,
    addresses *C.AudioObjectPropertyAddress,
    handle C.uintptr_t,
) {
    l := cgo.Handle(handle).Value().(*Listener)

    // Iterate over the addresses array using unsafe pointer arithmetic.
    addrs := unsafe.Slice(addresses, int(numAddresses))
    for _, addr := range addrs {
        event := PropertyEvent{
            ObjectID: AudioObjectID(objectID),
            Selector: PropertySelector(addr.mSelector),
            Scope:    PropertyScope(addr.mScope),
            Element:  PropertyElement(addr.mElement),
        }
        // Non-blocking send: drop events if consumer is too slow.
        select {
        case l.ch <- event:
        default:
        }
    }
}
```

#### File 3: `listener.go` (Listener type, Watch/Close)

```go
package coreaudio

/*
#include <CoreAudio/CoreAudio.h>

// Forward declaration so we can take the address of the gateway.
extern OSStatus propertyListenerGateway(
    AudioObjectID, UInt32, const AudioObjectPropertyAddress[], void*);
*/
import "C"
import (
    "runtime/cgo"
    "sync"
    "unsafe"
)

// PropertyEvent is emitted when a CoreAudio property changes.
type PropertyEvent struct {
    ObjectID AudioObjectID
    Selector PropertySelector
    Scope    PropertyScope
    Element  PropertyElement
}

// Listener receives CoreAudio property change notifications.
type Listener struct {
    ch            chan PropertyEvent
    handle        cgo.Handle
    mu            sync.Mutex
    registrations []registration
}

type registration struct {
    objectID AudioObjectID
    addr     PropertyAddress
}

// NewListener creates a listener that sends events to a buffered channel.
func NewListener(bufferSize int) *Listener {
    l := &Listener{
        ch: make(chan PropertyEvent, bufferSize),
    }
    l.handle = cgo.NewHandle(l)
    return l
}

// Events returns the channel that receives property change events.
func (l *Listener) Events() <-chan PropertyEvent {
    return l.ch
}

// Watch registers a property listener on the given audio object.
func (l *Listener) Watch(objectID AudioObjectID, addr PropertyAddress) error {
    cAddr := toCAddress(addr)
    status := C.AudioObjectAddPropertyListener(
        C.AudioObjectID(objectID),
        &cAddr,
        C.AudioObjectPropertyListenerProc(C.propertyListenerGateway),
        unsafe.Pointer(uintptr(l.handle)),
    )
    if err := checkStatus(OSStatus(status), "AddPropertyListener"); err != nil {
        return err
    }
    l.mu.Lock()
    l.registrations = append(l.registrations, registration{objectID, addr})
    l.mu.Unlock()
    return nil
}

// Unwatch removes a specific property listener.
func (l *Listener) Unwatch(objectID AudioObjectID, addr PropertyAddress) error {
    cAddr := toCAddress(addr)
    status := C.AudioObjectRemovePropertyListener(
        C.AudioObjectID(objectID),
        &cAddr,
        C.AudioObjectPropertyListenerProc(C.propertyListenerGateway),
        unsafe.Pointer(uintptr(l.handle)),
    )
    if err := checkStatus(OSStatus(status), "RemovePropertyListener"); err != nil {
        return err
    }
    l.mu.Lock()
    for i, r := range l.registrations {
        if r.objectID == objectID && r.addr == addr {
            l.registrations = append(l.registrations[:i], l.registrations[i+1:]...)
            break
        }
    }
    l.mu.Unlock()
    return nil
}

// Close removes all registered listeners and frees resources.
func (l *Listener) Close() error {
    l.mu.Lock()
    defer l.mu.Unlock()
    for _, r := range l.registrations {
        cAddr := toCAddress(r.addr)
        C.AudioObjectRemovePropertyListener(
            C.AudioObjectID(r.objectID),
            &cAddr,
            C.AudioObjectPropertyListenerProc(C.propertyListenerGateway),
            unsafe.Pointer(uintptr(l.handle)),
        )
    }
    l.registrations = nil
    l.handle.Delete()
    close(l.ch)
    return nil
}
```

**Key design decisions for the listener bridge:**

1. **Three-file split** -- avoids cgo linker errors with `//export` + C definitions
2. **`runtime/cgo.Handle`** (not a global map registry) -- officially supported since Go 1.17
3. **Handle passed as uintptr_t value** -- not as a pointer to the handle, which
   would be unsafe for async callbacks
4. **Non-blocking channel send** -- CoreAudio callbacks happen on system threads;
   we must not block or the audio subsystem stalls
5. **Single C gateway function** -- all property listeners use the same gateway;
   differentiation happens in Go via the PropertyEvent fields
6. **Unwatch method** -- needed when device list changes and we must unregister
   listeners for removed devices before registering for new ones
7. **Cleanup via Close()** -- unregisters all listeners and frees the handle

### Device Helpers (`device.go`)

```go
package coreaudio

import (
    "encoding/binary"
    "math"
    "unsafe"
)

// GetDeviceIDs returns all audio device IDs.
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
    count := len(data) / int(unsafe.Sizeof(AudioObjectID(0)))
    ids := make([]DeviceID, count)
    for i := 0; i < count; i++ {
        ids[i] = DeviceID(binary.LittleEndian.Uint32(data[i*4:]))
    }
    return ids, nil
}

// GetDefaultOutputDeviceID returns the current default output device.
func GetDefaultOutputDeviceID() (DeviceID, error) {
    addr := PropertyAddress{
        Selector: HardwarePropertyDefaultOutputDevice,
        Scope:    ScopeGlobal,
        Element:  ElementMain,
    }
    data, err := GetPropertyData(SystemObjectID, addr)
    if err != nil {
        return 0, err
    }
    return DeviceID(binary.LittleEndian.Uint32(data)), nil
}

// SetDefaultOutputDevice sets the default output device.
func SetDefaultOutputDevice(deviceID DeviceID) error {
    addr := PropertyAddress{
        Selector: HardwarePropertyDefaultOutputDevice,
        Scope:    ScopeGlobal,
        Element:  ElementMain,
    }
    data := make([]byte, 4)
    binary.LittleEndian.PutUint32(data, uint32(deviceID))
    return SetPropertyData(SystemObjectID, addr, data)
}

// GetDeviceVolume returns the scalar volume (0.0-1.0) for a device on the output scope.
func GetDeviceVolume(deviceID DeviceID) (float32, error) {
    addr := PropertyAddress{
        Selector: VirtualMainVolume,
        Scope:    ScopeOutput,
        Element:  ElementMain,
    }
    data, err := GetPropertyData(deviceID, addr)
    if err != nil {
        return 0, err
    }
    bits := binary.LittleEndian.Uint32(data)
    return math.Float32frombits(bits), nil
}

// SetDeviceVolume sets the scalar volume (0.0-1.0) for a device on the output scope.
func SetDeviceVolume(deviceID DeviceID, volume float32) error {
    addr := PropertyAddress{
        Selector: VirtualMainVolume,
        Scope:    ScopeOutput,
        Element:  ElementMain,
    }
    data := make([]byte, 4)
    binary.LittleEndian.PutUint32(data, math.Float32bits(volume))
    return SetPropertyData(deviceID, addr, data)
}

// GetDeviceMute returns whether the device output is muted.
func GetDeviceMute(deviceID DeviceID) (bool, error) {
    addr := PropertyAddress{
        Selector: DevicePropertyMute,
        Scope:    ScopeOutput,
        Element:  ElementMain,
    }
    data, err := GetPropertyData(deviceID, addr)
    if err != nil {
        return false, err
    }
    return binary.LittleEndian.Uint32(data) != 0, nil
}

// SetDeviceMute sets the mute state on the device output.
func SetDeviceMute(deviceID DeviceID, mute bool) error {
    addr := PropertyAddress{
        Selector: DevicePropertyMute,
        Scope:    ScopeOutput,
        Element:  ElementMain,
    }
    data := make([]byte, 4)
    if mute {
        binary.LittleEndian.PutUint32(data, 1)
    }
    return SetPropertyData(deviceID, addr, data)
}

// GetDeviceName returns the display name (CFString) of a device.
// This requires CoreFoundation CFString handling.
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
    return cfStringToGo(data), nil
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
    return cfStringToGo(data), nil
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
    return TransportType(binary.LittleEndian.Uint32(data)), nil
}
```

Note: `cfStringToGo` will be a helper that reads the CFStringRef pointer from
the property data and converts it to a Go string using CoreFoundation APIs.
This is handled in `coreaudio.go` or a separate `cfstring.go`.

---

## Layer 2: internal/audio (Business Logic)

This layer provides a clean Go API over the coreaudio package. It manages device
state, handles events, and exposes operations for the TUI layer.

### Event Types (`events.go`)

```go
package audio

// Event is the interface for all audio events.
type Event interface {
    audioEvent()
}

// DeviceListChanged fires when devices are added or removed.
type DeviceListChanged struct {
    Devices []Device
}

// DefaultDeviceChanged fires when the default output device changes.
type DefaultDeviceChanged struct {
    Device Device
}

// VolumeChanged fires when a device's volume changes.
type VolumeChanged struct {
    DeviceID uint32
    Volume   float32
}

// MuteChanged fires when a device's mute state changes.
type MuteChanged struct {
    DeviceID uint32
    Muted    bool
}

func (DeviceListChanged) audioEvent()    {}
func (DefaultDeviceChanged) audioEvent() {}
func (VolumeChanged) audioEvent()        {}
func (MuteChanged) audioEvent()          {}
```

### Device (`device.go`)

```go
package audio

// Device represents an audio output device with its current state.
type Device struct {
    ID            uint32
    UID           string
    Name          string
    TransportType string   // "built-in", "usb", "bluetooth", "hdmi", etc.
    Volume        float32  // 0.0 to 1.0
    Muted         bool
    IsDefault     bool
    HasVolume     bool     // whether volume control is available
    HasMute       bool     // whether mute control is available
}
```

### DeviceManager (`manager.go`)

```go
package audio

import "context"

// DeviceManager provides high-level audio device control.
// It owns the coreaudio.Listener and translates low-level property events
// into typed audio.Event values.
type DeviceManager interface {
    // Devices returns the current list of output devices.
    Devices() []Device

    // DefaultDevice returns the current default output device.
    DefaultDevice() (Device, error)

    // SetDefaultDevice changes the default output device.
    SetDefaultDevice(deviceID uint32) error

    // SetVolume sets the volume for a device (0.0 to 1.0).
    SetVolume(deviceID uint32, volume float32) error

    // SetMute sets the mute state for a device.
    SetMute(deviceID uint32, muted bool) error

    // ToggleMute toggles the mute state for a device.
    ToggleMute(deviceID uint32) error

    // Subscribe returns a channel that emits audio events.
    // The channel is closed when the context is cancelled.
    Subscribe(ctx context.Context) <-chan Event

    // Close shuts down the manager and releases CoreAudio resources.
    Close() error
}
```

**Implementation notes:**
- `DeviceManager` holds a `coreaudio.Listener` internally
- On creation, it calls `Listener.Watch()` for:
  - `SystemObjectID` + `HardwarePropertyDevices` (device list changes)
  - `SystemObjectID` + `HardwarePropertyDefaultOutputDevice` (default change)
  - Each known device + `VirtualMainVolume` (volume changes)
  - Each known device + `DevicePropertyMute` (mute changes)
- A goroutine reads `Listener.Events()` and translates `PropertyEvent` values
  into typed `audio.Event` values, re-querying the device state as needed
- When the device list changes, it re-registers listeners for new devices
  and unregisters for removed devices

---

## Layer 3: internal/tui (Bubble Tea UI)

The TUI layer consumes `audio.Event` values and renders the interface.

### Integration with Bubble Tea

```go
package tui

import (
    "context"
    tea "github.com/charmbracelet/bubbletea"
    "audeck/internal/audio"
)

// audioEventMsg wraps an audio.Event for Bubble Tea's message system.
type audioEventMsg struct {
    event audio.Event
}

// listenForAudioEvents returns a tea.Cmd that bridges audio events into
// Bubble Tea messages. It reads from the audio event channel and sends
// each event as a tea.Msg.
func listenForAudioEvents(ctx context.Context, mgr audio.DeviceManager) tea.Cmd {
    return func() tea.Msg {
        ch := mgr.Subscribe(ctx)
        event, ok := <-ch
        if !ok {
            return nil // channel closed
        }
        return audioEventMsg{event: event}
    }
}
```

**Key pattern:** Each audio event is wrapped in a `tea.Msg` and processed in the
`Update` function. After handling, a new `listenForAudioEvents` command is
returned to continue listening. This is the standard Bubble Tea pattern for
long-running subscriptions.

### Model Structure

```go
package tui

import "audeck/internal/audio"

type Model struct {
    manager       audio.DeviceManager
    devices       []audio.Device
    cursor        int          // selected device index
    width, height int          // terminal dimensions
    err           error
    quitting      bool
}
```

---

## Data Flow: Volume Change Example

This traces a complete round-trip for a volume change:

```
1. User presses Right arrow key
   |
2. tui.Update receives tea.KeyMsg("right")
   |
3. tui calls manager.SetVolume(selectedDeviceID, currentVolume + 0.05)
   |
4. audio.DeviceManager calls coreaudio.SetDeviceVolume(deviceID, newVolume)
   |
5. coreaudio.SetDeviceVolume calls C.AudioObjectSetPropertyData(...)
   |
6. CoreAudio applies the change and notifies listeners
   |
7. C propertyListenerBridge() is called on a CoreAudio thread
   |
8. propertyListenerBridge calls goPropertyListener() [exported Go func]
   |
9. goPropertyListener sends PropertyEvent to Listener.ch
   |
10. DeviceManager goroutine reads PropertyEvent from Listener.ch
    |
11. DeviceManager queries new volume, creates VolumeChanged event
    |
12. VolumeChanged sent on Subscribe() channel
    |
13. tui receives audioEventMsg{VolumeChanged{...}}
    |
14. tui.Update updates Model.devices[i].Volume
    |
15. tui.View re-renders the volume slider
```

---

## Data Flow: Device Hot-Plug Example

```
1. User plugs in USB headphones
   |
2. CoreAudio detects new device, fires property change for
   SystemObject + kAudioHardwarePropertyDevices
   |
3. C propertyListenerBridge -> goPropertyListener -> Listener.ch
   |
4. DeviceManager reads PropertyEvent, sees HardwarePropertyDevices changed
   |
5. DeviceManager calls coreaudio.GetDeviceIDs() to get new list
   |
6. DeviceManager registers volume/mute listeners on new devices
   |
7. DeviceManager emits DeviceListChanged{newDeviceList}
   |
8. tui receives the event, updates device list, re-renders
```

---

## Thread Safety Model

```
+-------------------+     +-------------------+     +-------------------+
| CoreAudio Thread  |     | Manager Goroutine |     | Bubble Tea Loop   |
| (C callbacks)     |     | (event processor) |     | (main goroutine)  |
+-------------------+     +-------------------+     +-------------------+
        |                         |                         |
        | PropertyEvent           |                         |
        |--- ch (buffered) ------>|                         |
        |                         | audio.Event             |
        |                         |--- ch (buffered) ------>|
        |                         |                         |
        |                         |                         | SetVolume()
        |                         |                         |----> (via mgr)
        |                         |                         |        |
        |                         |                         |  cgo call (safe,
        |                         |                         |  goroutine-safe)
```

**Rules:**
- CoreAudio callbacks run on arbitrary OS threads -- they MUST NOT block
- The Listener uses non-blocking channel sends (drop if full)
- The Manager goroutine is the only reader of Listener.Events()
- cgo calls to Get/Set property are goroutine-safe (CoreAudio API is thread-safe)
- The TUI goroutine calls Manager methods directly (they don't block long)

---

## Build Requirements

```
# go.mod
module audeck

go 1.22

require (
    github.com/charmbracelet/bubbletea v1.3.4
    github.com/charmbracelet/lipgloss  v1.0.0
    github.com/charmbracelet/bubbles   v0.20.0
)
```

```
# cgo flags (in coreaudio.go preamble)
#cgo darwin LDFLAGS: -framework CoreAudio -framework AudioToolbox -framework CoreFoundation
```

Build constraint: this project only builds on macOS (darwin).

```go
//go:build darwin
```

---

## Interface Contracts Summary

### coreaudio -> audio

The `audio` package depends on `internal/coreaudio` for:
- `GetDeviceIDs()`, `GetDefaultOutputDeviceID()`, `SetDefaultOutputDevice()`
- `GetDeviceVolume()`, `SetDeviceVolume()`
- `GetDeviceMute()`, `SetDeviceMute()`
- `GetDeviceName()`, `GetDeviceUID()`, `GetDeviceTransportType()`
- `NewListener()`, `Listener.Watch()`, `Listener.Events()`, `Listener.Close()`

### audio -> tui

The `tui` package depends on `internal/audio` for:
- `DeviceManager` interface (all methods)
- `Device` struct (for display)
- `Event` types (`VolumeChanged`, `MuteChanged`, `DeviceListChanged`, `DefaultDeviceChanged`)

### tui -> cmd/audeck

The `cmd/audeck` package:
- Creates a `DeviceManager` (the concrete implementation)
- Creates a `tui.Model` with the manager
- Starts `tea.NewProgram(model)`

---

## CFString Handling

CoreAudio returns device names and UIDs as `CFStringRef` values. The property
data contains a pointer to a CFString object. We need CoreFoundation to extract
the string:

```go
// cfStringToGo converts property data containing a CFStringRef to a Go string.
// The CFStringRef is released after conversion.
func cfStringToGo(data []byte) string {
    // data contains a CFStringRef (pointer-sized value)
    ref := *(*C.CFStringRef)(unsafe.Pointer(&data[0]))
    if ref == 0 {
        return ""
    }
    defer C.CFRelease(C.CFTypeRef(ref))

    length := C.CFStringGetLength(ref)
    maxSize := C.CFStringGetMaximumSizeForEncoding(length, C.kCFStringEncodingUTF8) + 1
    buf := C.malloc(C.size_t(maxSize))
    defer C.free(buf)

    if C.CFStringGetCString(ref, (*C.char)(buf), maxSize, C.kCFStringEncodingUTF8) == 0 {
        return ""
    }
    return C.GoString((*C.char)(buf))
}
```

---

## Design Decisions Log

| Decision | Rationale |
|----------|-----------|
| `runtime/cgo.Handle` over global map registry | Official API since Go 1.17, no manual bookkeeping |
| Non-blocking channel send in listener | CoreAudio callbacks must not block; dropped events are acceptable since we re-query state |
| Single C bridge function for all listeners | Reduces C code surface; Go-side differentiates by PropertyEvent fields |
| `DeviceManager` as interface | Enables testing with mocks; decouples TUI from CoreAudio |
| Typed audio events over raw property events | TUI should not know about CoreAudio internals |
| VirtualMainVolume over per-channel volume | Simpler UX; matches System Preferences behavior |
| Buffered channels (default 64) | Absorbs bursts of property change notifications |
| `//go:build darwin` on all coreaudio files | Prevents compilation errors on non-macOS platforms |
| Three-file split for listener bridge | cgo requires `//export` files to have no C definitions in preamble |
| `Unwatch()` method on Listener | Needed for dynamic listener management when device list changes |
| Compat define for `kAudioObjectPropertyElementMain` | Deprecated name `ElementMaster` in macOS 12; both are value 0 |
