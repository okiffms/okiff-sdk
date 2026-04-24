# okiff-sdk

Go SDK for connecting to and communicating over MQTT brokers.
Built as a precompiled static library with CGo bindings for performance and ease of deployment.

---

## Requirements

- Go 1.23 or later
- Linux x86-64
- GCC (CGo build toolchain)
- `libpaho-mqtt3c` installed system-wide

Install on Debian/Ubuntu:

```bash
sudo apt install gcc libpaho-mqtt3c-dev
```

Install on Manjaro/Arch:

```bash
sudo pacman -S gcc paho-mqtt-c
```

---

## Installation

Inside an existing Go module:

```bash
go get github.com/okiffms/okiff-sdk@latest
```

---

## Quick Start

```go
package main

import (
    "fmt"
    "time"

    okiffsdk "github.com/okiffms/okiff-sdk"
)

func main() {
    sdk := okiffsdk.New()
    defer sdk.Destroy()

    // Callbacks
    sdk.OnConnection(func(connected bool, rc int) {
        fmt.Printf("[onConnection] connected=%v rc=%d\n", connected, rc)
    })

    sdk.OnMessage(func(topic, payload string) {
        fmt.Printf("[onMessage] topic=%s payload=%s\n", topic, payload)
    })

    // Initialize and connect
    if err := sdk.Init(
        "my_client",
        "broker.example.com:1883",
        "tcp",
        "username",
        "password",
    ); err != nil {
        fmt.Println("Init error:", err)
        return
    }

    ok := sdk.Connect()
    fmt.Println("Connect() =>", ok)

    // Publish and subscribe
    if ok {
        sdk.Subscribe("my/topic", 0)
        time.AfterFunc(500*time.Millisecond, func() {
            sdk.Publish("my/topic", "hello", 0, false)
        })
    }

    // Cleanup
    time.Sleep(5 * time.Second)
    sdk.Disconnect()
    sdk.Stop()
}
```

## Paho Compatibility

If your application expects `github.com/eclipse/paho.mqtt.golang`.Client,
use the compatibility adapter instead of passing `*SDK` directly:

```go
package main

import (
    mqtt "github.com/eclipse/paho.mqtt.golang"
    okiffsdk "github.com/okiffms/okiff-sdk"
)

func main() {
    client, err := okiffsdk.Init(
        "my_client",
        "broker.example.com:1883",
        "tcp",
        "username",
        "password",
    )
    if err != nil {
        panic(err)
    }

    var mqttClient mqtt.Client = client
    _ = mqttClient
}
```

If you already have an initialized `*SDK`, wrap it with `sdk.AsPahoClient()`.

---

## API Reference

### `New() *SDK`

Allocates and returns a new SDK instance.

---

### `Init(clientId, brokerHost, protocol, username, password string) error`

Initializes the SDK. Must be called before `Connect`.

| Parameter | Type | Description |
|---|---|---|
| `clientId` | `string` | Unique identifier for this client |
| `brokerHost` | `string` | Broker address including port, e.g. `broker.example.com:1883` |
| `protocol` | `string` | Transport protocol: `"tcp"` or `"ssl"` |
| `username` | `string` | Authentication username |
| `password` | `string` | Authentication password |

**Returns:** `nil` on success, `error` if the internal client cannot be created.

---

### `Connect() bool`

Connects to the broker.

**Returns:** `true` on success, `false` on failure.

---

### `Disconnect()`

Gracefully disconnects from the broker.

---

### `Publish(topic, payload string, qos int, retained bool)`

Publishes a message to a topic.

| Parameter | Type | Description |
|---|---|---|
| `topic` | `string` | MQTT topic string |
| `payload` | `string` | Message body |
| `qos` | `int` | Quality of Service: `0`, `1`, or `2` |
| `retained` | `bool` | Whether the broker retains the message |

---

### `Subscribe(topic string, qos int) bool`

Subscribes to a topic.

| Parameter | Type | Description |
|---|---|---|
| `topic` | `string` | MQTT topic. Wildcards: `+` (single level), `#` (multi-level) |
| `qos` | `int` | Maximum QoS level for received messages |

**Returns:** `true` on success, `false` on failure.

---

### `Unsubscribe(topic string)`

Removes a topic subscription.

| Parameter | Type | Description |
|---|---|---|
| `topic` | `string` | Exact topic string used in `Subscribe` |

---

### `OnMessage(handler func(topic, payload string))`

Registers a callback invoked when a message arrives on any subscribed topic.

```go
sdk.OnMessage(func(topic, payload string) {
    // topic   — topic on which the message was received
    // payload — message body
})
```

---

### `OnConnection(handler func(connected bool, rc int))`

Registers a callback invoked on every connection state change.

```go
sdk.OnConnection(func(connected bool, rc int) {
    // connected — true = connected, false = disconnected or lost
    // rc        — 0 = clean disconnect, -1 = unexpected loss
})
```

| Event | `connected` | `rc` |
|---|---|---|
| `Connect()` succeeds | `true` | `0` |
| `Disconnect()` called | `false` | `0` |
| Broker drops connection | `false` | `-1` |

---

### `IsConnected() bool`

Returns the current connection state.

---

### `Stop()`

Stops all activity and releases all resources. Must be called after `Disconnect`.

---

### `Destroy()`

Frees the SDK handle. Must be called after `Stop`. Idiomatic usage:

```go
sdk := okiffsdk.New()
defer sdk.Destroy()
```

---

## License

Proprietary — all rights reserved.
