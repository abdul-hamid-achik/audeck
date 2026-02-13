# Cgo + CoreAudio Integration Patterns

Research document for the **audeck** project. Covers best practices, thread safety,
callback patterns, memory management, and common pitfalls when wrapping macOS
CoreAudio APIs from Go via cgo.

---

## Table of Contents

1. [Linking macOS Frameworks with Cgo](#1-linking-macos-frameworks-with-cgo)
2. [CoreAudio Object Model Overview](#2-coreaudio-object-model-overview)
3. [Thread Safety and `runtime.LockOSThread`](#3-thread-safety-and-runtimelockosthread)
4. [Callback Patterns for Property Listeners](#4-callback-patterns-for-property-listeners)
5. [Memory Management Between C and Go](#5-memory-management-between-c-and-go)
6. [Pointer Passing with `runtime/cgo.Handle`](#6-pointer-passing-with-runtimecgohandle)
7. [Reference Implementations](#7-reference-implementations)
8. [Common Pitfalls and How to Avoid Them](#8-common-pitfalls-and-how-to-avoid-them)
9. [Recommended Architecture for audeck](#9-recommended-architecture-for-audeck)

---

## 1. Linking macOS Frameworks with Cgo

Use `#cgo LDFLAGS` directives to link against macOS frameworks. Each framework
needs its own `-framework` flag. The `darwin` build constraint restricts the
link flags to macOS only.

```go
package coreaudio

/*
#cgo darwin LDFLAGS: -framework CoreAudio -framework CoreFoundation
#cgo darwin CFLAGS: -x objective-c
#include <CoreAudio/CoreAudio.h>
#include <CoreFoundation/CoreFoundation.h>
*/
import "C"
```

**Key rules:**

- There must be **no blank line** between the cgo comment block and `import "C"`.
- Use build tags or the `darwin` constraint in `#cgo` directives so the package
  compiles only on macOS.
- Place C preamble code in the comment block. If using `//export`, definitions
  must go in a **separate `.go` file** or use the `static inline` trick.
- Headers to include: `<CoreAudio/CoreAudio.h>` for the HAL API,
  `<CoreAudio/AudioHardware.h>` for device enumeration,
  `<CoreFoundation/CoreFoundation.h>` for CFString handling.

### Build Tags for Platform Restriction

```go
//go:build darwin

package coreaudio
```

This ensures the package is only compiled on macOS. All `.go` files in the
coreaudio package should carry this tag.

---

## 2. CoreAudio Object Model Overview

CoreAudio uses an **object-property** model. Everything is an `AudioObjectID`.
Properties are addressed with `AudioObjectPropertyAddress`:

```c
typedef struct {
    AudioObjectPropertySelector mSelector;   // What property
    AudioObjectPropertyScope    mScope;      // Input, Output, or Global
    AudioObjectPropertyElement  mElement;    // Usually kAudioObjectPropertyElementMain
} AudioObjectPropertyAddress;
```

### Core API Functions

| Function | Purpose |
|---|---|
| `AudioObjectGetPropertyDataSize` | Get the byte size of a property value |
| `AudioObjectGetPropertyData` | Read a property value |
| `AudioObjectSetPropertyData` | Write a property value |
| `AudioObjectHasProperty` | Check if an object supports a property |
| `AudioObjectAddPropertyListener` | Register a callback for property changes |
| `AudioObjectRemovePropertyListener` | Unregister a callback |

### Important Constants

```go
const (
    AudioObjectSystemObject = C.kAudioObjectSystemObject  // ID = 1

    // Property selectors
    HardwarePropertyDevices              = C.kAudioHardwarePropertyDevices
    HardwarePropertyDefaultInputDevice   = C.kAudioHardwarePropertyDefaultInputDevice
    HardwarePropertyDefaultOutputDevice  = C.kAudioHardwarePropertyDefaultOutputDevice

    // Scopes
    PropertyScopeGlobal = C.kAudioObjectPropertyScopeGlobal
    PropertyScopeInput  = C.kAudioObjectPropertyScopeInput
    PropertyScopeOutput = C.kAudioObjectPropertyScopeOutput

    // Elements
    PropertyElementMain = C.kAudioObjectPropertyElementMain  // was ElementMaster before macOS 12
)
```

### Pattern: Enumerate Audio Devices (C Reference from SwitchAudioSource)

```c
AudioObjectPropertyAddress addr = {
    kAudioHardwarePropertyDevices,
    kAudioObjectPropertyScopeGlobal,
    kAudioObjectPropertyElementMain
};
UInt32 dataSize = 0;
AudioObjectGetPropertyDataSize(kAudioObjectSystemObject, &addr, 0, NULL, &dataSize);

int numDevices = dataSize / sizeof(AudioDeviceID);
AudioDeviceID devices[64];
AudioObjectGetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, &dataSize, devices);
```

### Pattern: Get/Set Default Output Device

```c
// Get
AudioObjectPropertyAddress addr = {
    kAudioHardwarePropertyDefaultOutputDevice,
    kAudioObjectPropertyScopeGlobal,
    kAudioObjectPropertyElementMain
};
AudioDeviceID deviceID;
UInt32 size = sizeof(AudioDeviceID);
AudioObjectGetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, &size, &deviceID);

// Set
AudioObjectSetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, sizeof(AudioDeviceID), &newDeviceID);
```

### Pattern: Get Device Name (CFString Handling)

```c
AudioObjectPropertyAddress addr = {
    kAudioDevicePropertyDeviceNameCFString,
    kAudioObjectPropertyScopeGlobal,
    kAudioObjectPropertyElementMain
};
CFStringRef cfName = NULL;
UInt32 size = sizeof(CFStringRef);
AudioObjectGetPropertyData(deviceID, &addr, 0, NULL, &size, &cfName);

char name[256];
CFStringGetCString(cfName, name, sizeof(name), kCFStringEncodingUTF8);
CFRelease(cfName);  // IMPORTANT: caller must release
```

---

## 3. Thread Safety and `runtime.LockOSThread`

### The Problem

Some macOS APIs (Cocoa/AppKit/UIKit) require calls from the **main OS thread**.
CoreAudio's HAL API is generally thread-safe for property queries and does NOT
require calls from the main thread. However:

- **Property listener callbacks** are invoked on **CoreAudio's internal threads**,
  which are **not** Go threads. This is the critical challenge.
- If you create aggregate devices or interact with AudioUnits, some operations
  may require the main thread.

### When to Use `runtime.LockOSThread`

For CoreAudio property queries (`AudioObjectGetPropertyData`,
`AudioObjectSetPropertyData`), `LockOSThread` is **not required**. These are
thread-safe.

For interaction with `NSApplication` or AudioUnits that touch the UI, lock the
main thread via `init()`:

```go
func init() {
    runtime.LockOSThread()
}
```

**The canonical pattern** from the Go wiki (Russ Cox):

```go
package coreaudio

import "runtime"

func init() {
    runtime.LockOSThread()
}

// Main runs the main CoreAudio event loop.
// Must be called from main.main.
func Main() {
    for f := range mainfunc {
        f()
    }
}

var mainfunc = make(chan func())

// do runs f on the main thread.
func do(f func()) {
    done := make(chan bool, 1)
    mainfunc <- func() {
        f()
        done <- true
    }
    <-done
}
```

**For audeck:** Since we are only doing HAL-level property queries and listeners,
we likely do NOT need `LockOSThread` for the core functionality. The Bubble Tea
TUI runs its own event loop. We should keep CoreAudio operations in a separate
goroutine and communicate via channels.

---

## 4. Callback Patterns for Property Listeners

This is the most complex part of the integration. CoreAudio fires property
listener callbacks on **non-Go threads** (threads created by CoreAudio
internally). Go supports this since Go 1.6 -- cgo callbacks from non-Go threads
are handled by the runtime creating a new goroutine on the calling thread.

### The CoreAudio Listener Signature

```c
typedef OSStatus (*AudioObjectPropertyListenerProc)(
    AudioObjectID                       inObjectID,
    UInt32                              inNumberAddresses,
    const AudioObjectPropertyAddress    inAddresses[],
    void* __nullable                    inClientData
);
```

### Safe Callback Architecture

The callback must be a **C function** (not a Go function directly). The C
function then calls an exported Go function. The `inClientData` pointer carries
context back to Go.

**Step 1: C gateway function** (in a separate `.go` file)

```go
package coreaudio

/*
#include <CoreAudio/CoreAudio.h>

extern void goPropertyListenerCallback(
    unsigned int objectID,
    unsigned int numAddresses,
    const AudioObjectPropertyAddress *addresses,
    uintptr_t handle
);

static OSStatus propertyListenerGateway(
    AudioObjectID                    inObjectID,
    UInt32                           inNumberAddresses,
    const AudioObjectPropertyAddress inAddresses[],
    void*                            inClientData
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

**Step 2: Exported Go callback** (in a different `.go` file)

```go
package coreaudio

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
    h := cgo.Handle(handle)
    listener := h.Value().(*PropertyListener)

    // Convert addresses to Go slice (without copying, backed by C memory)
    addrs := unsafe.Slice(addresses, int(numAddresses))
    for _, addr := range addrs {
        listener.notify(PropertyAddress{
            Selector: uint32(addr.mSelector),
            Scope:    uint32(addr.mScope),
            Element:  uint32(addr.mElement),
        })
    }
}
```

**Step 3: Registration**

```go
type PropertyListener struct {
    handle   cgo.Handle
    ch       chan PropertyAddress
    objectID AudioObjectID
    address  AudioObjectPropertyAddress
}

func NewPropertyListener(objectID AudioObjectID, addr PropertyAddress) *PropertyListener {
    l := &PropertyListener{
        ch:       make(chan PropertyAddress, 16),
        objectID: objectID,
        address:  addr.toCAddress(),
    }
    l.handle = cgo.NewHandle(l)

    status := C.AudioObjectAddPropertyListener(
        C.AudioObjectID(objectID),
        &l.address,
        C.AudioObjectPropertyListenerProc(C.propertyListenerGateway),
        unsafe.Pointer(uintptr(l.handle)),
    )
    if status != 0 {
        l.handle.Delete()
        return nil
    }
    return l
}

func (l *PropertyListener) Close() {
    C.AudioObjectRemovePropertyListener(
        C.AudioObjectID(l.objectID),
        &l.address,
        C.AudioObjectPropertyListenerProc(C.propertyListenerGateway),
        unsafe.Pointer(uintptr(l.handle)),
    )
    l.handle.Delete()
    close(l.ch)
}

func (l *PropertyListener) notify(addr PropertyAddress) {
    select {
    case l.ch <- addr:
    default:
        // Drop notification if channel is full (backpressure)
    }
}

func (l *PropertyListener) Events() <-chan PropertyAddress {
    return l.ch
}
```

### Critical Notes on Callbacks from Non-Go Threads

1. **Go 1.6+** supports cgo callbacks from threads not created by Go. The runtime
   will create a new `m` (machine/OS-thread) and a temporary `g` (goroutine) to
   handle the callback.

2. **Keep callbacks short.** The callback runs on CoreAudio's real-time thread.
   Do minimal work: send a notification to a channel, then return. Never block,
   allocate heavily, or call back into CoreAudio from within the callback.

3. **Do not hold locks in callbacks.** The CoreAudio thread may hold internal locks.
   Attempting to acquire another lock that interacts with CoreAudio can deadlock.

4. **Channel sends with `select/default`** prevent blocking the CoreAudio thread
   if the Go side is slow to consume events.

---

## 5. Memory Management Between C and Go

### Rules

| Scenario | Who Allocates | Who Frees |
|---|---|---|
| `C.CString(goStr)` | C (via malloc) | Go must call `C.free()` |
| `C.GoString(cStr)` | Go (copies data) | Go GC handles it |
| `AudioObjectGetPropertyData` returning `CFStringRef` | CoreAudio | Go must call `C.CFRelease()` |
| `cgo.NewHandle(value)` | Go runtime | Go must call `handle.Delete()` |
| C struct on stack | C stack | Automatic |
| `unsafe.Slice` over C array | C (original) | Do NOT free from Go |

### Pattern: Safe CFString Conversion

```go
func cfStringToGo(cfStr C.CFStringRef) string {
    if cfStr == 0 {
        return ""
    }
    length := C.CFStringGetLength(cfStr)
    maxSize := C.CFStringGetMaximumSizeForEncoding(length, C.kCFStringEncodingUTF8) + 1

    buf := C.malloc(C.size_t(maxSize))
    defer C.free(buf)

    if C.CFStringGetCString(cfStr, (*C.char)(buf), maxSize, C.kCFStringEncodingUTF8) == 0 {
        return ""
    }
    return C.GoString((*C.char)(buf))
}
```

### Pattern: Safe Property Data Reading

```go
func getPropertyData[T any](objectID C.AudioObjectID, addr *C.AudioObjectPropertyAddress) (T, error) {
    var value T
    size := C.UInt32(unsafe.Sizeof(value))

    status := C.AudioObjectGetPropertyData(objectID, addr, 0, nil, &size, unsafe.Pointer(&value))
    if status != 0 {
        return value, fmt.Errorf("AudioObjectGetPropertyData failed: %d", status)
    }
    return value, nil
}
```

### Pattern: Get Variable-Size Property Data (Device List)

```go
func getDeviceIDs() ([]C.AudioDeviceID, error) {
    addr := C.AudioObjectPropertyAddress{
        mSelector: C.kAudioHardwarePropertyDevices,
        mScope:    C.kAudioObjectPropertyScopeGlobal,
        mElement:  C.kAudioObjectPropertyElementMain,
    }

    var dataSize C.UInt32
    status := C.AudioObjectGetPropertyDataSize(
        C.kAudioObjectSystemObject, &addr, 0, nil, &dataSize,
    )
    if status != 0 {
        return nil, fmt.Errorf("GetPropertyDataSize failed: %d", status)
    }

    count := int(dataSize) / int(unsafe.Sizeof(C.AudioDeviceID(0)))
    devices := make([]C.AudioDeviceID, count)

    status = C.AudioObjectGetPropertyData(
        C.kAudioObjectSystemObject, &addr, 0, nil, &dataSize,
        unsafe.Pointer(&devices[0]),
    )
    if status != 0 {
        return nil, fmt.Errorf("GetPropertyData failed: %d", status)
    }

    return devices, nil
}
```

---

## 6. Pointer Passing with `runtime/cgo.Handle`

Since Go 1.17, `runtime/cgo.Handle` is the **recommended** way to pass Go values
through C code. It replaces the older pattern of manual registries with
`sync.Mutex`.

### Why Not Pass Go Pointers Directly?

The cgo pointer passing rules state:

> Go code may pass a Go pointer to C provided the Go memory to which it points
> does not contain any Go pointers.

Most Go values (strings, slices, interfaces, function values, structs containing
any of these) violate this rule. The Go GC may move objects, invalidating raw
pointers held by C code.

### How `cgo.Handle` Works

`cgo.Handle` is a `uintptr` that maps to a Go value stored in a global,
GC-visible map inside the runtime. The handle integer itself is safe to pass
to C (it contains no Go pointers). When C passes it back, Go can retrieve
the original value.

```go
import "runtime/cgo"

// Store a Go value, get a handle
h := cgo.NewHandle(myGoValue)

// Pass to C as uintptr_t
C.someFunction(C.uintptr_t(h))

// In the callback, retrieve the value
val := cgo.Handle(h).Value().(MyType)

// When done, free the handle
h.Delete()
```

### Critical Pitfall: Async Callbacks and Handle Lifetime

If the handle is used in an **asynchronous** C callback (like
`AudioObjectAddPropertyListener`), the handle must remain valid for the entire
lifetime of the listener registration. Do NOT pass `&handle` as `void*` because
the local variable may go out of scope. Instead, convert the handle value
itself to a `uintptr_t` and pass that:

```go
// CORRECT: handle value (integer) passed directly
C.AudioObjectAddPropertyListener(
    objectID, &addr, callback,
    unsafe.Pointer(uintptr(handle)),  // uintptr -> void*
)

// WRONG: pointer to handle (local variable address)
C.AudioObjectAddPropertyListener(
    objectID, &addr, callback,
    unsafe.Pointer(&handle),  // DANGER: &handle may be moved/collected
)
```

### Handle Lifecycle Management

```
NewHandle() -----> pass to C -----> C stores/uses it
                                         |
                                    callback fires
                                         |
                                    Value() retrieves Go object
                                         |
RemovePropertyListener() <------------- done
         |
    handle.Delete()   <-- MUST be called to avoid memory leak
```

---

## 7. Reference Implementations

### moriyoshi/go-coreaudio

The most direct Go binding for CoreAudio. Key observations:

- Links with `#cgo darwin LDFLAGS: -framework AudioUnit -framework CoreAudio`
- Maps all CoreAudio constants to Go constants using `C.kAudioXxx` pattern
- Uses `C.AudioObjectGetPropertyData` directly from Go
- Relatively thin wrapper -- does not abstract away the property address model

Source: https://github.com/moriyoshi/go-coreaudio

### progrium/darwinkit

A comprehensive macOS framework binding generator for Go. The `macos/coreaudio`
package provides higher-level Go types. Uses code generation from Apple's
framework headers.

Source: https://pkg.go.dev/github.com/progrium/darwinkit/macos/coreaudio

### deweller/switchaudio-osx (C Reference)

Pure C implementation of audio device switching. Key patterns to adopt:

- **Device enumeration**: Query `kAudioHardwarePropertyDevices` on `kAudioObjectSystemObject`
- **Device type detection**: Check `kAudioDevicePropertyStreams` with input/output scope
- **Device switching**: `AudioObjectSetPropertyData` on system object with `kAudioHardwarePropertyDefaultOutputDevice`
- **Device name retrieval**: `kAudioDevicePropertyDeviceNameCFString` with `CFStringGetCString`
- **Volume/mute control**: `kAudioDevicePropertyMute`, `kAudioHardwareServiceSetOutputDeviceVolume`

### ExistentialAudio/BlackHole (C Reference)

Virtual audio driver implementing `AudioServerPlugIn`. Key observations:

- Uses `pthread_mutex_t` for all state protection (not CoreAudio thread primitives)
- Ring buffer pattern for audio data passing between input and output streams
- Supports multiple sample rates via property negotiation
- All property access goes through `HasProperty` / `GetPropertyData` / `SetPropertyData`
- Device state (sample rate, volume, mute) stored in global variables protected by mutex

**Key insight from BlackHole:** Even at the driver level, CoreAudio uses the
same object-property model. The property address pattern is universal.

### gen2brain/malgo (miniaudio bindings)

Cross-platform audio library with CoreAudio backend. Uses miniaudio's C
implementation which handles CoreAudio internally. Demonstrates the
`CoreAudioConfig` pattern for platform-specific configuration.

---

## 8. Common Pitfalls and How to Avoid Them

### Pitfall 1: Callbacks Crash on Non-Go Threads

**Problem:** CoreAudio fires property listener callbacks on its own internal
threads. Before Go 1.6, this would crash.

**Solution:** Go 1.6+ handles this automatically. The runtime creates a temporary
goroutine for the callback. No special action needed, but keep callbacks fast.

### Pitfall 2: Blocking in Callbacks

**Problem:** Blocking in a CoreAudio callback (e.g., waiting on a mutex, channel
send without select/default) can deadlock the audio system.

**Solution:** Use non-blocking channel sends:

```go
select {
case ch <- event:
default:
    // Drop event rather than block
}
```

### Pitfall 3: Memory Leaks from CFStringRef

**Problem:** CoreAudio returns `CFStringRef` values that must be released by
the caller.

**Solution:** Always call `CFRelease` after converting to Go string:

```go
var cfName C.CFStringRef
// ... get property ...
goName := cfStringToGo(cfName)
C.CFRelease(C.CFTypeRef(cfName))
```

### Pitfall 4: cgo.Handle Leaks

**Problem:** Every `cgo.NewHandle()` must have a corresponding `handle.Delete()`.
Forgetting to delete handles causes memory leaks in the runtime's global map.

**Solution:** Use `defer` where possible. For long-lived handles (property
listeners), ensure `Delete()` is called in the cleanup/close method.

### Pitfall 5: `kAudioObjectPropertyElementMaster` Deprecation

**Problem:** `kAudioObjectPropertyElementMaster` was deprecated in macOS 12 in
favor of `kAudioObjectPropertyElementMain`.

**Solution:** Use a compatibility define:

```c
#ifndef __MAC_12_0
#define kAudioObjectPropertyElementMain kAudioObjectPropertyElementMaster
#endif
```

Or always use `kAudioObjectPropertyElementMain` and set the minimum
deployment target to macOS 12+.

### Pitfall 6: `//export` and C Definitions in Same File

**Problem:** If a Go file uses `//export`, the cgo preamble in that file cannot
contain C function **definitions** (only declarations). This causes
"multiple definition" linker errors.

**Solution:** Split into two files:
- `callbacks.go` -- contains `//export` functions (no C definitions in preamble)
- `gateway.go` -- contains C gateway functions (no `//export` in this file)

Or use the `static inline` trick for simple gateway functions:

```go
/*
static inline void myGateway(void* data) {
    extern void goCallback(void*);
    goCallback(data);
}
*/
import "C"
```

### Pitfall 7: Struct Alignment Issues

**Problem:** Go does not support packed structs. Some C structs may have
alignment that differs from Go's rules.

**Solution:** `AudioObjectPropertyAddress` is safe (all `UInt32` fields).
For complex audio format structs (`AudioStreamBasicDescription`), read fields
individually via accessor functions if alignment issues arise.

### Pitfall 8: cgo Performance Overhead

**Problem:** Each cgo call has ~100-200ns overhead due to goroutine stack
switching. Frequent calls (e.g., per-sample processing) will be too slow.

**Solution:** For audeck, this is not an issue. We are doing:
- Device enumeration (infrequent, one-time)
- Property queries (infrequent, on user interaction)
- Property listeners (event-driven, not per-sample)

None of these are in a hot path. If per-sample audio processing were needed,
the audio callback would stay entirely in C.

### Pitfall 9: Race Between Listener Removal and Callback

**Problem:** After calling `AudioObjectRemovePropertyListener`, a callback may
still be in-flight on another thread.

**Solution:** After removing the listener, do NOT immediately delete the
`cgo.Handle`. Use a small delay or reference counting to ensure no callbacks
are still executing:

```go
func (l *PropertyListener) Close() {
    C.AudioObjectRemovePropertyListener(/* ... */)
    // Brief sync point -- CoreAudio guarantees no new callbacks after remove
    // but in-flight ones may still be executing
    runtime.Gosched()
    l.handle.Delete()
    close(l.ch)
}
```

### Pitfall 10: `net/http` + cgo Segfaults on macOS

**Problem:** Known issue where importing both `net/http` and a cgo package that
links macOS frameworks can cause segfaults due to clashing system call
handling.

**Solution:** Ensure you use the latest Go version. This has been largely fixed
in Go 1.21+. If issues persist, isolate cgo calls in a separate process.

---

## 9. Recommended Architecture for audeck

Based on the research above, here is the recommended layered architecture:

```
+---------------------------------------------------+
|  Bubble Tea TUI Layer                              |
|  (tea.Model, tea.Cmd, tea.Msg)                    |
+---------------------------------------------------+
        |  tea.Msg (DeviceChanged, VolumeChanged)
        |
+---------------------------------------------------+
|  Audio Manager (Pure Go)                           |
|  - Orchestrates device queries                     |
|  - Manages property listeners                      |
|  - Exposes channel-based event API                 |
|  - Converts C types to Go domain types             |
+---------------------------------------------------+
        |  Go function calls
        |
+---------------------------------------------------+
|  CoreAudio Bindings (cgo Layer)                    |
|  - Thin wrappers around AudioObject* functions     |
|  - C gateway functions for callbacks               |
|  - CFString conversion utilities                   |
|  - cgo.Handle management for listener context      |
+---------------------------------------------------+
        |  cgo calls
        |
+---------------------------------------------------+
|  macOS CoreAudio Framework                         |
|  (AudioHardware.h, AudioObject API)               |
+---------------------------------------------------+
```

### File Organization

```
internal/
  coreaudio/
    coreaudio.go          # Package doc, build tags, constants
    device.go             # Device type, enumeration, properties
    listener.go           # PropertyListener with cgo.Handle
    listener_gateway.go   # C gateway functions (static inline)
    listener_export.go    # //export Go callback functions
    cfstring.go           # CFString <-> Go string utilities
    errors.go             # OSStatus error codes and conversion
```

### Integration with Bubble Tea

```go
// In the TUI layer:
func listenForAudioChanges(mgr *coreaudio.Manager) tea.Cmd {
    return func() tea.Msg {
        event := <-mgr.Events()
        return AudioChangedMsg{Event: event}
    }
}
```

This keeps the cgo layer completely decoupled from the TUI and follows
Bubble Tea's command/message architecture.

---

## Appendix A: Device Enumeration Edge Cases

When enumerating audio devices, several edge cases can cause crashes or
incorrect behavior. This section documents them for the device enumeration
implementation.

### A.1 Aggregate Devices

Aggregate devices combine multiple physical devices into one virtual device.
They require special handling:

- **Detection**: Check `kAudioDevicePropertyTransportType` for
  `kAudioDeviceTransportTypeAggregate`. Alternatively, check
  `kAudioObjectPropertyClass` for `kAudioAggregateDeviceClassID` ('aagg').
- **Crash risk**: Aggregate devices can have ZERO inputs AND zero outputs.
  Code that assumes every device has at least one input or output will crash.
  Always check stream count before querying channels.
- **Sub-devices**: Use `kAudioAggregateDevicePropertyActiveSubDeviceList` to
  get the list of sub-devices bound to the aggregate.
- **Volatility**: Users or other apps can modify aggregate device composition
  at any time. Always re-query rather than cache.
- **Private aggregates**: Created with `kAudioAggregateDeviceIsPrivateKey`.
  These disappear when the creating process exits. They show up in device
  enumeration but may vanish at any moment.
- **No nesting**: Aggregate devices cannot contain other aggregate devices.
  Attempting to create nested aggregates fails silently or causes errors.

```go
func isAggregateDevice(deviceID C.AudioDeviceID) bool {
    addr := C.AudioObjectPropertyAddress{
        mSelector: C.kAudioDevicePropertyTransportType,
        mScope:    C.kAudioObjectPropertyScopeGlobal,
        mElement:  C.kAudioObjectPropertyElementMain,
    }
    var transportType C.UInt32
    size := C.UInt32(unsafe.Sizeof(transportType))
    status := C.AudioObjectGetPropertyData(deviceID, &addr, 0, nil, &size, unsafe.Pointer(&transportType))
    if status != 0 {
        return false
    }
    return transportType == C.kAudioDeviceTransportTypeAggregate
}
```

### A.2 Bluetooth Devices

Bluetooth audio devices have unique behaviors:

- **Transport type**: `kAudioDeviceTransportTypeBluetooth`
- **Disappearing devices**: Bluetooth devices appear/disappear as they
  connect/disconnect. Listen to `kAudioHardwarePropertyDevices` on the
  system object to detect this.
- **Profile switching**: Bluetooth devices may switch between A2DP (high
  quality output only) and HFP/HSP (lower quality bidirectional). This can
  cause the device to disappear and reappear with different capabilities
  (different number of channels, different sample rates).
- **Latency**: Bluetooth devices have significantly higher latency. Check
  `kAudioDevicePropertyLatency` and `kAudioDevicePropertySafetyOffset`.
- **Device alive**: Monitor `kAudioDevicePropertyDeviceIsAlive` per-device to
  detect disconnection of a specific Bluetooth device:

```c
AudioObjectPropertyAddress alive_address = {
    kAudioDevicePropertyDeviceIsAlive,
    kAudioObjectPropertyScopeGlobal,
    kAudioObjectPropertyElementMain
};
AudioObjectAddPropertyListener(deviceID, &alive_address, callback, clientData);
```

### A.3 Virtual Audio Devices

Virtual devices (BlackHole, Loopback, Soundflower, etc.):

- **Transport type**: `kAudioDeviceTransportTypeVirtual`
- **Hidden devices**: Some virtual devices set `kAudioDevicePropertyIsHidden`
  to true. Decide whether to show these in the UI or filter them out.
- **No physical controls**: Virtual devices typically do not support hardware
  volume or mute. Check `kAudioObjectPropertyControlList` before attempting
  to read volume/mute controls.

### A.4 HDMI / DisplayPort Audio

- **Transport types**: `kAudioDeviceTransportTypeHDMI`,
  `kAudioDeviceTransportTypeDisplayPort`
- **Output only**: These are always output devices. Querying input streams
  returns zero.
- **Hot-plug**: HDMI/DisplayPort devices appear/disappear when monitors are
  connected/disconnected. Same listener pattern as Bluetooth.

### A.5 USB Audio Devices

- **Transport type**: `kAudioDeviceTransportTypeUSB`
- **Hot-plug**: Like Bluetooth, USB devices can appear/disappear at any time.
- **macOS 26 note**: Apple changed the IO Registry for USB audio devices.
  `usbaudiod` replaced `AppleUSBAudioEngine` and no longer includes
  `kIOAudioEngineGlobalUniqueIDKey`. This may affect UID-based lookups on
  newer macOS versions.

### A.6 Default Device Fallback

When the current default device is disconnected:

- macOS automatically switches the default device to another available device
  (usually the built-in speakers/mic).
- The `kAudioHardwarePropertyDefaultOutputDevice` /
  `kAudioHardwarePropertyDefaultInputDevice` listener fires.
- The old device ID becomes invalid. Querying properties on it returns
  `kAudioHardwareBadDeviceError` ('!dev').
- **Always handle the case where a cached device ID is no longer valid.**

### A.7 Device Enumeration Race Conditions

The device list can change between the time you query the size and the time
you read the data:

```go
// Step 1: Get size
AudioObjectGetPropertyDataSize(systemObj, &addr, 0, nil, &dataSize)

// *** A device could be added/removed here ***

// Step 2: Read data
AudioObjectGetPropertyData(systemObj, &addr, 0, nil, &dataSize, devices)
```

**Mitigation:** If `GetPropertyData` returns `kAudioHardwareBadPropertySizeError`,
retry the entire size+read sequence. The `dataSize` parameter is in/out --
CoreAudio updates it to the actual bytes written, so always check it after
the call.

### A.8 Transport Type Summary

| Transport Type | Constant | Hot-Plug | Input | Output | Notes |
|---|---|---|---|---|---|
| Built-in | `kAudioDeviceTransportTypeBuiltIn` | No | Yes | Yes | Always present |
| USB | `kAudioDeviceTransportTypeUSB` | Yes | Maybe | Maybe | Check streams |
| Bluetooth | `kAudioDeviceTransportTypeBluetooth` | Yes | Maybe | Yes | Profile-dependent |
| HDMI | `kAudioDeviceTransportTypeHDMI` | Yes | No | Yes | Display-dependent |
| DisplayPort | `kAudioDeviceTransportTypeDisplayPort` | Yes | No | Yes | Display-dependent |
| Aggregate | `kAudioDeviceTransportTypeAggregate` | No* | Maybe | Maybe | User-created |
| Virtual | `kAudioDeviceTransportTypeVirtual` | No* | Maybe | Maybe | Software driver |
| Thunderbolt | `kAudioDeviceTransportTypeThunderbolt` | Yes | Maybe | Maybe | Pro audio |
| AirPlay | `kAudioDeviceTransportTypeAirPlay` | Yes | No | Yes | Network |
| FireWire | `kAudioDeviceTransportTypeFireWire` | Yes | Maybe | Maybe | Legacy |
| PCI | `kAudioDeviceTransportTypePCI` | No | Maybe | Maybe | Internal cards |

*Aggregate/Virtual devices can appear/disappear if created programmatically with
the private flag, or if the creating app exits.

---

## Sources

- [Go Wiki: cgo](https://go.dev/wiki/cgo) -- Official cgo reference
- [Go Wiki: LockOSThread](https://go.dev/wiki/LockOSThread) -- Thread locking patterns
- [Eli Bendersky: Passing callbacks and pointers to Cgo](https://eli.thegreenplace.net/2019/passing-callbacks-and-pointers-to-cgo/) -- Detailed callback walkthrough
- [runtime/cgo.Handle](https://pkg.go.dev/runtime/cgo) -- Go 1.17+ handle facility
- [golang.design: cgo Handle research](https://golang.design/research/cgo-handle/) -- Deep dive on cgo.Handle design
- [moriyoshi/go-coreaudio](https://github.com/moriyoshi/go-coreaudio) -- Go CoreAudio bindings
- [progrium/darwinkit](https://pkg.go.dev/github.com/progrium/darwinkit/macos/coreaudio) -- macOS framework bindings for Go
- [deweller/switchaudio-osx](https://github.com/deweller/switchaudio-osx) -- C reference for audio device switching
- [ExistentialAudio/BlackHole](https://github.com/ExistentialAudio/BlackHole) -- C reference for CoreAudio driver patterns
- [rnine/SimplyCoreAudio](https://github.com/rnine/SimplyCoreAudio) -- Swift CoreAudio wrapper (design reference)
- [Apple: Core Audio Overview](https://developer.apple.com/library/archive/documentation/MusicAudio/Conceptual/CoreAudioOverview/CoreAudioEssentials/CoreAudioEssentials.html) -- Official Apple documentation
- [gen2brain/malgo](https://pkg.go.dev/github.com/gen2brain/malgo) -- miniaudio Go bindings with CoreAudio support
