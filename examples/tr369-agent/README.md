# TR-369 USP Client Example

This example demonstrates how to create a TR-369 User Services Platform (USP) client that communicates with the OpenUSP platform via WebSocket MTP (Message Transfer Protocol).

## Features

- **USP 1.4 Protocol Support**: Uses USP version 1.4 protocol buffers
- **WebSocket MTP**: Communicates via WebSocket message transport
- **USP Operations**: Demonstrates GET, SET, and GetSupportedDM operations
- **Protocol Buffer Messages**: Creates and parses USP records and messages
- **Base64 Encoding**: Handles message encoding for WebSocket transmission

## USP Operations Demonstrated

1. **GET Request**: Retrieves parameter values from device data model
   - Parameters: `Device.DeviceInfo.*`, `Device.Ethernet.*`
   - Shows how to request parameter values from the device

2. **GetSupportedDM Request**: Queries supported data model elements
   - Object Path: `Device.*`
   - Returns supported objects, parameters, commands, and events

3. **SET Request**: Updates parameter values on the device
   - Example: Sets `Device.DeviceInfo.Description`
   - Demonstrates parameter modification

## Prerequisites

1. **OpenUSP Platform Running**:
   ```bash
   # Start PostgreSQL database
   make postgres-up
   
   # Start all services in proper order
   make run-services-ordered
   ```

2. **MTP Service**: Must be running on `localhost:8081`
3. **WebSocket Endpoint**: Available at `ws://localhost:8081/ws`

## Running the Example

```bash
# From the OpenUSP root directory
go run examples/tr369-agent/main.go
```

## Expected Output

```
OpenUSP TR-369 Client Example
=============================
This example demonstrates TR-369 USP protocol communication
with the OpenUSP platform via WebSocket MTP.

Connecting to MTP Service at: ws://localhost:8081/ws
Connected to MTP Service successfully

🚀 Starting TR-369 USP Client Demonstration
=========================================

1️⃣  Sending USP GET Request...
Sending USP Record (size: 156 bytes)
Record Details - Version: 1.4, From: tr369-example-agent, To: tr369-example-controller
USP Record sent successfully
Waiting for GET response...
Received response (size: 89 bytes): Welcome to OpenUSP MTP Service WebSocket Demo!

✅ TR-369 USP Client demonstration completed!
```

## Code Structure

### USPClient Class
- **Connection Management**: WebSocket connection handling
- **Message Creation**: USP record and message construction
- **Protocol Handling**: USP 1.4 protocol buffer operations
- **Response Processing**: Response parsing and display

### Key Methods
- `createGetMessage()`: Creates USP GET request records
- `createSetMessage()`: Creates USP SET request records  
- `createGetSupportedDMMessage()`: Creates GetSupportedDM request records
- `sendRecord()`: Sends USP records via WebSocket
- `readResponse()`: Reads and processes responses

## USP Record Structure

The client creates USP 1.4 records with:
- **Version**: "1.4"
- **Endpoint IDs**: From/To agent and controller identification
- **Security**: PLAINTEXT (no encryption for demo)
- **Record Type**: NoSessionContext for stateless communication
- **Payload**: Marshaled USP messages

## Integration Points

- **MTP Service**: WebSocket endpoint at `/ws`
- **USP Parser**: Server-side USP record processing
- **Data Service**: Backend parameter storage via gRPC
- **TR-181 Data Model**: Device parameter namespace

## Customization

You can modify the example to:
- Change parameter paths in GET requests
- Add more SET operations
- Implement different USP operations (Add, Delete, Operate)
- Use different MTP transports (MQTT, STOMP)
- Add error handling and retry logic

## Troubleshooting

1. **Connection Failed**: Ensure MTP service is running
2. **No Response**: Check MTP service logs for processing errors
3. **Parse Errors**: Verify USP protocol buffer compatibility
4. **Timeout**: Increase read deadline or check network connectivity

## Related Files

- `/pkg/proto/v1_4/`: USP 1.4 protocol buffer definitions
- `/cmd/mtp-service/`: MTP service implementation
- `/internal/usp/parser.go`: USP message parsing logic