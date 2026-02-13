//go:build darwin

package coreaudio

/*
#include <CoreAudio/CoreAudio.h>
#include <stdint.h>

// Forward declaration of the Go callback (defined in listener_export.go).
extern void goAudioPropertyCallback(
	uintptr_t handle,
	AudioObjectID objectID,
	AudioObjectPropertySelector selector,
	AudioObjectPropertyScope scope,
	AudioObjectPropertyElement element
);

// C gateway function that CoreAudio calls on its internal threads.
// It iterates the changed addresses and calls into Go for each one.
static OSStatus audioPropertyListenerGateway(
	AudioObjectID                    inObjectID,
	UInt32                           inNumberAddresses,
	const AudioObjectPropertyAddress inAddresses[],
	void*                            inClientData
) {
	for (UInt32 i = 0; i < inNumberAddresses; i++) {
		goAudioPropertyCallback(
			(uintptr_t)inClientData,
			inObjectID,
			inAddresses[i].mSelector,
			inAddresses[i].mScope,
			inAddresses[i].mElement
		);
	}
	return 0;
}

// Wrapper to add a property listener using the gateway function.
// Accepts clientData as uintptr_t to avoid Go unsafe.Pointer conversion.
static OSStatus addPropertyListener(
	AudioObjectID id,
	AudioObjectPropertyAddress *addr,
	uintptr_t clientData
) {
	return AudioObjectAddPropertyListener(id, addr, audioPropertyListenerGateway, (void*)clientData);
}

// Wrapper to remove a property listener using the gateway function.
// Accepts clientData as uintptr_t to avoid Go unsafe.Pointer conversion.
static OSStatus removePropertyListener(
	AudioObjectID id,
	AudioObjectPropertyAddress *addr,
	uintptr_t clientData
) {
	return AudioObjectRemovePropertyListener(id, addr, audioPropertyListenerGateway, (void*)clientData);
}
*/
import "C"
import (
	"runtime"
	"runtime/cgo"
	"sync"
)

// PropertyEvent is emitted when a watched CoreAudio property changes.
type PropertyEvent struct {
	ObjectID AudioObjectID
	Address  PropertyAddress
}

// Listener receives CoreAudio property change notifications via a channel.
// It manages one cgo.Handle and any number of property watch registrations.
type Listener struct {
	ch      chan PropertyEvent
	handle  cgo.Handle
	mu      sync.Mutex
	watches []watchEntry
}

type watchEntry struct {
	objectID AudioObjectID
	addr     PropertyAddress
}

// NewListener creates a Listener that sends property change events to a
// buffered channel of the given size. The caller must call Close when done.
func NewListener(bufSize int) *Listener {
	l := &Listener{
		ch: make(chan PropertyEvent, bufSize),
	}
	l.handle = cgo.NewHandle(l)
	return l
}

// Events returns the read-only channel that receives property change events.
func (l *Listener) Events() <-chan PropertyEvent {
	return l.ch
}

// Watch registers a CoreAudio property listener for the given object and
// property address. Events will be delivered on the Events() channel.
func (l *Listener) Watch(objectID AudioObjectID, addr PropertyAddress) error {
	cAddr := toCAddress(addr)
	status := C.addPropertyListener(
		C.AudioObjectID(objectID),
		&cAddr,
		C.uintptr_t(l.handle),
	)
	if err := checkStatus(OSStatus(status), "AddPropertyListener"); err != nil {
		return err
	}
	l.mu.Lock()
	l.watches = append(l.watches, watchEntry{objectID: objectID, addr: addr})
	l.mu.Unlock()
	return nil
}

// Unwatch removes a previously registered property listener.
func (l *Listener) Unwatch(objectID AudioObjectID, addr PropertyAddress) error {
	cAddr := toCAddress(addr)
	status := C.removePropertyListener(
		C.AudioObjectID(objectID),
		&cAddr,
		C.uintptr_t(l.handle),
	)
	if err := checkStatus(OSStatus(status), "RemovePropertyListener"); err != nil {
		return err
	}
	l.mu.Lock()
	for i, w := range l.watches {
		if w.objectID == objectID && w.addr == addr {
			l.watches = append(l.watches[:i], l.watches[i+1:]...)
			break
		}
	}
	l.mu.Unlock()
	return nil
}

// Close removes all registered property listeners, frees the cgo handle,
// and closes the events channel. After Close returns, no more events will
// be delivered.
func (l *Listener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, w := range l.watches {
		cAddr := toCAddress(w.addr)
		C.removePropertyListener(
			C.AudioObjectID(w.objectID),
			&cAddr,
			C.uintptr_t(l.handle),
		)
	}
	l.watches = nil
	// Give any in-flight callbacks a chance to complete before deleting
	// the handle. CoreAudio guarantees no new callbacks after remove, but
	// an in-flight one may still be executing.
	runtime.Gosched()
	l.handle.Delete()
	close(l.ch)
	return nil
}
