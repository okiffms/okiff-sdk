package okiffsdk

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var _ mqtt.Client = (*PahoClient)(nil)

// PahoClient adapts SDK to the github.com/eclipse/paho.mqtt.golang Client interface.
type PahoClient struct {
	sdk *SDK

	mu             sync.RWMutex
	routes         map[string]mqtt.MessageHandler
	defaultHandler mqtt.MessageHandler
	opts           *mqtt.ClientOptions
}

// Init creates, initializes, and returns a Paho-compatible MQTT client.
func Init(clientID, brokerHost, protocol, username, password string) (mqtt.Client, error) {
	client, err := NewPahoClient(clientID, brokerHost, protocol, username, password)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// NewPahoClient creates, initializes, and returns a Paho-compatible client adapter.
func NewPahoClient(clientID, brokerHost, protocol, username, password string) (*PahoClient, error) {
	sdk := New()
	if err := sdk.Init(clientID, brokerHost, protocol, username, password, false, true, true, true, 5, 1, 30, 10, 10, false, 30, "Enterprise"); err != nil {
		sdk.Destroy()
		return nil, err
	}
	return WrapPahoClient(sdk), nil
}

// WrapPahoClient exposes an initialized SDK as a Paho-compatible client.
func WrapPahoClient(sdk *SDK) *PahoClient {
	client := &PahoClient{
		sdk:    sdk,
		routes: make(map[string]mqtt.MessageHandler),
		opts:   newClientOptionsFromSDK(sdk),
	}

	sdk.OnMessage(client.dispatchMessage)
	sdk.OnConnection(client.dispatchConnection)

	return client
}

// AsPahoClient exposes the SDK instance as a Paho-compatible client.
func (s *SDK) AsPahoClient() *PahoClient {
	return WrapPahoClient(s)
}

// SDK returns the underlying SDK handle.
func (c *PahoClient) SDK() *SDK {
	return c.sdk
}

// Stop stops the underlying SDK.
func (c *PahoClient) Stop() {
	c.sdk.Stop()
}

// Destroy destroys the underlying SDK handle.
func (c *PahoClient) Destroy() {
	c.sdk.Destroy()
}

// Close releases the underlying SDK resources.
func (c *PahoClient) Close() {
	c.sdk.Stop()
	c.sdk.Destroy()
}

func (c *PahoClient) IsConnected() bool {
	return c.sdk.IsConnected()
}

func (c *PahoClient) IsConnectionOpen() bool {
	return c.sdk.IsConnected()
}

func (c *PahoClient) Connect() mqtt.Token {
	if ok := c.sdk.Connect(); !ok {
		return newCompatToken(errors.New("okiffsdk: connect failed"))
	}
	return newCompatToken(nil)
}

func (c *PahoClient) Disconnect(quiesce uint) {
	_ = quiesce
	c.sdk.Disconnect()
}

func (c *PahoClient) Publish(topic string, qos byte, retained bool, payload interface{}) mqtt.Token {
	body, err := payloadToString(payload)
	if err != nil {
		return newCompatToken(err)
	}

	c.sdk.Publish(topic, body, int(qos), retained)
	return newCompatToken(nil)
}

func (c *PahoClient) Subscribe(topic string, qos byte, callback mqtt.MessageHandler) mqtt.Token {
	if callback != nil {
		c.AddRoute(topic, callback)
	}

	if ok := c.sdk.Subscribe(topic, int(qos)); !ok {
		return newCompatToken(fmt.Errorf("okiffsdk: subscribe failed for topic %q", topic))
	}
	return newCompatToken(nil)
}

func (c *PahoClient) SubscribeMultiple(filters map[string]byte, callback mqtt.MessageHandler) mqtt.Token {
	for topic, qos := range filters {
		if callback != nil {
			c.AddRoute(topic, callback)
		}

		if ok := c.sdk.Subscribe(topic, int(qos)); !ok {
			return newCompatToken(fmt.Errorf("okiffsdk: subscribe failed for topic %q", topic))
		}
	}

	return newCompatToken(nil)
}

func (c *PahoClient) Unsubscribe(topics ...string) mqtt.Token {
	for _, topic := range topics {
		c.deleteRoute(topic)
		c.sdk.Unsubscribe(topic)
	}
	return newCompatToken(nil)
}

func (c *PahoClient) AddRoute(topic string, callback mqtt.MessageHandler) {
	c.mu.Lock()
	c.routes[topic] = callback
	c.mu.Unlock()
}

func (c *PahoClient) OptionsReader() mqtt.ClientOptionsReader {
	return mqtt.NewOptionsReader(c.opts)
}

func (c *PahoClient) dispatchConnection(connected bool, rc int) {
	c.mu.RLock()
	onConnect := c.opts.OnConnect
	onConnectionLost := c.opts.OnConnectionLost
	c.mu.RUnlock()

	switch {
	case connected && onConnect != nil:
		onConnect(c)
	case !connected && rc != 0 && onConnectionLost != nil:
		onConnectionLost(c, fmt.Errorf("okiffsdk: connection lost (rc=%d)", rc))
	}
}

func (c *PahoClient) dispatchMessage(topic, payload string) {
	message := &compatMessage{
		topic:   topic,
		payload: []byte(payload),
	}

	c.mu.RLock()
	var handlers []mqtt.MessageHandler
	for route, handler := range c.routes {
		if handler != nil && routeMatchesTopic(route, topic) {
			handlers = append(handlers, handler)
		}
	}
	if len(handlers) == 0 && c.defaultHandler != nil {
		handlers = append(handlers, c.defaultHandler)
	}
	c.mu.RUnlock()

	for _, handler := range handlers {
		handler(c, message)
	}
}

func (c *PahoClient) deleteRoute(topic string) {
	c.mu.Lock()
	delete(c.routes, topic)
	c.mu.Unlock()
}

func newClientOptionsFromSDK(s *SDK) *mqtt.ClientOptions {
	s.mu.Lock()
	clientID := s.clientID
	brokerHost := s.brokerHost
	protocol := s.protocol
	username := s.username
	password := s.password
	s.mu.Unlock()

	opts := mqtt.NewClientOptions().
		SetClientID(clientID).
		AddBroker(fmt.Sprintf("%s://%s", protocol, brokerHost)).
		SetUsername(username).
		SetPassword(password)

	return opts
}

type compatToken struct {
	done chan struct{}
	err  error
}

func newCompatToken(err error) *compatToken {
	done := make(chan struct{})
	close(done)
	return &compatToken{
		done: done,
		err:  err,
	}
}

func (t *compatToken) Wait() bool {
	<-t.done
	return true
}

func (t *compatToken) WaitTimeout(d time.Duration) bool {
	select {
	case <-t.done:
		return true
	case <-time.After(d):
		return false
	}
}

func (t *compatToken) Done() <-chan struct{} {
	return t.done
}

func (t *compatToken) Error() error {
	return t.err
}

var _ mqtt.Message = (*compatMessage)(nil)

type compatMessage struct {
	topic   string
	payload []byte
}

func (m *compatMessage) Duplicate() bool { return false }
func (m *compatMessage) Qos() byte       { return 0 }
func (m *compatMessage) Retained() bool  { return false }
func (m *compatMessage) Topic() string   { return m.topic }
func (m *compatMessage) MessageID() uint16 {
	return 0
}
func (m *compatMessage) Payload() []byte { return m.payload }
func (m *compatMessage) Ack()            {}

func payloadToString(payload interface{}) (string, error) {
	switch p := payload.(type) {
	case string:
		return p, nil
	case []byte:
		for _, b := range p {
			if b == 0 {
				return "", errors.New("okiffsdk: binary payloads containing NUL bytes are not supported")
			}
		}
		return string(p), nil
	default:
		return "", fmt.Errorf("okiffsdk: unsupported payload type %T", payload)
	}
}

func routeMatchesTopic(route, topic string) bool {
	return route == topic || routeIncludesTopic(route, topic)
}

func routeIncludesTopic(route, topic string) bool {
	return matchRouteParts(routeSplit(route), strings.Split(topic, "/"))
}

func routeSplit(route string) []string {
	if strings.HasPrefix(route, "$share/") {
		parts := strings.Split(route, "/")
		if len(parts) >= 3 {
			return parts[2:]
		}
	}
	return strings.Split(route, "/")
}

func matchRouteParts(route, topic []string) bool {
	if len(route) == 0 {
		return len(topic) == 0
	}

	if len(topic) == 0 {
		return route[0] == "#"
	}

	if route[0] == "#" {
		return true
	}

	if route[0] == "+" || route[0] == topic[0] {
		return matchRouteParts(route[1:], topic[1:])
	}

	return false
}
