# 🔧 Development Guide

Set up your development environment and learn how to contribute to OpenUSP.

## Development Environment Setup

### Prerequisites

- **Go 1.21+**: For building and running services
- **Docker & Docker Compose**: For infrastructure services
- **Git**: Version control
- **Make**: Build automation
- **IDE**: VS Code, GoLand, or your preferred editor

### Recommended Tools

```bash
# Install additional development tools
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/go-delve/delve/cmd/dlv@latest  # Debugger
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### IDE Configuration

#### VS Code Extensions
- Go (official)
- Docker
- Protocol Buffers
- REST Client

#### Settings (`.vscode/settings.json`)
```json
{
    "go.toolsManagement.checkForUpdates": "local",
    "go.useLanguageServer": true,
    "go.formatTool": "goimports",
    "go.lintTool": "golangci-lint",
    "go.testFlags": ["-v"],
    "go.coverOnSave": true
}
```

## Project Structure

```
openusp/
├── cmd/                    # Service entry points
│   ├── api-gateway/       # REST API Gateway
│   ├── data-service/      # Database service
│   ├── mtp-service/       # Message transport
│   ├── usp-service/       # USP protocol engine
│   ├── cwmp-service/      # CWMP compatibility
│   ├── usp-agent/         # TR-369 USP agent
│   └── cwmp-agent/        # TR-069 CWMP agent
├── internal/              # Private application code
│   ├── database/          # Database layer
│   ├── grpc/             # gRPC implementations
│   ├── mtp/              # Transport protocols
│   ├── usp/              # USP protocol handling
│   └── cwmp/             # CWMP protocol handling
├── pkg/                   # Public library code
│   ├── config/           # Configuration management
│   ├── consul/           # Service discovery
│   ├── metrics/          # Monitoring
│   ├── proto/            # Protocol buffers
│   └── version/          # Version management

├── configs/              # Configuration files
├── deployments/          # Deployment configs
└── docs/                # Documentation
```

## Development Workflow

### 1. Fork and Clone

```bash
# Fork the repository on GitHub, then:
git clone https://github.com/YOUR-USERNAME/openusp.git
cd openusp
git remote add upstream https://github.com/original-org/openusp.git
```

### 2. Set Up Environment

```bash
# Install dependencies
go mod download

# Start infrastructure services
make infra-up

# Verify setup
make version
```

### 3. Development Commands

```bash
# Build all services and agents
make build-all

# Build individual service
make build-api-gateway

# Build agents
make build-usp-agent
make build-cwmp-agent

# Run tests
make test

# Format code
make fmt

# Lint code  
make lint

# Run all quality checks
make go-check
```

### 4. Running Services for Development

```bash
# Start all services
make start-all

# Or start individual services in separate terminals
make start-data-service
make start-api-gateway
make start-mtp-service
make start-usp-service
make start-cwmp-service

# Start protocol agents
make start-usp-agent
make start-cwmp-agent

# View logs for specific services
make logs-api-gateway
# Or tail specific service logs manually
tail -f logs/api-gateway.log
```

## Development Patterns

### Adding a New Service

1. **Create service directory**:
```bash
mkdir -p cmd/new-service
mkdir -p internal/new-service
```

2. **Create main.go**:
```go
package main

import (
    "log"
    "openusp/pkg/config"
    "openusp/pkg/service"
    "openusp/pkg/version"
)

func main() {
    log.Printf("🚀 Starting New Service...")
    
    // Service configuration
    opts := service.ServiceOptions{
        RequiresHTTP:     true,
        DefaultHTTPPort:  8090,
    }
    
    manager, err := service.NewServiceManager("new-service", "utility", opts)
    if err != nil {
        log.Fatal(err)
    }
    
    // Start service
    if err := manager.Start(); err != nil {
        log.Fatal(err)
    }
}
```

3. **Add to Makefile**:
```makefile
build-new-service: $(BINARY_DIR)
	@echo "Building New Service..."
	@go build $(LDFLAGS) -o $(BINARY_DIR)/new-service cmd/new-service/main.go
	@echo "✅ New Service built -> $(BINARY_DIR)/new-service"
```

### Adding gRPC Service

1. **Define Protocol Buffer**:
```protobuf
// pkg/proto/newservice/newservice.proto
syntax = "proto3";

package newservice;
option go_package = "openusp/pkg/proto/newservice";

service NewService {
    rpc GetStatus(StatusRequest) returns (StatusResponse);
}

message StatusRequest {
    string service_id = 1;
}

message StatusResponse {
    string status = 1;
    int64 timestamp = 2;
}
```

2. **Generate Code**:
```bash
protoc --go_out=. --go-grpc_out=. pkg/proto/newservice/newservice.proto
```

3. **Implement Server**:
```go
// internal/grpc/newservice_server.go
package grpc

import (
    "context"
    pb "openusp/pkg/proto/newservice"
)

type NewServiceServer struct {
    pb.UnimplementedNewServiceServer
}

func (s *NewServiceServer) GetStatus(ctx context.Context, req *pb.StatusRequest) (*pb.StatusResponse, error) {
    return &pb.StatusResponse{
        Status:    "healthy",
        Timestamp: time.Now().Unix(),
    }, nil
}
```

### Adding REST Endpoint

```go
// In api-gateway service
func setupRoutes(r *gin.Engine) {
    v1 := r.Group("/api/v1")
    {
        // Existing routes...
        
        v1.GET("/new-endpoint", handleNewEndpoint)
        v1.POST("/new-endpoint", createNewResource)
    }
}

func handleNewEndpoint(c *gin.Context) {
    c.JSON(200, gin.H{
        "message": "New endpoint",
        "timestamp": time.Now().Unix(),
    })
}
```

## Testing

### Unit Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Test specific package
go test ./internal/database/

# Verbose output
go test -v ./...
```

### Integration Tests

```bash
# Run integration tests (requires infrastructure)
make infra-up
go test -tags=integration ./test/integration/

# Or use make target
make test-integration
```

### End-to-End Tests

```bash
# Start full system
make start

# Run E2E tests
go test ./test/e2e/

# Using example clients
make run-examples
```

### Writing Tests

```go
// Example unit test
func TestConfigLoading(t *testing.T) {
    config := config.LoadDeploymentConfig("test-service", "test", 8080)
    
    assert.NotNil(t, config)
    assert.Equal(t, "test-service", config.ServiceName)
    assert.Equal(t, 8080, config.ServicePort)
}

// Example integration test
//go:build integration
func TestDatabaseConnection(t *testing.T) {
    db, err := database.NewDatabase(testConfig.GetDatabaseDSN())
    require.NoError(t, err)
    defer db.Close()
    
    // Test operations
    device := &database.Device{
        DeviceID: "test-device",
        EndpointID: "test-endpoint",
    }
    
    err = db.CreateDevice(device)
    assert.NoError(t, err)
}
```

## Debugging

### Local Debugging

```bash
# Debug specific service
dlv debug cmd/api-gateway/main.go

# Debug with arguments
dlv debug cmd/data-service/main.go -- --port 9090

# Attach to running process
dlv attach $(pgrep data-service)
```

### VS Code Debugging

Create `.vscode/launch.json`:
```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Debug API Gateway",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/cmd/api-gateway/main.go",
            "env": {
                "OPENUSP_API_GATEWAY_PORT": "6500"
            }
        }
    ]
}
```

### Logging

Use structured logging:
```go
import "log/slog"

// Service startup
slog.Info("Starting service", 
    "service", "api-gateway",
    "port", config.ServicePort,
    "version", version.Version)

// Error handling
slog.Error("Database connection failed",
    "error", err,
    "dsn", config.GetDatabaseDSN())

// Request processing
slog.Debug("Processing request",
    "method", r.Method,
    "path", r.URL.Path,
    "user_agent", r.UserAgent())
```

## Code Style and Standards

### Go Code Style

Follow standard Go conventions:
- Use `gofmt` and `goimports`
- Follow effective Go guidelines
- Use meaningful variable names
- Add comments for exported functions
- Handle errors explicitly

### Error Handling

```go
// Good: Explicit error handling
func processDevice(deviceID string) (*Device, error) {
    device, err := db.GetDevice(deviceID)
    if err != nil {
        return nil, fmt.Errorf("failed to get device %s: %w", deviceID, err)
    }
    
    if device == nil {
        return nil, fmt.Errorf("device %s not found", deviceID)
    }
    
    return device, nil
}

// Usage
device, err := processDevice("device001")
if err != nil {
    slog.Error("Device processing failed", "error", err)
    return err
}
```

### API Design

```go
// RESTful endpoint naming
GET    /api/v1/devices          // List devices
GET    /api/v1/devices/{id}     // Get device
POST   /api/v1/devices          // Create device
PUT    /api/v1/devices/{id}     // Update device
DELETE /api/v1/devices/{id}     // Delete device

// Consistent response format
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
    Meta    *Meta       `json:"meta,omitempty"`
}
```

## Performance Considerations

### Database Optimization

```go
// Use connection pooling
db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
sqlDB, err := db.DB()
sqlDB.SetMaxIdleConns(10)
sqlDB.SetMaxOpenConns(100)
sqlDB.SetConnMaxLifetime(time.Hour)

// Use indices
type Device struct {
    DeviceID   string `gorm:"index:idx_device_id"`
    EndpointID string `gorm:"index:idx_endpoint_id"`
}

// Batch operations
var devices []Device
db.CreateInBatches(devices, 100)
```

### gRPC Optimization

```go
// Use connection pooling
conn, err := grpc.Dial(addr, 
    grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(4*1024*1024)),
    grpc.WithKeepaliveParams(keepalive.ClientParameters{
        Time:                10 * time.Second,
        Timeout:             time.Second,
        PermitWithoutStream: true,
    }))
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed contribution guidelines.

### Pull Request Process

1. Create feature branch: `git checkout -b feature/new-feature`
2. Make changes and add tests
3. Run checks: `make check`
4. Commit with descriptive message
5. Push and create pull request
6. Address review feedback

### Commit Messages

Use conventional commits:
```
feat: add device discovery endpoint
fix: resolve parameter validation issue
docs: update API documentation
test: add integration tests for MTP service
refactor: improve error handling in USP parser
```

## Getting Help

- **Documentation**: Check relevant guides in `docs/`
- **Code Examples**: Look at the working protocol agents in `cmd/usp-agent/` and `cmd/cwmp-agent/`
- **GitHub Issues**: Ask questions or report bugs
- **Code Review**: Submit PRs for feedback
- **Discord/Slack**: Community channels (if available)

Ready to contribute to OpenUSP! 🚀