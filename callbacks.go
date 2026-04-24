package okiffsdk

/*
#include "okiff_sdk_capi.h"
*/
import "C"
import "unsafe"

// goMessageCallback is exported to C and called by the SDK from the Paho thread.
// It looks up the Go SDK instance and dispatches to the registered MessageHandler.
//
//export goMessageCallback
func goMessageCallback(topic *C.char, payload *C.char, userdata unsafe.Pointer) {
	s := lookup(userdata)
	if s == nil {
		return
	}

	s.mu.Lock()
	h := s.messageHandler
	s.mu.Unlock()

	if h != nil {
		h(C.GoString(topic), C.GoString(payload))
	}
}

// goConnectionCallback is exported to C and called by the SDK on connection state changes.
//
//export goConnectionCallback
func goConnectionCallback(connected C.int, rc C.int, userdata unsafe.Pointer) {
	s := lookup(userdata)
	if s == nil {
		return
	}

	s.mu.Lock()
	h := s.connectionHandler
	s.mu.Unlock()

	if h != nil {
		h(connected == 1, int(rc))
	}
}
