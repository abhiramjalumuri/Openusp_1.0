# OpenUSP API Gateway

A modern REST API Gateway service that provides HTTP endpoints backed by gRPC calls to the OpenUSP data service.

## Features

- **Clean REST API**: Modern HTTP/JSON API with proper status codes and error handling
- **gRPC Backend**: All data operations are proxied to the gRPC data service
- **Device Management**: Complete CRUD operations for devices
- **Parameter Management**: Handle device parameters
- **Alert Management**: Manage device alerts and notifications
- **Session Management**: Track device sessions and activities
- **Health Monitoring**: Built-in health checks and status endpoints
- **CORS Support**: Cross-origin resource sharing enabled
- **Graceful Shutdown**: Proper cleanup of resources on shutdown

## Architecture

```
HTTP/REST Client → API Gateway → gRPC Data Service → PostgreSQL Database
```

The API Gateway acts as a translation layer between HTTP/REST clients and the internal gRPC microservices.

## Configuration

### Environment Variables

- `PORT` - Server port (default: 8080)
- `DATA_SERVICE_ADDR` - Data service gRPC address (default: localhost:9092)

### Command Line Flags

- `-version` - Show version information
- `-help` - Show help information

## API Endpoints

### Health & Status

- `GET /health` - Health check endpoint
- `GET /status` - Detailed status information

### Device Management

- `GET /api/v1/devices` - List devices (with pagination)
  - Query params: `offset`, `limit`
- `POST /api/v1/devices` - Create new device
- `GET /api/v1/devices/{id}` - Get device by ID
- `PUT /api/v1/devices/{id}` - Update device
- `DELETE /api/v1/devices/{id}` - Delete device
- `GET /api/v1/devices/{id}/parameters` - Get device parameters
- `GET /api/v1/devices/{id}/alerts` - Get device alerts
  - Query params: `resolved` (true/false)
- `GET /api/v1/devices/{id}/sessions` - Get device sessions

### Parameter Management

- `POST /api/v1/parameters` - Create/update parameter
- `DELETE /api/v1/parameters/{device_id}/{path}` - Delete parameter

### Alert Management

- `GET /api/v1/alerts` - List alerts (with pagination)
  - Query params: `offset`, `limit`
- `POST /api/v1/alerts` - Create new alert
- `PUT /api/v1/alerts/{id}/resolve` - Resolve alert

### Session Management

- `POST /api/v1/sessions` - Create new session
- `PUT /api/v1/sessions/{session_id}/activity` - Update session activity
- `PUT /api/v1/sessions/{session_id}/close` - Close session

## Example Usage

### Create a Device

```bash
curl -X POST http://localhost:8080/api/v1/devices \
  -H "Content-Type: application/json" \
  -d '{
    "endpoint_id": "device-001",
    "manufacturer": "Example Corp",
    "model": "Model X1",
    "software_version": "1.0.0",
    "status": "ONLINE"
  }'
```

### Get All Devices

```bash
curl http://localhost:8080/api/v1/devices?offset=0&limit=10
```

### Get Device Parameters

```bash
curl http://localhost:8080/api/v1/devices/1/parameters
```

### Health Check

```bash
curl http://localhost:8080/health
```

## Error Handling

The API Gateway provides consistent error responses:

```json
{
  "error": "Brief error description",
  "details": "Detailed error message"
}
```

Common HTTP status codes:
- `200` - Success
- `201` - Created
- `400` - Bad Request
- `404` - Not Found
- `500` - Internal Server Error
- `503` - Service Unavailable

## Development

### Building

```bash
go build -o build/api-gateway cmd/api-gateway/main.go
```

### Running

```bash
./build/api-gateway
```

### Running with Custom Port

```bash
PORT=8081 ./build/api-gateway
```

### Running with Custom Data Service Address

```bash
DATA_SERVICE_ADDR=localhost:9093 ./build/api-gateway
```

## Dependencies

- **Gin Framework**: HTTP web framework
- **gRPC**: Communication with data service
- **Protocol Buffers**: Message serialization
- **OpenUSP Internal Libraries**: gRPC client and data models

## Monitoring

The API Gateway provides built-in monitoring endpoints:

- `/health` - Simple health check
- `/status` - Detailed service status including data service connectivity

These endpoints can be used with monitoring systems like Prometheus, Kubernetes health checks, or load balancers.