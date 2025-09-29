# OpenUSP MTP Service

The MTP (Message Transfer Protocol) Service provides multi-protocol message transport capabilities for the OpenUSP TR-369 implementation. It supports MQTT, STOMP, WebSocket, and Unix Domain Socket protocols for USP message exchange.

## Features

- **Multi-Protocol Support**: MQTT, STOMP, WebSocket, and Unix Domain Socket
- **USP Version Support**: Both USP 1.3 and 1.4 protocol versions
- **Automatic Version Detection**: Detects USP protocol version from incoming messages
- **Health Monitoring**: Built-in health check and status endpoints
- **Demo Interface**: Web-based demonstration UI

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    MTP Service                              │
├─────────────────┬─────────────────┬─────────────────────────┤
│  MQTT Protocol  │ STOMP Protocol  │   WebSocket Protocol    │
│                 │                 │                         │
│ • Topics        │ • Destinations  │ • HTTP Upgrade          │
│ • Pub/Sub       │ • Queues        │ • Binary Messages       │
│ • QoS Support   │ • Transactions  │ • Client Management     │
└─────────────────┴─────────────────┴─────────────────────────┘
                            │
┌───────────────────────────┼───────────────────────────────────┐
│            Unix Domain Socket Protocol                       │
│                                                              │
│ • Local IPC        • Binary Protocol    • File Descriptors  │
└──────────────────────────────────────────────────────────────┘
                            │
                    ┌───────▼──────┐
                    │ USP Message  │
                    │   Handler    │
                    │              │
                    │ • v1.3 & v1.4│
                    │ • Validation │
                    │ • Processing │
                    └──────────────┘
```

## Running the Service

### Start MTP Service
```bash
cd openusp
go run cmd/mtp-service/main.go
```

### Access Demo Interface
Open your browser to: http://localhost:8081/usp

### Health Check
```bash
curl http://localhost:8081/health
```

### Service Status
```bash
curl http://localhost:8081/status
```

### Simulate USP Message Processing
```bash
curl -X POST http://localhost:8081/simulate
```

## Protocol Endpoints

### MQTT
- **Broker URL**: tcp://localhost:1883 (configurable)
- **Topics**:
  - `/usp/controller/+` - Messages to controllers
  - `/usp/agent/+` - Messages to agents  
  - `/usp/broadcast` - Broadcast messages

### STOMP
- **Broker URL**: tcp://localhost:61613 (configurable)
- **Destinations**:
  - `/queue/usp.controller` - Controller queue
  - `/queue/usp.agent` - Agent queue
  - `/topic/usp.broadcast` - Broadcast topic

### WebSocket
- **Port**: 8081 (configurable)
- **Endpoint**: `ws://localhost:8081/usp`
- **Protocol**: Binary USP messages over WebSocket

### Unix Domain Socket
- **Socket Path**: `/tmp/usp-agent.sock` (configurable)
- **Protocol**: Binary USP messages over Unix socket

## Configuration

The service can be configured through the `Config` struct:

```go
config := &Config{
    WebSocketPort:    8081,
    UnixSocketPath:   "/tmp/usp-agent.sock",
    EnableWebSocket:  true,
    EnableUnixSocket: true,
    EnableMQTT:       true,
    EnableSTOMP:      true,
}
```

## USP Message Flow

1. **Message Reception**: Incoming USP messages via any supported protocol
2. **Version Detection**: Automatic detection of USP 1.3 or 1.4 protocol version
3. **Message Processing**: Route to appropriate USP message handler
4. **Response Generation**: Create appropriate USP response message
5. **Response Delivery**: Send response back via the same protocol

## Integration with USP Core Service

The MTP service is designed to integrate with the USP Core Service:

```go
// Create message handler that connects to USP Core Service
handler := &USPServiceHandler{
    uspCoreURL: "http://localhost:50051",
}

// Create MTP service with handler
mtpService, err := NewMTPService(config, handler)
```

## Dependencies

- `github.com/eclipse/paho.mqtt.golang` - MQTT client library
- `github.com/gorilla/websocket` - WebSocket implementation
- Protocol Buffers for USP message serialization

## Development

### Adding New Protocols

1. Create new protocol implementation in `internal/mtp/`
2. Implement the protocol interface
3. Add protocol to MTP service configuration
4. Update service initialization

### Testing

The service includes a demo mode with simulated USP message processing for testing without requiring actual USP agents or controllers.

## Production Deployment

For production deployment:

1. Configure proper broker URLs for MQTT/STOMP
2. Set up SSL/TLS certificates for secure connections
3. Configure authentication and authorization
4. Set up monitoring and logging
5. Deploy with container orchestration (Docker/Kubernetes)

## Monitoring

The service provides several monitoring endpoints:

- `/health` - Basic health check
- `/status` - Detailed service status
- `/metrics` - Prometheus metrics (future)

## Security Considerations

- Use TLS for all external protocol connections
- Implement proper authentication for WebSocket connections
- Secure Unix domain socket file permissions
- Validate all incoming USP messages
- Rate limiting for protocol endpoints