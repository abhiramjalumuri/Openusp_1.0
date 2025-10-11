# OpenUSP API Documentation

This directory contains API documentation and specifications for the OpenUSP platform.

## Structure

- `openapi/` - OpenAPI/Swagger specifications
- `protobuf/` - Protocol Buffer API definitions (symlinked from pkg/proto)
- Working protocol agents in `cmd/usp-agent/` and `cmd/cwmp-agent/` directories

## API Endpoints

The OpenUSP platform provides the following APIs:

### REST API (External)
- Device Management API - CRUD operations for USP devices
- Parameter Management API - TR-181 parameter operations
- Alert Management API - System alerts and notifications
- Session Management API - Communication session tracking

### gRPC API (Internal)
- Data Service API - Internal database operations
- USP Core API - USP protocol processing
- MTP Service API - Message transport operations

## Documentation Access

- **Swagger UI**: http://localhost:6500/swagger/index.html (API Gateway)
- **API Documentation**: See [API Reference](../docs/API_REFERENCE.md) for detailed documentation