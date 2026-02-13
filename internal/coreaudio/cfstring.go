//go:build darwin

package coreaudio

/*
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
*/
import "C"
import "unsafe"

// cfStringRefToGo converts a CFStringRef to a Go string. The caller is
// responsible for releasing the CFStringRef after this function returns
// (this function does NOT release it).
func cfStringRefToGo(ref C.CFStringRef) string {
	if ref == 0 {
		return ""
	}
	length := C.CFStringGetLength(ref)
	maxSize := C.CFStringGetMaximumSizeForEncoding(length, C.kCFStringEncodingUTF8) + 1
	buf := C.malloc(C.size_t(maxSize))
	if buf == nil {
		return ""
	}
	defer C.free(buf)

	if C.CFStringGetCString(ref, (*C.char)(buf), maxSize, C.kCFStringEncodingUTF8) == 0 {
		return ""
	}
	return C.GoString((*C.char)(buf))
}

// cfStringFromPropertyData reads a CFStringRef from raw property data bytes,
// converts it to a Go string, and releases the CFStringRef.
func cfStringFromPropertyData(data []byte) string {
	if len(data) < int(unsafe.Sizeof(C.CFStringRef(0))) {
		return ""
	}
	ref := *(*C.CFStringRef)(unsafe.Pointer(&data[0]))
	if ref == 0 {
		return ""
	}
	defer C.CFRelease(C.CFTypeRef(ref))
	return cfStringRefToGo(ref)
}
