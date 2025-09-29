# OpenUSP Examples

This directory contains working protocol agent examples demonstrating how to interact with the OpenUSP platform using TR-369 USP and TR-069 CWMP protocols.

## Available Examples

### 1. TR-369 USP Agent (`tr369-agent/`)

A complete USP (User Services Platform) agent that demonstrates TR-369 protocol communication with onboarding functionality via WebSocket MTP.

**Features:**
- USP 1.4 protocol support with automatic version detection
- WebSocket Message Transfer Protocol (MTP)
- Device onboarding and registration
- GET, SET, ADD, DELETE, and GetSupportedDM operations
- Protocol buffer message handling
- Real-time device communication with OpenUSP platform

**Usage:**
```bash
make start-tr369-agent
# or
go run examples/tr369-agent/main.go
```

### 2. TR-069 Agent (`tr069-agent/`)

A comprehensive CWMP (Common WAN Management Protocol) agent that demonstrates TR-069 protocol communication with onboarding functionality via HTTP/SOAP.

**Features:**
- TR-069 CWMP protocol support
- Device onboarding and registration
- SOAP/XML message handling with flexible parsing
- HTTP transport with basic authentication
- Inform, GetRPCMethods, GetParameterNames, GetParameterValues, SetParameterValues
- Session management and correlation
- TR-181 parameter validation

**Usage:**
```bash
make start-tr069-agent
# or
go run examples/tr069-agent/main.go
```

## Prerequisites

Before running any examples, ensure the OpenUSP platform is running:

```bash
# 1. Start PostgreSQL database
make postgres-up

# 2. Start all OpenUSP services
make run-services-ordered

# 3. Verify services are healthy
make health
```

This will start:
- **PostgreSQL Database**: `localhost:5433`
- **Data Service**: `localhost:9092` (gRPC)
- **MTP Service**: `localhost:8081` (WebSocket, MQTT, STOMP)
- **CWMP Service**: `localhost:7547` (HTTP/SOAP)
- **API Gateway**: `localhost:8080` (REST API)

## Service Endpoints

| Service | Protocol | Endpoint | Purpose |
|---------|----------|----------|---------|
| MTP Service | WebSocket | `ws://localhost:8081/ws` | TR-369 USP communication |
| MTP Service | HTTP | `http://localhost:8081/usp` | Demo UI |
| CWMP Service | HTTP | `http://localhost:7547` | TR-069 CWMP communication |
| API Gateway | HTTP | `http://localhost:8080` | REST API access |
| Data Service | gRPC | `localhost:9092` | Internal data operations |

## Protocol Comparison

### TR-369 USP vs TR-069 CWMP

| Aspect | TR-369 USP | TR-069 CWMP |
|--------|-----------|-------------|
| **Protocol** | Binary Protocol Buffers | XML/SOAP |
| **Transport** | WebSocket, MQTT, STOMP, Unix Socket | HTTP/HTTPS |
| **Message Format** | Structured binary | Text-based XML |
| **Session Model** | Stateless or stateful | Session-based |
| **Performance** | High (binary, compressed) | Moderate (XML overhead) |
| **Complexity** | Modern, flexible | Traditional, well-established |
| **Use Cases** | IoT, real-time, high-volume | Traditional CPE management |

## Running Examples

### 1. TR-369 USP Client

```bash
# Start OpenUSP platform
make run-services-ordered

# In another terminal, run the USP agent
go run examples/tr369-agent/main.go

# Expected: WebSocket connection, USP messages exchange
```

### 2. TR-069 CWMP Client

```bash
# Start OpenUSP platform  
make run-services-ordered

# In another terminal, run the CWMP agent
go run examples/tr069-agent/main.go

# Expected: HTTP/SOAP messages exchange with authentication
```

## Development Tips

### Adding New Examples

1. Create a new directory under `examples/`
2. Add `main.go` with your example code
3. Include a `README.md` with documentation
4. Update this main examples README
5. Test compilation: `go build examples/your-example/main.go`

### Common Patterns

**USP Client Pattern:**
```go
// 1. Create client
client := NewUSPClient(endpointID, controllerID)

// 2. Connect to MTP service
client.Connect("ws://localhost:8081/ws")

// 3. Create USP messages
record := createGetMessage()

// 4. Send and receive
client.sendRecord(record)
response := client.readResponse()
```

**CWMP Client Pattern:**
```go
// 1. Create client with auth
client := NewCWMPClient(url, username, password, deviceID)

// 2. Create SOAP envelope
envelope := createSOAPEnvelope(msgID)

// 3. Send HTTP request
client.sendSOAPRequest(envelope)
```

## Testing and Validation

### Manual Testing
```bash
# Test TR-369 WebSocket endpoint
curl -i -H "Connection: Upgrade" -H "Upgrade: websocket" \
     -H "Sec-WebSocket-Key: test" -H "Sec-WebSocket-Version: 13" \
     http://localhost:8081/ws

# Test TR-069 CWMP endpoint
curl -u acs:acs123 -X POST -H "Content-Type: text/xml" \
     -d @examples/soap-request.xml http://localhost:7547
```

### Integration Testing
```bash
# Run example programs while monitoring logs
tail -f logs/*.log

# In separate terminals:
go run examples/tr369-agent/main.go
go run examples/tr069-agent/main.go
```

## Troubleshooting

### Common Issues

1. **Connection Refused**
   - Ensure OpenUSP services are running: `make health`
   - Check port availability: `lsof -i :8081 -i :7547`

2. **Authentication Failed (CWMP)**
   - Verify credentials: `acs/acs123`
   - Check CWMP service logs

3. **Protocol Errors**
   - Verify protocol buffer compatibility
   - Check SOAP envelope structure
   - Review service logs for detailed errors

4. **Timeout Issues**
   - Increase client timeouts
   - Check network connectivity
   - Verify service health

### Debug Mode

Enable verbose logging in examples by modifying log levels:
```go
// Add to main() function
log.SetLevel(log.DebugLevel)
```

## Contributing

When adding new examples:

1. Follow the existing code structure
2. Include comprehensive error handling
3. Add detailed logging and output
4. Provide clear documentation
5. Test with the OpenUSP platform
6. Update this README with your example

## Resources

- **TR-369 Specification**: [Broadband Forum TR-369](https://www.broadband-forum.org/technical/download/TR-369.pdf)
- **TR-069 Specification**: [Broadband Forum TR-069](https://www.broadband-forum.org/technical/download/TR-069.pdf)
- **TR-181 Data Model**: [Broadband Forum TR-181](https://www.broadband-forum.org/technical/download/TR-181.pdf)
- **Protocol Buffers**: [Google Protocol Buffers](https://developers.google.com/protocol-buffers)
- **WebSocket Protocol**: [RFC 6455](https://tools.ietf.org/html/rfc6455)