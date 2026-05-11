// Package okiffsdk provides Go bindings for the Okiff MQTT SDK.
// It wraps a prebuilt static library (libokiff_sdk.a) via CGo.
package okiffsdk

/*
#cgo CFLAGS: -I.
#cgo linux,amd64 LDFLAGS: -L${SRCDIR}/lib/linux_amd64
#cgo linux,arm64 LDFLAGS: -L${SRCDIR}/lib/linux_arm64
#cgo darwin,amd64 LDFLAGS: -L${SRCDIR}/lib/darwin_amd64
#cgo darwin,arm64 LDFLAGS: -L${SRCDIR}/lib/darwin_arm64
#cgo LDFLAGS: -lokiff_sdk -lpaho-mqtt3c -lpthread -lstdc++
#include "okiff_sdk_capi.h"
#include <stdlib.h>

// Forward declarations for Go-exported callbacks (defined in callbacks.go)
extern void goMessageCallback(const char* topic, const char* payload, void* userdata);
extern void goConnectionCallback(int connected, int rc, void* userdata);
*/
import "C"

import (
	"errors"
	"sync"
	"unsafe"
)

// MessageHandler is called when a message arrives on a subscribed topic.
type MessageHandler func(topic, payload string)

// ConnectionHandler is called on every connection state change.
// connected: true = connected, false = disconnected or lost.
// rc: 0 = clean disconnect, -1 = unexpected loss.
type ConnectionHandler func(connected bool, rc int)

// SDK is the main client handle. Create one with New().
type SDK struct {
	handle C.OkiffHandle

	mu                sync.Mutex
	messageHandler    MessageHandler
	connectionHandler ConnectionHandler
	clientID          string
	brokerHost        string
	protocol          string
	username          string
	password          string
}

// registry maps raw C pointer (used as key) back to the Go SDK instance
// so CGo callbacks can locate the correct SDK instance.
var (
	registryMu sync.RWMutex
	registry   = make(map[uintptr]*SDK)
)

func register(s *SDK) {
	registryMu.Lock()
	registry[uintptr(s.handle)] = s
	registryMu.Unlock()
}

func unregister(s *SDK) {
	registryMu.Lock()
	delete(registry, uintptr(s.handle))
	registryMu.Unlock()
}

func lookup(handle unsafe.Pointer) *SDK {
	registryMu.RLock()
	s := registry[uintptr(handle)]
	registryMu.RUnlock()
	return s
}

// New allocates and returns a new SDK instance.
func New() *SDK {
	s := &SDK{
		handle: C.okiff_create(),
	}
	register(s)
	return s
}

// Init initializes the MQTT client. Must be called before Connect.
func (s *SDK) Init(clientId, brokerHost, protocol, username, password string) error {
	cClientId := C.CString(clientId)
	cBrokerHost := C.CString(brokerHost)
	cProtocol := C.CString(protocol)
	cUsername := C.CString(username)
	cPassword := C.CString(password)

	defer C.free(unsafe.Pointer(cClientId))
	defer C.free(unsafe.Pointer(cBrokerHost))
	defer C.free(unsafe.Pointer(cProtocol))
	defer C.free(unsafe.Pointer(cUsername))
	defer C.free(unsafe.Pointer(cPassword))

	rc := C.okiff_init(s.handle, cClientId, cBrokerHost, cProtocol, cUsername, cPassword)
	if rc != 0 {
		return errors.New("okiffsdk: Init failed — internal client creation error")
	}

	s.mu.Lock()
	s.clientID = clientId
	s.brokerHost = brokerHost
	s.protocol = protocol
	s.username = username
	s.password = password
	s.mu.Unlock()

	return nil
}

// Connect establishes the connection to the broker.
// Returns true on success, false on failure.
func (s *SDK) Connect() bool {
	return C.okiff_connect(s.handle) == 1
}

// Disconnect gracefully disconnects from the broker.
func (s *SDK) Disconnect() {
	C.okiff_disconnect(s.handle)
}

// Stop stops all activity and releases all resources.
// Must be called as the final cleanup step, after Disconnect.
func (s *SDK) Stop() {
	unregister(s)
	C.okiff_stop(s.handle)
}

// Destroy frees the SDK handle. Call after Stop.
func (s *SDK) Destroy() {
	C.okiff_destroy(s.handle)
	s.handle = nil
}

// IsConnected returns the current connection state.
func (s *SDK) IsConnected() bool {
	return C.okiff_is_connected(s.handle) == 1
}

// Publish sends a message to a topic.
// qos: 0 = at most once, 1 = at least once, 2 = exactly once.
// retained: broker stores the message for future new subscribers.
func (s *SDK) Publish(topic, payload string, qos int, retained bool) {
	cTopic := C.CString(topic)
	cPayload := C.CString(payload)
	defer C.free(unsafe.Pointer(cTopic))
	defer C.free(unsafe.Pointer(cPayload))

	cRetained := C.int(0)
	if retained {
		cRetained = 1
	}

	C.okiff_publish(s.handle, cTopic, cPayload, C.int(qos), cRetained)
}

// Subscribe registers a subscription on the given topic.
// Returns true on success, false on failure.
func (s *SDK) Subscribe(topic string, qos int) bool {
	cTopic := C.CString(topic)
	defer C.free(unsafe.Pointer(cTopic))
	return C.okiff_subscribe(s.handle, cTopic, C.int(qos)) == 1
}

// Unsubscribe removes a topic subscription.
func (s *SDK) Unsubscribe(topic string) {
	cTopic := C.CString(topic)
	defer C.free(unsafe.Pointer(cTopic))
	C.okiff_unsubscribe(s.handle, cTopic)
}

// OnMessage registers a callback invoked when a message arrives on any subscribed topic.
// Replaces any previously registered handler.
func (s *SDK) OnMessage(h MessageHandler) {
	s.mu.Lock()
	s.messageHandler = h
	s.mu.Unlock()

	C.okiff_set_message_callback(
		s.handle,
		C.OkiffMessageCallback(C.goMessageCallback),
		unsafe.Pointer(s.handle),
	)
}

// OnConnection registers a callback invoked on every connection state change.
// Replaces any previously registered handler.
func (s *SDK) OnConnection(h ConnectionHandler) {
	s.mu.Lock()
	s.connectionHandler = h
	s.mu.Unlock()

	C.okiff_set_connection_callback(
		s.handle,
		C.OkiffConnectionCallback(C.goConnectionCallback),
		unsafe.Pointer(s.handle),
	)
}
