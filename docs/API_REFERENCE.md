# 📖 API Reference

Complete API documentation for OpenUSP services.

## Table of Contents

- [REST API Gateway](#rest-api-gateway)
- [WebSocket API](#websocket-api)
- [gRPC Internal APIs](#grpc-internal-apis)
- [Authentication](#authentication)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)

## REST API Gateway

Base URL: `http://localhost:8080/api/v1`

### Devices API

#### List Devices
```http
GET /api/v1/devices
```

**Query Parameters:**
- `limit` (optional): Number of devices to return (default: 50)
- `offset` (optional): Number of devices to skip (default: 0)
- `filter` (optional): Filter by device properties

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "device-001",
      "endpoint_id": "cpe-device-001",
      "oui": "00D09E",
      "product_class": "CPE",
      "serial_number": "SN123456789",
      "hardware_version": "1.0",
      "software_version": "2.1.0",
      "manufacturer": "Example Corp",
      "model_name": "Model-X1",
      "description": "Residential Gateway",
      "device_type": "InternetGatewayDevice",
      "protocol_version": "1.4",
      "supported_protocols": ["usp"],
      "last_contact_time": "2024-01-15T10:30:00Z",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ],
  "meta": {
    "total": 1,
    "limit": 50,
    "offset": 0
  }
}
```

#### Get Device
```http
GET /api/v1/devices/{device_id}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "device-001",
    "endpoint_id": "cpe-device-001",
    "oui": "00D09E",
    "product_class": "CPE",
    "serial_number": "SN123456789",
    "hardware_version": "1.0",
    "software_version": "2.1.0",
    "manufacturer": "Example Corp",
    "model_name": "Model-X1",
    "description": "Residential Gateway",
    "device_type": "InternetGatewayDevice",
    "protocol_version": "1.4",
    "supported_protocols": ["usp"],
    "last_contact_time": "2024-01-15T10:30:00Z",
    "created_at": "2024-01-01T00:00:00Z",
    "updated_at": "2024-01-15T10:30:00Z"
  }
}
```

#### Create Device
```http
POST /api/v1/devices
Content-Type: application/json
```

**Request Body:**
```json
{
  "endpoint_id": "cpe-device-002",
  "oui": "00D09E",
  "product_class": "CPE",
  "serial_number": "SN987654321",
  "hardware_version": "1.1",
  "software_version": "2.2.0",
  "manufacturer": "Example Corp",
  "model_name": "Model-X2",
  "description": "Business Gateway",
  "device_type": "InternetGatewayDevice",
  "protocol_version": "1.4",
  "supported_protocols": ["usp", "cwmp"]
}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "id": "device-002",
    "endpoint_id": "cpe-device-002",
    "oui": "00D09E",
    "product_class": "CPE",
    "serial_number": "SN987654321",
    "hardware_version": "1.1",
    "software_version": "2.2.0",
    "manufacturer": "Example Corp",
    "model_name": "Model-X2",
    "description": "Business Gateway",
    "device_type": "InternetGatewayDevice",
    "protocol_version": "1.4",
    "supported_protocols": ["usp", "cwmp"],
    "last_contact_time": null,
    "created_at": "2024-01-15T11:00:00Z",
    "updated_at": "2024-01-15T11:00:00Z"
  }
}
```

#### Update Device
```http
PUT /api/v1/devices/{device_id}
Content-Type: application/json
```

**Request Body:**
```json
{
  "software_version": "2.3.0",
  "description": "Updated Business Gateway"
}
```

#### Delete Device
```http
DELETE /api/v1/devices/{device_id}
```

**Response:**
```json
{
  "success": true,
  "data": {
    "message": "Device deleted successfully"
  }
}
```

### Parameters API

#### List Device Parameters
```http
GET /api/v1/devices/{device_id}/parameters
```

**Query Parameters:**
- `path` (optional): Filter by parameter path
- `writable` (optional): Filter by writable status (true/false)

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "param-001",
      "device_id": "device-001",
      "path": "Device.DeviceInfo.ModelName",
      "value": "Model-X1",
      "type": "string",
      "writable": false,
      "last_update": "2024-01-15T10:30:00Z",
      "created_at": "2024-01-01T00:00:00Z",
      "updated_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

#### Get Parameter
```http
GET /api/v1/devices/{device_id}/parameters/{parameter_id}
```

#### Set Parameter Value
```http
PUT /api/v1/devices/{device_id}/parameters/{parameter_id}
Content-Type: application/json
```

**Request Body:**
```json
{
  "value": "New Value"
}
```

#### Bulk Parameter Operations
```http
POST /api/v1/devices/{device_id}/parameters/bulk
Content-Type: application/json
```

**Request Body:**
```json
{
  "operation": "set",
  "parameters": [
    {
      "path": "Device.WiFi.SSID.1.SSID",
      "value": "MyNewNetwork"
    },
    {
      "path": "Device.WiFi.SSID.1.Enable",
      "value": "true"
    }
  ]
}
```

### Alerts API

#### List Alerts
```http
GET /api/v1/alerts
```

**Query Parameters:**
- `device_id` (optional): Filter by device
- `severity` (optional): Filter by severity level
- `start_date` (optional): Filter from date (ISO 8601)
- `end_date` (optional): Filter to date (ISO 8601)

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "alert-001",
      "device_id": "device-001",
      "alert_type": "firmware_update_available",
      "severity": "info",
      "message": "Firmware update available: v2.3.0",
      "details": {
        "current_version": "2.1.0",
        "available_version": "2.3.0",
        "release_notes": "Security fixes and performance improvements"
      },
      "acknowledged": false,
      "created_at": "2024-01-15T09:00:00Z",
      "updated_at": "2024-01-15T09:00:00Z"
    }
  ]
}
```

#### Acknowledge Alert
```http
POST /api/v1/alerts/{alert_id}/acknowledge
```

### Sessions API

#### List Active Sessions
```http
GET /api/v1/sessions
```

**Response:**
```json
{
  "success": true,
  "data": [
    {
      "id": "session-001",
      "device_id": "device-001",
      "protocol": "usp",
      "transport": "websocket",
      "status": "active",
      "started_at": "2024-01-15T10:00:00Z",
      "last_activity": "2024-01-15T10:30:00Z",
      "client_info": {
        "ip_address": "192.168.1.100",
        "user_agent": "OpenUSP-Agent/1.0"
      }
    }
  ]
}
```

#### Terminate Session
```http
DELETE /api/v1/sessions/{session_id}
```

## WebSocket API

### Connect to MTP Service
```
ws://localhost:8081/ws
```

### USP Message Format

Send USP Protocol Buffer messages directly:

```javascript
// Connect to WebSocket
const ws = new WebSocket('ws://localhost:8081/ws');

// Send USP Get request
const uspRecord = {
  version: "1.4",
  to_id: "controller-001",
  from_id: "device-001",
  payload_security: "PLAINTEXT",
  payload: {
    request: {
      get: {
        param_paths: ["Device.DeviceInfo.ModelName"]
      }
    }
  }
};

ws.send(JSON.stringify(uspRecord));

// Handle responses
ws.onmessage = function(event) {
  const response = JSON.parse(event.data);
  console.log('USP Response:', response);
};
```

### Message Types

#### Get Request
```json
{
  "version": "1.4",
  "to_id": "device-001",
  "from_id": "controller-001",
  "payload_security": "PLAINTEXT",
  "payload": {
    "request": {
      "get": {
        "param_paths": [
          "Device.DeviceInfo.",
          "Device.WiFi.SSID.1."
        ]
      }
    }
  }
}
```

#### Set Request
```json
{
  "version": "1.4",
  "to_id": "device-001",
  "from_id": "controller-001",
  "payload_security": "PLAINTEXT",
  "payload": {
    "request": {
      "set": {
        "update_objs": [
          {
            "obj_path": "Device.WiFi.SSID.1.",
            "param_settings": [
              {
                "param": "SSID",
                "value": "MyNewNetwork"
              }
            ]
          }
        ]
      }
    }
  }
}
```

## gRPC Internal APIs

### Data Service

Service: `dataservice.DataService`

#### Device Operations

```protobuf
// Get device by ID
rpc GetDevice(GetDeviceRequest) returns (GetDeviceResponse);

// List devices
rpc ListDevices(ListDevicesRequest) returns (ListDevicesResponse);

// Create device
rpc CreateDevice(CreateDeviceRequest) returns (CreateDeviceResponse);

// Update device
rpc UpdateDevice(UpdateDeviceRequest) returns (UpdateDeviceResponse);

// Delete device
rpc DeleteDevice(DeleteDeviceRequest) returns (DeleteDeviceResponse);
```

#### Parameter Operations

```protobuf
// Get parameters for device
rpc GetDeviceParameters(GetDeviceParametersRequest) returns (GetDeviceParametersResponse);

// Set parameter value
rpc SetParameterValue(SetParameterValueRequest) returns (SetParameterValueResponse);

// Bulk parameter operations
rpc BulkParameterOperation(BulkParameterOperationRequest) returns (BulkParameterOperationResponse);
```

## Authentication

### API Key Authentication

Include API key in request headers:

```http
Authorization: Bearer your-api-key-here
```

### Basic Authentication (CWMP)

For CWMP/TR-069 endpoints:

```http
Authorization: Basic base64(username:password)
```

Default credentials:
- Username: `acs`
- Password: `acs123`

### JWT Authentication (Future)

Support for JWT tokens:

```http
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

## Error Handling

### HTTP Status Codes

- `200 OK`: Successful request
- `201 Created`: Resource created successfully
- `400 Bad Request`: Invalid request parameters
- `401 Unauthorized`: Authentication required
- `403 Forbidden`: Insufficient permissions
- `404 Not Found`: Resource not found
- `409 Conflict`: Resource already exists
- `422 Unprocessable Entity`: Validation errors
- `500 Internal Server Error`: Server error

### Error Response Format

```json
{
  "success": false,
  "error": "Device not found",
  "details": {
    "code": "DEVICE_NOT_FOUND",
    "message": "Device with ID 'device-001' not found",
    "field": "device_id",
    "timestamp": "2024-01-15T10:30:00Z"
  }
}
```

### USP Error Codes

USP operations may return specific error codes:

- `7000`: Message failed
- `7001`: Message not supported  
- `7002`: Request denied
- `7003`: Internal error
- `7004`: Invalid arguments
- `7005`: Resources exceeded
- `7006`: Permission denied
- `7007`: Invalid configuration
- `7008`: Invalid path syntax
- `7009`: Parameter action failed
- `7010`: Unsupported parameter
- `7011`: Invalid parameter value
- `7012`: Attempt to update non-writable parameter

## Rate Limiting

### Default Limits

- **REST API**: 1000 requests per minute per IP
- **WebSocket**: 100 messages per minute per connection
- **Bulk Operations**: 10 operations per minute per device

### Rate Limit Headers

```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1642248600
```

### Exceeding Limits

```json
{
  "success": false,
  "error": "Rate limit exceeded",
  "details": {
    "code": "RATE_LIMIT_EXCEEDED",
    "message": "Too many requests. Try again in 60 seconds.",
    "retry_after": 60
  }
}
```

## SDK and Client Libraries

### Go Client

```go
import "openusp/pkg/client"

client := client.NewOpenUSPClient("http://localhost:8080", "your-api-key")

devices, err := client.ListDevices(context.Background(), &client.ListDevicesOptions{
    Limit: 10,
})
```

### JavaScript/Node.js Client

```javascript
const OpenUSP = require('openusp-client');

const client = new OpenUSP.Client({
  baseURL: 'http://localhost:8080',
  apiKey: 'your-api-key'
});

const devices = await client.devices.list({ limit: 10 });
```

### Python Client

```python
import openusp

client = openusp.Client(
    base_url='http://localhost:8080',
    api_key='your-api-key'
)

devices = client.devices.list(limit=10)
```

## Testing the API

### Using curl

```bash
# List devices
curl -H "Authorization: Bearer your-api-key" \
     http://localhost:8080/api/v1/devices

# Get device
curl -H "Authorization: Bearer your-api-key" \
     http://localhost:8080/api/v1/devices/device-001

# Create device
curl -X POST \
     -H "Authorization: Bearer your-api-key" \
     -H "Content-Type: application/json" \
     -d '{"endpoint_id":"test-device","oui":"123456"}' \
     http://localhost:8080/api/v1/devices
```

### Using httpie

```bash
# List devices
http GET localhost:8080/api/v1/devices Authorization:"Bearer your-api-key"

# Create device
http POST localhost:8080/api/v1/devices \
     Authorization:"Bearer your-api-key" \
     endpoint_id=test-device oui=123456
```

### Using Postman

Import the OpenAPI specification from `/api/v1/swagger.json` to automatically generate a Postman collection.

## API Versioning

### Current Version

- **Version**: v1
- **Base Path**: `/api/v1`
- **Supported Until**: TBD

### Backward Compatibility

- Additive changes (new fields, endpoints) don't require version bump
- Breaking changes require new API version
- Previous versions supported for 12 months minimum

---

For more examples and interactive testing, visit the [Swagger UI](http://localhost:8080/swagger/index.html) when the API Gateway is running.