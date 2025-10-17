# MTP Routing Architecture

## Overview

The OpenUSP platform now implements a comprehensive MTP (Message Transfer Protocol) routing system that enables the USP service to provide protocol-specific destination information to MTP services. This allows each agent to have its own dedicated destination (queue, topic, URL, etc.) rather than all agents sharing a single destination.

## Architecture

### Message Flow

```
Agent (WebSocket/STOMP/MQTT)
    ↓ [Inbound Message with Routing Info]
MTP Service (WebSocket/STOMP/MQTT)
    ↓ [Kafka: USPMessageEvent with MTPDestination]
USP Service
    ├── [Cache routing info by endpoint ID]
    └── [Process USP message]
    ↓ [Kafka: USPMessageEvent with MTPDestination]
MTP Service (WebSocket/STOMP/MQTT)
    ↓ [Extract routing from envelope]
Agent (Specific destination: queue/topic/URL)
```

### Key Components

#### 1. MTPDestination Structure (`pkg/kafka/events.go`)

```go
type MTPDestination struct {
    WebSocketURL string `json:"websocket_url,omitempty"` // For WebSocket agents
    STOMPQueue   string `json:"stomp_queue,omitempty"`   // For STOMP agents
    MQTTTopic    string `json:"mqtt_topic,omitempty"`    // For MQTT agents
    HTTPURL      string `json:"http_url,omitempty"`      // For HTTP agents (future)
}
```

This structure is included in the `USPMessageEvent` envelope that flows through Kafka.

#### 2. Enhanced Producer (`pkg/kafka/producer.go`)

- **PublishUSPMessageWithDestination()**: New method that accepts `MTPDestination` parameter
- **PublishUSPMessage()**: Wrapper that calls the new method with empty destination (backward compatible)

#### 3. Database Model (`internal/database/models.go`)

The `Device` model now includes MTP routing fields for persistent storage:

```go
type Device struct {
    // ... existing fields ...
    MTPProtocol  string `json:"mtp_protocol"`   // websocket, stomp, mqtt, http
    WebSocketURL string `json:"websocket_url"`  // For WebSocket routing
    STOMPQueue   string `json:"stomp_queue"`    // For STOMP routing
    MQTTTopic    string `json:"mqtt_topic"`     // For MQTT routing
    HTTPURL      string `json:"http_url"`       // For HTTP routing
}
```

#### 4. USP Service Routing Cache (`cmd/usp-service/main.go`)

**In-Memory Cache**:
```go
var (
    endpointMTPCache   = make(map[string]struct{
        protocol    string
        destination string
    })
    endpointMTPCacheMu sync.RWMutex
)
```

**Key Methods**:
- **storeEndpointMTPRouting()**: Stores routing info when inbound messages arrive
- **getEndpointMTPRouting()**: Retrieves routing info when sending responses

**Logic**:
1. When inbound message arrives: Extract and cache routing from `MTPDestination`
2. When sending response: Retrieve cached routing and include in outbound message

### MTP Service Updates

#### WebSocket MTP (`cmd/mtp-services/websocket/main.go`)

**Inbound** (processUSPMessage):
- Constructs WebSocket URL from config: `ws://localhost:8081/usp`
- Includes in `MTPDestination.WebSocketURL`
- Publishes with `PublishUSPMessageWithDestination()`

**Outbound** (setupKafkaConsumer handler):
- Parses Kafka envelope to extract `MTPDestination`
- Uses `WebSocketURL` if provided, otherwise falls back to default
- Sends message to specific WebSocket connection

#### STOMP MTP (`cmd/mtp-services/stomp/main.go`)

**Inbound** (ProcessUSPMessage):
- Gets STOMP queue from config: `/queue/controller->agent`
- Includes in `MTPDestination.STOMPQueue`
- Publishes with `PublishUSPMessageWithDestination()`

**Outbound** (setupKafkaConsumer handler):
- Parses Kafka envelope to extract `MTPDestination`
- Uses `STOMPQueue` if provided, otherwise falls back to config or default `/queue/controller->agent`
- Sends message to specific STOMP queue

#### MQTT MTP (`cmd/mtp-services/mqtt/main.go`)

**Inbound** (ProcessUSPMessage):
- Uses default topic: `usp/agent/response`
- Includes in `MTPDestination.MQTTTopic`
- Publishes with `PublishUSPMessageWithDestination()`

**Outbound** (setupKafkaConsumer handler):
- Parses Kafka envelope to extract `MTPDestination`
- Uses `MQTTTopic` if provided, otherwise falls back to default `usp/agent/response`
- Ready to send to specific MQTT topic (when broker implementation is complete)

## Configuration

### STOMP Example (`configs/agents.yml`)

```yaml
stomp:
  broker_url: "tcp://localhost:61613"
  username: "admin"
  password: "admin"
  destinations:
    inbound: "/queue/controller->agent"   # Controller sends to agent
    outbound: "/queue/agent->controller"  # Agent sends to controller
```

### WebSocket Example (`configs/agents.yml`)

```yaml
websocket:
  url: "ws://localhost:8081/usp"
  subprotocol: "v1.usp"
```

### MQTT Example (`configs/agents.yml`)

```yaml
mqtt:
  broker_url: "tcp://localhost:1883"
  topic_request: "usp/agent/request"    # Controller sends to agent
  topic_response: "usp/agent/response"  # Agent sends to controller
```

## Benefits

### 1. **Multi-Agent Support**
- Multiple agents can use the same MTP protocol with different destinations
- Example: Agent A uses `/queue/agent-A`, Agent B uses `/queue/agent-B`

### 2. **Proper Isolation**
- Each agent receives only its own messages
- No message leakage between agents

### 3. **Flexible Configuration**
- Agents can specify custom queue/topic names
- No hardcoded destination assumptions

### 4. **Scalability**
- Supports thousands of agents with unique destinations
- In-memory cache for fast routing lookups

### 5. **Protocol Agnostic**
- Same architecture works for WebSocket, STOMP, MQTT, and future protocols
- Consistent routing model across all MTPs

## Fallback Strategy

Each MTP service implements a three-tier fallback:

1. **Primary**: Use `MTPDestination` from Kafka envelope (preferred)
2. **Secondary**: Use configuration value from `openusp.yml` or `agents.yml`
3. **Tertiary**: Use hardcoded default value

Example (STOMP):
```go
stompQueue := event.MTPDestination.STOMPQueue
if stompQueue == "" {
    stompQueue = s.config.MTP.STOMP.Destinations.Outbound
    if stompQueue == "" {
        stompQueue = "/queue/controller->agent" // Default fallback
    }
    log.Printf("⚠️ STOMP: No queue in MTPDestination, using fallback: %s", stompQueue)
}
```

## Future Enhancements

### 1. Database Persistence
- Store routing info in Device model
- Query from database if not in cache
- Update routing info when agents reconnect

### 2. Dynamic Discovery
- Agents announce their preferred destinations on connection
- Controller updates routing table automatically

### 3. Load Balancing
- Multiple agents share same endpoint ID
- Route to least-loaded agent

### 4. HTTP MTP Support
- Add HTTPURL to MTPDestination
- Support HTTP/HTTPS callback URLs
- Enable CoAP over HTTP

### 5. Routing Analytics
- Track routing patterns
- Identify routing failures
- Optimize cache size

## Testing Strategy

### Unit Tests
- Test `storeEndpointMTPRouting()` and `getEndpointMTPRouting()`
- Verify cache behavior under concurrent access
- Test fallback logic in each MTP service

### Integration Tests
1. Start WebSocket agent with custom URL
2. Send message from controller
3. Verify message routes to correct WebSocket connection

4. Start STOMP agent with custom queue
5. Send message from controller
6. Verify message routes to correct STOMP queue

7. Start multiple agents on same protocol
8. Send messages to each agent
9. Verify proper isolation

### Performance Tests
- Cache lookup latency
- Memory usage with 10,000+ cached routes
- Throughput under high message load

## Migration Guide

### For Existing Deployments

1. **Update Code**: Pull latest changes with MTP routing
2. **No Config Changes**: Fallback ensures backward compatibility
3. **Optional Database Migration**: Add new fields to Device table if persistent routing desired
4. **Test**: Verify existing agents continue to work
5. **Enable Routing**: Configure custom destinations for new agents

### For New Deployments

1. **Configure Destinations**: Set unique queues/topics for each agent in configuration
2. **Enable Routing**: USP service automatically stores and uses routing info
3. **Monitor**: Check logs for routing decisions and fallbacks

## Troubleshooting

### Issue: "No queue in MTPDestination, using fallback"

**Cause**: MTP service didn't include routing info in inbound message

**Solution**: Check MTP service logs to ensure `MTPDestination` is populated

### Issue: Messages not reaching agent

**Cause**: Incorrect routing info cached

**Solution**: 
1. Check USP service logs for cached routing info
2. Restart USP service to clear cache
3. Verify agent configuration matches cached destination

### Issue: Multiple agents receiving same message

**Cause**: Agents sharing same destination

**Solution**: Configure unique destinations for each agent

## Version History

- **v1.3.0 (2025-01-17)**: Initial MTP routing implementation
  - Added MTPDestination structure
  - Updated WebSocket, STOMP, and MQTT MTP services
  - Implemented USP service routing cache
  - Full bidirectional routing support
