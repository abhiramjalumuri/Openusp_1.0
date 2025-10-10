# OpenUSP gRPC Services - Static Configuration

## Overview

With the removal of Consul service discovery, OpenUSP now uses static port configuration for gRPC communication between services.

## Static gRPC Port Assignments

| Service            | Main Port | Health Port | gRPC Port | Purpose                          |
|-------------------|-----------|-------------|-----------|----------------------------------|
| api-gateway       | 6500      | 6501        | 6502      | External API, service coordination|
| data-service      | 6100      | 6101        | 6102      | Database operations              |
| connection-manager| 6200      | 6201        | 6202      | Service discovery coordination   |
| usp-service       | 6400      | 6401        | 6402      | USP protocol processing          |
| cwmp-service      | 7547      | 7548        | 7549      | CWMP/TR-069 protocol            |
| mtp-service       | 8081      | 8082        | 8083      | Message transport protocol       |

## gRPC Service Communication

### Current Inter-Service Communication:

1. **MTP Service → USP Service**
   - MTP service forwards USP messages to USP service via gRPC
   - Static target: `localhost:6402`

2. **USP Service → Data Service** 
   - USP service queries/updates device data via gRPC
   - Static target: `localhost:6102`

3. **API Gateway → All Services**
   - API Gateway coordinates with services for REST API operations
   - Uses static service configuration for routing

## Configuration

Services use the static configuration from `configs/services.yaml`:

```yaml
services:
  usp-service:
    port: 6400
    health_port: 6401
    grpc_port: 6402
    protocol: http
```

## Implementation

### Service Startup
Services start their gRPC servers on their assigned static ports:

```go
// Example: USP Service
grpcPort := 6402
lis, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
grpcServer := grpc.NewServer()
// Register service implementations
grpcServer.Serve(lis)
```

### Client Connection
Services connect to other services using static ports:

```go
// Example: MTP Service connecting to USP Service
target := "localhost:6402"
conn, err := grpc.Dial(target, grpc.WithInsecure())
uspClient := uspservice.NewUSPServiceClient(conn)
```

## Benefits of Static Configuration

✅ **Predictable**: No dynamic port allocation
✅ **Simple**: No service discovery complexity  
✅ **Reliable**: No dependency on external service registry
✅ **Debuggable**: Easy to test individual service connections
✅ **Fast**: No service lookup overhead

## Migration Notes

- Removed all Consul-based service discovery
- Services now use hardcoded static ports
- gRPC connections established directly using `localhost:port`
- Service health checks use dedicated health ports
- Monitoring tools (Prometheus) use static target configuration