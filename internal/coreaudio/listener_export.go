//go:build darwin

package coreaudio

// This file contains ONLY the //export callback function.
// cgo requires that files with //export must not have C function definitions
// in the preamble -- only declarations and #includes are allowed.

/*
#include <CoreAudio/CoreAudio.h>
#include <stdint.h>
*/
import "C"
import "runtime/cgo"

//export goAudioPropertyCallback
func goAudioPropertyCallback(
	handle C.uintptr_t,
	objectID C.AudioObjectID,
	selector C.AudioObjectPropertySelector,
	scope C.AudioObjectPropertyScope,
	element C.AudioObjectPropertyElement,
) {
	h := cgo.Handle(handle)
	l := h.Value().(*Listener)
	evt := PropertyEvent{
		ObjectID: AudioObjectID(objectID),
		Address: PropertyAddress{
			Selector: PropertySelector(selector),
			Scope:    PropertyScope(scope),
			Element:  PropertyElement(element),
		},
	}
	// Non-blocking send to prevent stalling the CoreAudio thread.
	select {
	case l.ch <- evt:
	default:
	}
}
