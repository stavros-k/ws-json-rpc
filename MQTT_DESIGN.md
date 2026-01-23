# MQTT RPC Design

## Overview

This document describes the design for an MQTT-based RPC system that complements the existing WebSocket/HTTP JSON-RPC implementation. The MQTT system leverages the broker's native pub/sub capabilities to provide a lightweight, scalable alternative for clients that prefer or require MQTT connectivity.

## Key Differences from WebSocket

| Aspect | WebSocket | MQTT |
|--------|-----------|------|
| Connection Management | Hub manages all clients | Broker manages connections |
| Subscription Management | Hub tracks subscriptions | Broker handles subscriptions |
| Message Routing | Hub routes to specific clients | Broker routes by topic |
| Request/Response | JSON-RPC 2.0 envelope | Simple JSON (no envelope) |
| Message Size | Larger (verbose RPC) | Minimal (optimized for embedded) |
| Target Devices | Web/Desktop clients | Arduino/embedded devices |
| Complexity | Higher (custom hub) | Lower (leverage broker) |

## Design Principles

### Optimized for Embedded Devices

The MQTT interface is designed specifically for resource-constrained devices like Arduino, ESP8266/ESP32, and similar microcontrollers:

1. **No JSON-RPC Envelope**: Removes overhead of `jsonrpc`, `id`, and other metadata
2. **Method in Topic**: Method name is part of the topic, not payload
3. **Minimal Payload**: Only the essential data, nothing more
4. **Simple Parsing**: Easy to parse with libraries like ArduinoJson

**Payload Size Comparison:**

WebSocket JSON-RPC:
```json
{"jsonrpc":"2.0","id":"123e4567-e89b-12d3-a456-426614174000","method":"setLED","params":{"on":true}}
```
**~100 bytes**

MQTT Simplified:
```json
{"on":true}
```
**~11 bytes** (9x smaller!)

## Architecture

### Components

1. **MQTT Handler** - Handles incoming method calls from MQTT topics and publishes events
2. **MQTT Method Registry** - Separate registry for MQTT methods (different from RPC)
3. **MQTT Event Registry** - Separate registry for MQTT events (different from RPC)

**Key Decision**: MQTT registration is **separate** from WebSocket/HTTP JSON-RPC registration because:
- Different message formats (no JSON-RPC envelope for MQTT)
- Different handler signatures (no request IDs, no RPC response format)
- Simpler, lighter-weight handlers for embedded devices
- Cleaner separation of concerns

### Topic Structure

We use a simple two-tier hierarchical structure:

```
{base_prefix}/
├── call/{method_name}       # Method calls (device → server)
└── event/{event_name}       # Event broadcasts (server → devices)
```

**Default base prefix**: `rpc` (configurable)

#### Examples:
- Method call: `rpc/call/updateSensor`
- Event: `rpc/event/sensorUpdated`
- Event: `rpc/event/temperature`

## Message Flow

### Complete Request/Response Flow

The pattern is beautifully simple: devices call methods, server responds with events.

```
Arduino                        MQTT Broker                    Server
  |                                  |                           |
  | PUBLISH (QoS 2)                  |                           |
  | to: rpc/call/updateSensor        |                           |
  | payload: {"id":"temp1","val":25} |    Forward to Server      |
  |                           -----> | -----------------------> |
  |                                  |                           | Process
  |                                  |                           | Request
  |                                  |  PUBLISH (QoS 2)          | |
  |                                  |  rpc/event/sensorUpdated  | |
  |                                  | <------------------------ | |
  |       Receive Event              |                           | |
  | <------------------------------- |                           | |
  | from: rpc/event/sensorUpdated    |                           | |
  | payload: {"id":"temp1","ok":true}|                           | |
```

**Key Points:**
- Device publishes method call (QoS 2 - exactly once)
- Server processes the request
- Server responds by publishing an event (QoS 2 - exactly once)
- All devices subscribed to that event receive it

## Message Formats

> **Design Philosophy**: Keep payloads minimal for resource-constrained devices like Arduino. No JSON-RPC envelope, no UUIDs unless necessary. Method name is in the topic, not the payload.

### Method Call (Device → Server)

Published to: `rpc/call/{method_name}` with **QoS 2** (exactly once)

**The payload is just the method parameters directly** - no envelope:

```json
{"sensor_id": "temp1", "value": 25.5}
```

For methods with no parameters, payload can be empty `{}` or omitted entirely.

**Examples:**

```
Topic: rpc/call/updateSensor
QoS: 2
Payload: {"sensor_id": "temp1", "value": 25.5}
```

```
Topic: rpc/call/setLED
QoS: 2
Payload: {"on": true, "brightness": 80}
```

```
Topic: rpc/call/ping
QoS: 2
Payload: {}
```

### Event Message (Server → Devices)

Published to: `rpc/event/{event_name}` with **QoS 2** (exactly once)

**The payload is just the event data directly** - no envelope. Events serve dual purpose:
1. Broadcasting state changes to all devices
2. Responding to method calls (optional)

```json
{"temp": 25.5}
```

**Examples:**

Periodic event (broadcast):
```
Topic: rpc/event/temperature
QoS: 2
Payload: {"temp": 25.5, "unit": "C"}
```

Response to a method call:
```
Topic: rpc/event/sensorUpdated
QoS: 2
Payload: {"sensor_id": "temp1", "value": 25.5, "ok": true}
```

Alert/notification:
```
Topic: rpc/event/alert
QoS: 2
Payload: {"level": "high", "msg": "Temperature exceeded threshold"}
```

## Implementation Details

### Server-Side API

```go
// MQTTHandler handles MQTT-based method calls and event publishing
type MQTTHandler struct {
    client      mqtt.Client
    logger      *slog.Logger
    topicPrefix string
    methods     map[string]MQTTMethod
    events      map[string]struct{}  // Registered event names
    methodsMutex sync.RWMutex
    eventsMutex  sync.RWMutex
}

type MQTTHandlerOptions struct {
    TopicPrefix string            // Default: "rpc"
    // QoS is always 2 (exactly once) for both calls and events
}

// MQTTHandlerFunc is a function that handles an MQTT method call
// Receives the raw parameters (just the payload, no envelope)
// Returns the data to publish as event (if any)
type MQTTHandlerFunc func(ctx context.Context, params json.RawMessage) (any, error)

// TypedMQTTHandlerFunc is a typed version for convenience
type TypedMQTTHandlerFunc[TParams any, TResult any] func(ctx context.Context, params TParams) (TResult, error)

// NewMQTTHandler creates a new MQTT handler
func NewMQTTHandler(client mqtt.Client, logger *slog.Logger, opts MQTTHandlerOptions) *MQTTHandler

// RegisterMethod registers an MQTT method
func RegisterMQTTMethod[TParams any, TResult any](
    h *MQTTHandler,
    methodName string,
    handler TypedMQTTHandlerFunc[TParams, TResult],
) error

// RegisterEvent registers an MQTT event
func RegisterMQTTEvent(h *MQTTHandler, eventName string) error

// Start begins listening for MQTT messages on call topics
// Subscribes to {topicPrefix}/call/+ with QoS 2
func (h *MQTTHandler) Start(ctx context.Context) error

// Stop gracefully shuts down the handler
func (h *MQTTHandler) Stop() error

// PublishEvent publishes an event to MQTT subscribers with QoS 2
// Payload is just the event data (no envelope)
func (h *MQTTHandler) PublishEvent(eventName string, data any) error
```

### Registration

MQTT has **separate** registration from WebSocket/HTTP RPC. This keeps the implementations clean and focused.

```go
// In main.go

// 1. Create and setup WebSocket/HTTP RPC (JSON-RPC 2.0)
hub := rpc.NewHub(logger, generator)
rpc.RegisterMethod(hub, "ping", rpcHandlers.Ping, options)
rpc.RegisterEvent[DataCreated](hub, "data.created", options)
go hub.Run()
mux.HandleFunc("/ws", hub.ServeWS())
mux.HandleFunc("/rpc", hub.ServeHTTP())

// 2. Create and setup MQTT (Simplified protocol)
mqttClient := mqtt.NewClient(mqttOpts)
if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
    log.Fatal(token.Error())
}

mqttHandler := rpc.NewMQTTHandler(mqttClient, logger, rpc.MQTTHandlerOptions{
    TopicPrefix: "rpc",
})

// Register MQTT methods (separate from RPC methods)
rpc.RegisterMQTTMethod(mqttHandler, "updateSensor", mqttHandlers.UpdateSensor)
rpc.RegisterMQTTMethod(mqttHandler, "ping", mqttHandlers.Ping)

// Register MQTT events (separate from RPC events)
rpc.RegisterMQTTEvent(mqttHandler, "sensorUpdated")
rpc.RegisterMQTTEvent(mqttHandler, "temperature")

go mqttHandler.Start(context.Background())

// 3. Bridge between transports if needed
// You can manually publish to both if you want:
go func() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        temp := getAverageTemperature()

        // Publish to WebSocket clients
        hub.PublishEvent(rpc.NewEvent("temperature", Temperature{Temp: temp}))

        // Publish to MQTT devices
        mqttHandler.PublishEvent("temperature", map[string]any{"temp": temp})
    }
}()
```

### Device Responsibilities

Devices using MQTT must:

1. **Connect to MQTT broker** with appropriate credentials and unique client ID
2. **Subscribe to desired events** with QoS 2:
   - `rpc/event/+` (subscribe to all events), OR
   - Specific events like `rpc/event/temperature`, `rpc/event/sensorUpdated`
3. **Publish method calls** to `rpc/call/{method_name}` with QoS 2
   - Payload is just the parameters (no envelope)
4. **Handle incoming events** from subscribed topics
   - Events may be responses to their calls or general broadcasts

### Topic Subscriptions

**Server subscribes to:**
- `rpc/call/+` (all method calls) with QoS 2

**Device subscribes to:**
- `rpc/event/+` (all events) with QoS 2, OR
- Specific events like `rpc/event/temperature`, `rpc/event/sensorUpdated` with QoS 2

### QoS Level

**All messages use QoS 2 (exactly once delivery):**

- **Method calls**: QoS 2 - ensures call is processed exactly once
- **Events**: QoS 2 - ensures event is delivered exactly once

This prevents duplicate processing of commands and duplicate event notifications, which is critical for actuators and state management.

## Configuration

```go
type MQTTConfig struct {
    BrokerURL    string        // e.g., "tcp://localhost:1883"
    ClientID     string        // Server's MQTT client ID
    Username     string        // Optional
    Password     string        // Optional
    TopicPrefix  string        // e.g., "rpc"
    QoS          byte          // Default QoS level (0, 1, or 2)
    CleanSession bool          // Whether to start fresh session
}
```

## Error Handling

Errors are communicated through events, just like successful responses.

### Server-Side Error Handling

1. **Method Not Found**: Logged as warning, no event published
2. **Invalid Parameters**: Server can publish error event
3. **Processing Error**: Server can publish error event

### Error Events

When a method call fails, server publishes an event describing the error:

```
Topic: rpc/event/error
QoS: 2
Payload: {
  "method": "updateSensor",
  "err": "invalid_params",
  "msg": "sensor_id is required"
}
```

Or use method-specific error events:

```
Topic: rpc/event/sensorUpdateFailed
QoS: 2
Payload: {
  "sensor_id": "temp1",
  "err": "out_of_range"
}
```

### Devices Handle Errors

Devices subscribe to error events or check response events for error fields:

```cpp
void handleEvent(const char* eventName, JsonDocument& data) {
    if (strcmp(eventName, "error") == 0) {
        const char* err = data["err"];
        Serial.printf("Error: %s\n", err);
    } else if (strcmp(eventName, "sensorUpdated") == 0) {
        bool ok = data["ok"] | true;  // Default to true if field missing
        if (!ok) {
            const char* err = data["err"];
            Serial.printf("Update failed: %s\n", err);
        }
    }
}
```

## Security Considerations

1. **Authentication**: Use MQTT broker's authentication (username/password or certificates)
2. **Authorization**: Configure broker ACLs to restrict topic access
3. **Topic Isolation**: Use topic prefixes to separate environments (e.g., `prod/rpc`, `dev/rpc`)
4. **Message Size**: Configure max message size on broker to prevent DoS
5. **Rate Limiting**: Use broker rate limiting or implement application-level limits

### Example ACL Rules

```
# Server can publish to events and read method calls
user server_user
topic write rpc/event/#
topic read rpc/call/#

# Devices can publish to method calls and read events
user device_*
topic read rpc/event/#
topic write rpc/call/#
```

## Advantages of MQTT Approach

1. **Scalability**: Broker handles connection management and routing
2. **Flexibility**: Devices can subscribe only to events they need
3. **Resilience**: MQTT's built-in retry and QoS mechanisms
4. **Simplicity**: No JSON-RPC envelope overhead
5. **Minimal Payload Size**: Optimized for resource-constrained devices
6. **Standardization**: Uses standard MQTT protocol
7. **Low Power**: MQTT designed for low-power devices
8. **Multi-server**: Multiple servers can handle methods (load balancing)

## Limitations

1. **No Request Tracking**: No request IDs to correlate specific requests to response events
2. **Broadcast Responses**: Events are broadcast to all subscribers, not targeted to requester
3. **No Client State**: Server doesn't track individual device state or capabilities
4. **Broker Dependency**: Requires external MQTT broker
5. **Optional Responses**: Server may or may not respond to a call with an event

## Future Enhancements

1. **Request Timeout Tracking**: Server tracks pending requests and logs timeouts
2. **Metrics**: Publish method call metrics to monitoring topics
3. **Dead Letter Queue**: Route failed messages to DLQ topic
4. **Batch Operations**: Support multiple operations in single message
5. **Compression**: Optional payload compression for large messages
6. **Priority Queues**: Use different topics for priority levels

## Dual Transport Support

Applications can support both WebSocket/HTTP and MQTT simultaneously:

1. Register methods/events separately for each transport
2. Implement handlers independently (RPC handlers vs MQTT handlers)
3. Manually bridge between transports when needed (publish same event to both)
4. Clients choose connection type based on their needs:
   - Web/Desktop clients → WebSocket with JSON-RPC
   - Embedded devices → MQTT with simplified protocol

## Example Usage

### Server Setup

```go
func main() {
    logger := slog.Default()

    // Setup WebSocket/HTTP RPC
    hub := rpc.NewHub(logger, generator)
    rpc.RegisterMethod(hub, "ping", rpcHandlers.Ping, options)
    go hub.Run()
    mux.HandleFunc("/ws", hub.ServeWS())
    mux.HandleFunc("/rpc", hub.ServeHTTP())

    // Setup MQTT
    mqttOpts := mqtt.NewClientOptions()
    mqttOpts.AddBroker("tcp://localhost:1883")
    mqttOpts.SetClientID("server-001")
    mqttClient := mqtt.NewClient(mqttOpts)

    if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
        log.Fatal(token.Error())
    }

    mqttHandler := rpc.NewMQTTHandler(mqttClient, logger, rpc.MQTTHandlerOptions{
        TopicPrefix: "rpc",
    })

    // Register MQTT methods
    rpc.RegisterMQTTMethod(mqttHandler, "updateSensor", mqttHandlers.UpdateSensor)

    // Register MQTT events
    rpc.RegisterMQTTEvent(mqttHandler, "sensorUpdated")
    rpc.RegisterMQTTEvent(mqttHandler, "temperature")

    go mqttHandler.Start(context.Background())

    // Publish periodic temperature to both transports
    go func() {
        ticker := time.NewTicker(5 * time.Second)
        for range ticker.C {
            temp := readTemperature()

            // Publish to WebSocket clients
            hub.PublishEvent(rpc.NewEvent("temperature", Temperature{Temp: temp}))

            // Publish to MQTT devices
            mqttHandler.PublishEvent("temperature", map[string]any{"temp": temp})
        }
    }()
}
```

### Arduino Example (C++)

```cpp
#include <WiFi.h>
#include <PubSubClient.h>
#include <ArduinoJson.h>

WiFiClient wifiClient;
PubSubClient mqtt(wifiClient);

const char* clientId = "arduino-001";

void setup() {
    Serial.begin(115200);

    // Connect to WiFi
    WiFi.begin("SSID", "password");
    while (WiFi.status() != WL_CONNECTED) delay(500);

    // Connect to MQTT broker
    mqtt.setServer("broker.local", 1883);
    mqtt.setCallback(messageCallback);

    while (!mqtt.connected()) {
        if (mqtt.connect(clientId)) {
            Serial.println("MQTT connected");

            // Subscribe to all events with QoS 2
            mqtt.subscribe("rpc/event/+", 2);

            // Or subscribe to specific events
            // mqtt.subscribe("rpc/event/sensorUpdated", 2);
            // mqtt.subscribe("rpc/event/temperature", 2);
        }
    }
}

void messageCallback(char* topic, byte* payload, unsigned int length) {
    StaticJsonDocument<200> doc;
    deserializeJson(doc, payload, length);

    if (strncmp(topic, "rpc/event/", 10) == 0) {
        handleEvent(topic + 10, doc);  // Skip "rpc/event/"
    }
}

void handleEvent(const char* eventName, JsonDocument& data) {
    if (strcmp(eventName, "temperature") == 0) {
        float temp = data["temp"];
        Serial.printf("Temperature: %.1f°C\n", temp);
    }
    else if (strcmp(eventName, "sensorUpdated") == 0) {
        const char* sensorId = data["sensor_id"];
        bool ok = data["ok"] | true;  // Default to true
        if (ok) {
            Serial.printf("Sensor %s updated successfully\n", sensorId);
        } else {
            const char* err = data["err"];
            Serial.printf("Sensor %s update failed: %s\n", sensorId, err);
        }
    }
    else if (strcmp(eventName, "error") == 0) {
        const char* method = data["method"];
        const char* err = data["err"];
        Serial.printf("Error on %s: %s\n", method, err);
    }
}

// Call a method with QoS 2
void callUpdateSensor(const char* sensorId, float value) {
    StaticJsonDocument<64> doc;
    doc["sensor_id"] = sensorId;
    doc["value"] = value;

    char buffer[64];
    serializeJson(doc, buffer);

    // QoS 2 = exactly once
    mqtt.publish("rpc/call/updateSensor", buffer, 2);
}

// Call method with no params
void callPing() {
    mqtt.publish("rpc/call/ping", "{}", 2);
}

void loop() {
    mqtt.loop();

    // Example: send sensor reading every 5 seconds
    static unsigned long lastCall = 0;
    if (millis() - lastCall > 5000) {
        float temp = readTemperature();  // Your sensor reading function
        callUpdateSensor("temp1", temp);
        lastCall = millis();
    }
}
```

### JavaScript Example (Node.js or Browser)

```javascript
const mqtt = require('mqtt');
const client = mqtt.connect('mqtt://localhost:1883');

const clientId = 'js-client-001';

// Subscribe to events with QoS 2
client.subscribe('rpc/event/+', {qos: 2});

// Handle incoming messages
client.on('message', (topic, payload) => {
    const data = JSON.parse(payload.toString());

    if (topic.startsWith('rpc/event/')) {
        const eventName = topic.replace('rpc/event/', '');
        handleEvent(eventName, data);
    }
});

function handleEvent(eventName, data) {
    switch(eventName) {
        case 'temperature':
            console.log(`Temperature: ${data.temp}°C`);
            break;
        case 'sensorUpdated':
            if (data.ok) {
                console.log(`Sensor ${data.sensor_id} updated`);
            } else {
                console.error(`Sensor ${data.sensor_id} failed: ${data.err}`);
            }
            break;
        case 'error':
            console.error(`Error on ${data.method}: ${data.err}`);
            break;
        default:
            console.log(`Event [${eventName}]:`, data);
    }
}

// Call a method with QoS 2
function callMethod(method, params) {
    client.publish(`rpc/call/${method}`, JSON.stringify(params), {qos: 2});
}

// Examples
callMethod('updateSensor', {sensor_id: 'temp1', value: 25.5});
callMethod('ping', {});
```

## Testing Strategy

1. **Unit Tests**: Test message parsing, validation, error handling
2. **Integration Tests**: Test with actual MQTT broker (e.g., Mosquitto in Docker)
3. **Load Tests**: Test with multiple concurrent clients
4. **Failure Tests**: Test broker disconnects, network issues, message loss

## Documentation Needs

1. **API Documentation**: Auto-generated from existing RegisterMethod/RegisterEvent
2. **Topic Reference**: List of all available topics
3. **Client Library Examples**: Go, JavaScript, Python examples
4. **Broker Setup Guide**: How to configure MQTT broker for production
5. **Migration Guide**: How to migrate from WebSocket to MQTT

## Example Use Case Flow

Let's walk through a complete example: Arduino reads a temperature sensor and sends it to the server.

### 1. Arduino Setup
```cpp
// On boot, Arduino connects and subscribes to events
mqtt.connect("arduino-livingroom");
mqtt.subscribe("rpc/event/sensorUpdated", 2);
mqtt.subscribe("rpc/event/temperature", 2);  // To receive other sensors
```

### 2. Arduino Sends Reading
```cpp
// Every 30 seconds, read sensor and publish
float temp = dht.readTemperature();
StaticJsonDocument<64> doc;
doc["sensor_id"] = "livingroom";
doc["value"] = temp;
doc["unit"] = "C";

char buffer[64];
serializeJson(doc, buffer);
mqtt.publish("rpc/call/updateSensor", buffer, 2);
```

### 3. Server Processes Call
```go
// MQTT handler for updateSensor
type UpdateSensorParams struct {
    SensorID string  `json:"sensor_id"`
    Value    float64 `json:"value"`
    Unit     string  `json:"unit"`
}

type UpdateSensorResult struct {
    SensorID string  `json:"sensor_id"`
    Value    float64 `json:"value"`
    OK       bool    `json:"ok"`
}

func (h *MQTTHandlers) UpdateSensor(ctx context.Context, params UpdateSensorParams) (UpdateSensorResult, error) {
    // Store in database
    err := h.db.SaveSensorReading(params.SensorID, params.Value, params.Unit)
    if err != nil {
        // Return error - handler will publish error event
        return UpdateSensorResult{}, fmt.Errorf("db_error: %w", err)
    }

    // Return success result - handler will publish sensorUpdated event
    return UpdateSensorResult{
        SensorID: params.SensorID,
        Value:    params.Value,
        OK:       true,
    }, nil
}

// The MQTTHandler automatically publishes the result as an event
// to rpc/event/sensorUpdated
```

### 4. Arduino Receives Confirmation
```cpp
// Arduino receives the sensorUpdated event
void handleEvent(const char* eventName, JsonDocument& data) {
    if (strcmp(eventName, "sensorUpdated") == 0) {
        const char* sensorId = data["sensor_id"];
        if (strcmp(sensorId, "livingroom") == 0) {
            Serial.println("✓ Reading uploaded successfully");
            digitalWrite(LED_PIN, HIGH);  // Flash success LED
        }
    }
}
```

### 5. Server Broadcasts Aggregate
```go
// Separately, server periodically broadcasts average temperature
go func() {
    ticker := time.NewTicker(1 * time.Minute)
    for range ticker.C {
        avg := calculateAverageTemperature()

        // Publish to MQTT devices
        mqttHandler.PublishEvent("temperature", map[string]any{
            "temp":   avg,
            "source": "all_sensors",
        })
    }
}()
```

### 6. All Arduinos Receive Broadcast
```cpp
// All devices subscribed to temperature events receive this
void handleEvent(const char* eventName, JsonDocument& data) {
    if (strcmp(eventName, "temperature") == 0) {
        float temp = data["temp"];
        updateDisplay(temp);  // Show on local display
    }
}
```

## Open Questions / Decisions Needed

1. **Client ID Requirements**: How should device client IDs be structured? Enforce pattern like `device-{type}-{location}`?
2. **Retained Messages**: Should any events be retained for late-joining devices?
3. **Will Messages**: Should server publish availability events using MQTT Will?
4. **Shared Subscriptions**: Support `$share` for load balancing across multiple servers?
5. **Schema Validation**: Should server validate params against expected schema?
6. **Binary Payloads**: Support for compact binary formats (MessagePack, CBOR) alongside JSON for extreme bandwidth constraints?
