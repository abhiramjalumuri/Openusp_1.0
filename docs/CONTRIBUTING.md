# 🤝 Contributing to OpenUSP

Thank you for your interest in contributing to OpenUSP! This guide will help you get started with contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Contributing Guidelines](#contributing-guidelines)
- [Pull Request Process](#pull-request-process)
- [Coding Standards](#coding-standards)
- [Testing Requirements](#testing-requirements)
- [Documentation Standards](#documentation-standards)

## Code of Conduct

### Our Pledge

We as members, contributors, and leaders pledge to make participation in our community a harassment-free experience for everyone, regardless of age, body size, visible or invisible disability, ethnicity, sex characteristics, gender identity and expression, level of experience, education, socio-economic status, nationality, personal appearance, race, religion, or sexual identity and orientation.

### Our Standards

Examples of behavior that contributes to a positive environment:

- Using welcoming and inclusive language
- Being respectful of differing viewpoints and experiences
- Gracefully accepting constructive criticism
- Focusing on what is best for the community
- Showing empathy towards other community members

### Enforcement

Instances of abusive, harassing, or otherwise unacceptable behavior may be reported to the project maintainers. All complaints will be reviewed and investigated promptly and fairly.

## Getting Started

### Ways to Contribute

- **Bug Reports**: Report issues you encounter
- **Feature Requests**: Suggest new functionality
- **Code Contributions**: Fix bugs or implement features
- **Documentation**: Improve guides, examples, and API docs
- **Testing**: Help test releases and write test cases
- **Community Support**: Help other users in discussions

### Before You Start

1. **Check existing issues**: Look for similar bug reports or feature requests
2. **Read the documentation**: Familiarize yourself with the project architecture
3. **Join discussions**: Participate in GitHub Discussions or Issues
4. **Start small**: Begin with good first issues or documentation improvements

## Development Setup

### Prerequisites

- **Go 1.21+**: Primary development language
- **Docker & Docker Compose**: For infrastructure services
- **Git**: Version control
- **Make**: Build automation
- **PostgreSQL**: Database (can use Docker)

### Fork and Clone

```bash
# Fork the repository on GitHub, then clone your fork
git clone https://github.com/YOUR-USERNAME/openusp.git
cd openusp

# Add upstream remote
git remote add upstream https://github.com/original-org/openusp.git

# Verify remotes
git remote -v
```

### Environment Setup

```bash
# Install dependencies
go mod download

# Start infrastructure services
make infra-up

# Build all services
make build

# Run tests
make test

# Start services for development
make start
```

### Development Workflow

```bash
# Create feature branch
git checkout -b feature/your-feature-name

# Make your changes
# ... edit files ...

# Run tests and checks
make test
make lint
make format

# Commit changes
git add .
git commit -m "feat: add new feature description"

# Push to your fork
git push origin feature/your-feature-name

# Create pull request on GitHub
```

## Contributing Guidelines

### Types of Contributions

#### Bug Fixes
- **Small fixes**: Direct pull request is fine
- **Complex fixes**: Create issue first to discuss approach
- **Security fixes**: Email maintainers privately first

#### New Features
- **Always create an issue first** to discuss the feature
- **Include use cases** and rationale
- **Consider breaking changes** and migration path
- **Update documentation** as part of the implementation

#### Documentation
- **Improve clarity** of existing documentation
- **Add missing examples** or use cases
- **Fix typos** and formatting issues
- **Translate documentation** (if applicable)

### Issue Guidelines

When creating issues:

#### Bug Reports
```markdown
**Describe the bug**
A clear description of what the bug is.

**To Reproduce**
Steps to reproduce the behavior:
1. Start services with '...'
2. Send request to '...'
3. See error

**Expected behavior**
What you expected to happen.

**Environment:**
- OS: [e.g. Ubuntu 22.04]
- OpenUSP version: [e.g. v1.0.0]
- Go version: [e.g. 1.21.0]
- Docker version: [e.g. 24.0.0]

**Additional context**
Logs, screenshots, or other relevant information.
```

#### Feature Requests
```markdown
**Feature Description**
Clear description of the feature you'd like to see.

**Use Case**
Explain why this feature would be useful.

**Proposed Solution**
Describe how you envision this working.

**Alternatives Considered**
Other approaches you've considered.

**Additional Context**
Any other relevant information.
```

## Pull Request Process

### Before Submitting

1. **Rebase on latest main**:
   ```bash
   git fetch upstream
   git rebase upstream/main
   ```

2. **Run full test suite**:
   ```bash
   make test
   make lint
   make format
   ```

3. **Update documentation** if needed

4. **Add tests** for new functionality

### Pull Request Template

```markdown
## Description
Brief description of changes made.

## Type of Change
- [ ] Bug fix (non-breaking change which fixes an issue)
- [ ] New feature (non-breaking change which adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] Tests added/updated
- [ ] No breaking changes (or migration guide provided)
```

### Review Process

1. **Automated checks** must pass (CI/CD pipeline)
2. **At least one maintainer review** required
3. **Address feedback** and update PR as needed
4. **Squash commits** if requested
5. **Maintainer will merge** when approved

### After Merge

1. **Delete feature branch**:
   ```bash
   git branch -d feature/your-feature-name
   git push origin --delete feature/your-feature-name
   ```

2. **Update local main**:
   ```bash
   git checkout main
   git pull upstream main
   ```

## Coding Standards

### Go Code Style

Follow standard Go conventions:

#### Formatting
```bash
# Format code
go fmt ./...
goimports -w .

# Or use make target
make format
```

#### Naming Conventions
```go
// Good: Exported functions use PascalCase
func ProcessUSPMessage(msg *Message) error {
    // Good: Local variables use camelCase
    deviceID := msg.GetDeviceId()
    
    // Good: Constants use PascalCase or SCREAMING_SNAKE_CASE
    const MaxRetries = 3
    const DEFAULT_TIMEOUT = 30 * time.Second
    
    return nil
}

// Good: Interfaces end with -er when possible
type MessageProcessor interface {
    ProcessMessage(msg *Message) error
}

// Good: Struct names are clear and concise
type USPController struct {
    endpointID string
    version    string
}
```

#### Error Handling
```go
// Good: Wrap errors with context
func GetDevice(id string) (*Device, error) {
    device, err := db.FindDevice(id)
    if err != nil {
        return nil, fmt.Errorf("failed to get device %s: %w", id, err)
    }
    
    if device == nil {
        return nil, fmt.Errorf("device %s not found", id)
    }
    
    return device, nil
}

// Good: Use specific error types when helpful
type ValidationError struct {
    Field   string
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation error for field %s: %s", e.Field, e.Message)
}
```

#### Comments
```go
// Package documentation
// Package usp provides USP protocol implementation for TR-369 compliance.
// It supports both USP version 1.3 and 1.4 with automatic version detection.
package usp

// Function documentation for exported functions
// ProcessUSPRecord parses and processes a USP record according to TR-369 specification.
// It automatically detects the USP version and routes the message appropriately.
//
// Parameters:
//   - record: Raw USP record bytes
//   - fromID: Sender endpoint ID
//
// Returns:
//   - Processed USP response record
//   - Error if parsing or processing fails
func ProcessUSPRecord(record []byte, fromID string) (*Record, error) {
    // Implementation...
}
```

### API Design

#### REST API Standards
```go
// Good: RESTful resource naming
// GET    /api/v1/devices
// GET    /api/v1/devices/{id}
// POST   /api/v1/devices
// PUT    /api/v1/devices/{id}
// DELETE /api/v1/devices/{id}

// Good: Consistent response format
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   string      `json:"error,omitempty"`
    Meta    *Meta       `json:"meta,omitempty"`
}

// Good: Use appropriate HTTP status codes
func CreateDevice(c *gin.Context) {
    var device Device
    if err := c.ShouldBindJSON(&device); err != nil {
        c.JSON(http.StatusBadRequest, APIResponse{
            Success: false,
            Error:   "invalid request body",
        })
        return
    }
    
    // ... create device ...
    
    c.JSON(http.StatusCreated, APIResponse{
        Success: true,
        Data:    device,
    })
}
```

#### gRPC API Standards
```protobuf
// Good: Clear service and method names
service DataService {
    rpc GetDevice(GetDeviceRequest) returns (GetDeviceResponse);
    rpc ListDevices(ListDevicesRequest) returns (ListDevicesResponse);
    rpc CreateDevice(CreateDeviceRequest) returns (CreateDeviceResponse);
}

// Good: Consistent request/response patterns
message GetDeviceRequest {
    string device_id = 1;
}

message GetDeviceResponse {
    Device device = 1;
    Error error = 2;
}

// Good: Include pagination for list operations
message ListDevicesRequest {
    int32 page_size = 1;
    string page_token = 2;
    string filter = 3;
}

message ListDevicesResponse {
    repeated Device devices = 1;
    string next_page_token = 2;
    int32 total_size = 3;
}
```

### Database Standards

#### Model Definitions
```go
// Good: Use GORM tags consistently
type Device struct {
    ID                 uint      `gorm:"primarykey" json:"id"`
    DeviceID          string    `gorm:"uniqueIndex;not null" json:"device_id"`
    EndpointID        string    `gorm:"index;not null" json:"endpoint_id"`
    OUI               string    `gorm:"size:6" json:"oui"`
    ProductClass      string    `gorm:"size:255" json:"product_class"`
    SerialNumber      string    `gorm:"size:255;not null" json:"serial_number"`
    HardwareVersion   string    `gorm:"size:255" json:"hardware_version"`
    SoftwareVersion   string    `gorm:"size:255" json:"software_version"`
    Manufacturer      string    `gorm:"size:255" json:"manufacturer"`
    ModelName         string    `gorm:"size:255" json:"model_name"`
    Description       string    `gorm:"size:500" json:"description"`
    DeviceType        string    `gorm:"size:255" json:"device_type"`
    ProtocolVersion   string    `gorm:"size:10" json:"protocol_version"`
    SupportedProtocols []string `gorm:"type:text[]" json:"supported_protocols"`
    LastContactTime   *time.Time `json:"last_contact_time"`
    CreatedAt         time.Time `json:"created_at"`
    UpdatedAt         time.Time `json:"updated_at"`
}
```

#### Repository Pattern
```go
// Good: Use repository pattern for database operations
type DeviceRepository interface {
    GetByID(id string) (*Device, error)
    GetByEndpointID(endpointID string) (*Device, error)
    List(limit, offset int, filter string) ([]*Device, error)
    Create(device *Device) error
    Update(device *Device) error
    Delete(id string) error
}

type deviceRepository struct {
    db *gorm.DB
}

func (r *deviceRepository) GetByID(id string) (*Device, error) {
    var device Device
    err := r.db.Where("device_id = ?", id).First(&device).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, nil
        }
        return nil, fmt.Errorf("failed to get device: %w", err)
    }
    return &device, nil
}
```

## Testing Requirements

### Unit Tests

```go
// Good: Table-driven tests
func TestValidateUSPRecord(t *testing.T) {
    tests := []struct {
        name    string
        record  []byte
        want    bool
        wantErr bool
    }{
        {
            name:    "valid USP 1.4 record",
            record:  validUSP14Record,
            want:    true,
            wantErr: false,
        },
        {
            name:    "invalid record format",
            record:  []byte("invalid"),
            want:    false,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ValidateUSPRecord(tt.record)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidateUSPRecord() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if got != tt.want {
                t.Errorf("ValidateUSPRecord() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Integration Tests

```go
//go:build integration

func TestDeviceAPIIntegration(t *testing.T) {
    // Setup test database
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)
    
    // Start test server
    server := setupTestServer(t, db)
    defer server.Close()
    
    // Test create device
    device := &Device{
        EndpointID:   "test-device-001",
        OUI:          "123456",
        SerialNumber: "SN123",
    }
    
    resp, err := http.Post(server.URL+"/api/v1/devices", "application/json", 
        bytes.NewBuffer(mustMarshalJSON(device)))
    require.NoError(t, err)
    require.Equal(t, http.StatusCreated, resp.StatusCode)
    
    // Test get device
    resp, err = http.Get(server.URL + "/api/v1/devices/test-device-001")
    require.NoError(t, err)
    require.Equal(t, http.StatusOK, resp.StatusCode)
}
```

### Test Coverage

- **Minimum coverage**: 80% for new code
- **Run coverage**: `make test-coverage`
- **View coverage**: `make coverage-html`

```bash
# Check coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Documentation Standards

### Code Documentation

```go
// Package documentation should explain purpose and usage
// Package config provides configuration management for OpenUSP services.
// It supports environment variables, configuration files, and runtime overrides
// with automatic validation and type conversion.
//
// Basic usage:
//   config := config.LoadDeploymentConfig("api-gateway", "production", 8080)
//   fmt.Printf("Database DSN: %s", config.GetDatabaseDSN())
//
// Environment variables are automatically loaded and validated.
// See CONFIGURATION.md for complete environment variable reference.
package config

// Function documentation for exported functions
// LoadDeploymentConfig loads and validates configuration for a service.
// It combines environment variables, default values, and runtime parameters
// to create a complete configuration object.
//
// Parameters:
//   - serviceName: Name of the service (used for logging and metrics)
//   - environment: Deployment environment (dev, test, prod)
//   - defaultPort: Default port if not specified in environment
//
// Returns:
//   - Fully populated configuration object
//   - Validation error if configuration is invalid
//
// Environment variables:
//   - CONSUL_ENABLED: Enable service discovery (default: true)
//   - OPENUSP_DATABASE_HOST: Database hostname (required)
//   - OPENUSP_DATABASE_PORT: Database port (default: 5432)
//
// Example:
//   config := LoadDeploymentConfig("api-gateway", "production", 8080)
//   if config.ConsulEnabled {
//       // Initialize service discovery
//   }
func LoadDeploymentConfig(serviceName, environment string, defaultPort int) *DeploymentConfig {
    // Implementation...
}
```

### README and User Documentation

- **Use clear headings** and table of contents
- **Include code examples** for common use cases
- **Provide troubleshooting** information
- **Keep examples up to date** with current API
- **Use consistent formatting** and style

### API Documentation

- **Document all public APIs** with examples
- **Include error responses** and status codes
- **Provide curl examples** for REST endpoints
- **Document rate limits** and authentication
- **Keep OpenAPI specs** up to date

## Release Process

### Versioning

We use [Semantic Versioning](https://semver.org/):

- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Release Checklist

1. **Update version** in relevant files
2. **Update CHANGELOG.md** with release notes
3. **Run full test suite** including integration tests
4. **Update documentation** if needed
5. **Create release tag** and GitHub release
6. **Update Docker images** if applicable

## Community Guidelines

### Communication

- **Be respectful** and professional
- **Use clear, descriptive titles** for issues and PRs
- **Provide context** and examples when asking for help
- **Search existing issues** before creating new ones
- **Use appropriate labels** for issues and PRs

### Recognition

Contributors are recognized in:
- **CONTRIBUTORS.md** file
- **Release notes** for significant contributions
- **GitHub contributor stats**
- **Special mentions** in documentation updates

## Getting Help

### Resources

- **Documentation**: Check the `docs/` directory
- **Examples**: Look at the `examples/` directory
- **Issues**: Search existing GitHub issues
- **Discussions**: Join GitHub Discussions for questions

### Contact

- **General questions**: GitHub Discussions
- **Bug reports**: GitHub Issues
- **Security issues**: Email maintainers privately
- **Feature requests**: GitHub Issues with feature template

---

Thank you for contributing to OpenUSP! Your contributions help make TR-369 device management more accessible and reliable for everyone. 🚀