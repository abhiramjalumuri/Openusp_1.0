# OpenUSP - TR-369 User Service Platform

## Project Overview
This is a cloud-native microservice implementation of the Broadband Forum's TR-369 based User Service Platform (USP). The platform provides integrated service suite with built-in features for remote device management, lifecycle management, and network management.

## Key Features
- **TR-369 Compliance**: Full implementation of the USP standard for device management and communication
- **Multi-Protocol Support**: STOMP, MQTT, WebSocket, and Unix Domain Socket MTPs (Message Transfer Protocols)
- **TR-181 Device Model**: Complete support for the standardized device data model (Device:2 v2.19.1)
- **Advanced USP Parsing**: Comprehensive USP Protocol Buffers parsing and validation for versions 1.3 and 1.4
- **Dual Version Support**: Native support for both USP 1.3 and 1.4 with automatic version detection
- **Device Lifecycle Management**: Agent/device onboarding, provisioning, and management
- **Scalable Database Layer**: Multi-device data storage based on TR-181 data model hierarchy
- **RESTful Management API**: Standard interfaces for device configuration and monitoring via API Gateway
- **Complete USP Operations**: Full TR-369 command set (Get, Set, Add, Delete, Operate, Notify, GetSupportedDM, GetInstances)
- **Message Transport Layer**: Enterprise-grade MTP service with comprehensive USP message processing
- **CWMP (TR-069) Support**: Backward compatibility with existing management systems
- **Swagger UI Integration**: Interactive API documentation and testing interface for all REST endpoints
- **Working Protocol Agents**: Complete TR-369 and TR-069 agents with onboarding functionality
- **Development Infrastructure**: Comprehensive development scripts and Docker-based infrastructure
- **Service Discovery**: Consul-enabled microservice architecture with automatic service registration
- **Unified Configuration**: Single binary architecture with runtime Consul flag support

## Architecture Principles
1. **Single Binary Architecture**: All services are unified binaries with runtime --consul flag support
2. **Modern Microservices Architecture**: Clean separation between REST API Gateway and gRPC backend services
3. **API Gateway Pattern**: Dedicated REST API Gateway (Gin framework) that proxies all requests to internal gRPC services
4. **gRPC Internal Communication**: All inter-service communication uses Protocol Buffers and gRPC for type safety and performance
5. **Data Service Layer**: Dedicated gRPC data service for all database operations with PostgreSQL backend
6. **Enhanced MTP Service**: Full-featured message transport with comprehensive USP parsing for MQTT, STOMP, WebSocket, Unix Domain Socket
7. **Advanced Message Processing**: USP version 1.3 and 1.4 protocol buffer parsing with automatic version detection and validation
8. **USP Core Service**: Device discovery, onboarding, registration per TR-369 standards with dual version support
9. **Multi-Controller Support**: Support for multiple USP controllers with proper message routing
10. **TR-181 Integration**: Complete Device:2 data model integration with namespace management
11. **Clean API Design**: RESTful endpoints for external clients, gRPC for internal microservice communication
12. **Service Discovery**: Consul-enabled by default for development with runtime configuration
13. **Unified Configuration**: Environment-driven configuration system with comprehensive defaults
14. **Observability**: Prometheus metrics, structured logging, health checks, and comprehensive error handling
15. **Protocol Compliance**: Full TR-369 specification compliance with comprehensive USP operation support
16. **CWMP Service**: Separate service for TR-069 backward compatibility

## Technology Stack
- **Language**: Go (Golang) 1.24+
- **Message Brokers**: RabbitMQ (STOMP), Mosquitto (MQTT), WebSocket (Gorilla), Unix Domain Socket
- **Database**: PostgreSQL with TR-181 schema support
- **Web Framework**: Gin (API Gateway), Gorilla WebSocket (MTP service)
- **Protocol Buffers**: USP v1.3 and v1.4 with comprehensive parsing
- **Inter-service Communication**: gRPC (internal), REST API (external via Gin)
- **Monitoring**: Prometheus metrics with health check endpoints
- **Testing**: Broadband Forum's obuspa agent (https://github.com/BroadbandForum/obuspa)
- **TR-181 Integration**: Device:2 v2.19.1 data model from usp.technology

## Project Structure Guidelines
```
openusp/
├── cmd/                       # Application entry points (single binaries with --consul flag)
│   ├── api-gateway/          # Modern REST API Gateway service (Gin-based, gRPC backend)
│   │   ├── main.go           # Unified binary with runtime Consul configuration
│   │   └── README.md         # Complete API documentation and usage examples
│   ├── data-service/         # gRPC Data Service for all database operations
│   │   └── main.go           # Unified binary with PostgreSQL backend and versioning
│   ├── usp-service/          # Main USP core service with dual version support
│   │   └── main.go           # Unified binary with TR-369 compliance engine
│   ├── mtp-service/          # Enhanced Message Transfer Protocol service with USP parsing
│   │   └── main.go           # Unified binary with multi-transport support
│   └── cwmp-service/         # CWMP/TR-069 service for backward compatibility
│       └── main.go           # Unified binary with SOAP/XML processing
├── internal/                 # Private application code  
│   ├── usp/                  # Comprehensive USP protocol implementation with parser
│   │   └── parser.go         # Advanced USP 1.3/1.4 record and message parser
│   ├── mtp/                  # MTP protocol implementations (MQTT, STOMP, WebSocket, Unix)
│   ├── tr181/                # TR-181 Device:2 data model implementation
│   │   ├── datamodel.go      # TR-181 data model structures
│   │   └── manager.go        # Data model management and validation
│   ├── database/             # Database layer with TR-181 schema and GORM models
│   │   ├── database.go       # PostgreSQL connection and initialization
│   │   ├── models.go         # GORM database models for devices, parameters, alerts, sessions
│   │   ├── repository.go     # Database repository layer with CRUD operations
│   │   └── converters.go     # Bidirectional conversion between database models and gRPC messages
│   ├── grpc/                 # gRPC service implementations
│   │   ├── dataservice_server.go  # gRPC server implementation for data service
│   │   └── dataservice_client.go  # gRPC client wrapper for API Gateway
│   ├── cwmp/                 # CWMP/TR-069 protocol implementation
│   │   ├── processor.go      # SOAP/XML message processing
│   │   └── onboarding.go     # TR-069 device onboarding functionality
│   └── test_utils.go         # Shared testing utilities
├── pkg/                      # Public library code
│   ├── config/               # Configuration management
│   │   └── deployment.go     # Unified configuration system with environment variables
│   ├── service/              # Service lifecycle management
│   │   └── manager.go        # Service manager with Consul integration
│   ├── consul/               # Service discovery
│   │   └── client.go         # Consul client wrapper
│   ├── metrics/              # Monitoring and observability
│   │   └── prometheus.go     # Prometheus metrics collection
│   ├── proto/                # Protocol buffer definitions
│   │   ├── dataservice/      # Data service gRPC protocol definitions
│   │   │   ├── dataservice.proto        # gRPC service and message definitions
│   │   │   ├── dataservice.pb.go        # Generated Protocol Buffer code
│   │   │   └── dataservice_grpc.pb.go   # Generated gRPC service code
│   │   ├── v1_3/            # USP 1.3 protocol buffers
│   │   └── v1_4/            # USP 1.4 protocol buffers  
│   ├── datamodel/           # TR-181 XML schemas and types
│   │   ├── tr-181-2-19-1-usp-full.xml  # Official TR-181 Device:2 schema
│   │   └── tr-106-types.xml             # TR-106 data types
│   └── version/             # Version management
│       └── version.go       # Application version information
├── examples/                # Working protocol agents (onboarding functionality only)
│   ├── tr369-agent/         # TR-369 USP agent with onboarding
│   │   ├── main.go          # Complete WebSocket-based USP agent
│   │   └── README.md        # Agent usage and configuration
│   ├── tr069-agent/         # TR-069 agent with onboarding
│   │   └── main.go          # Complete SOAP/XML CWMP agent with onboarding
│   └── README.md            # Examples overview and usage
├── test/                    # Test files and demonstrations
│   ├── usp_parsing_demo.go  # USP parsing functionality demonstration
│   ├── consul_demo.go       # Consul service discovery demonstration
│   ├── consul_debug.go      # Consul debugging utilities
│   └── README.md            # Testing documentation
├── configs/                 # Configuration files
│   ├── openusp.env          # Main environment configuration (Consul enabled by default)
│   ├── version.env          # Version information
│   ├── init-db.sql          # Database initialization
│   ├── mosquitto.conf       # MQTT broker configuration
│   ├── prometheus-dev.yml   # Prometheus configuration
│   ├── grafana-datasources.yml     # Grafana data sources
│   ├── grafana-dashboard-provider.yml  # Grafana dashboard provider
│   ├── rabbitmq-plugins     # RabbitMQ plugins configuration
│   └── grafana-dashboards/  # Grafana dashboard definitions
├── deployments/             # Deployment configurations
│   ├── docker-compose.infra.yml   # Infrastructure services (PostgreSQL, Consul, RabbitMQ, MQTT, Grafana, Prometheus)
│   ├── deploy-compose.sh    # Docker Compose deployment script
│   ├── deploy-k8s.sh        # Kubernetes deployment script
│   ├── docker/              # Individual Dockerfiles
│   ├── environments/        # Environment-specific configurations
│   ├── kubernetes/          # Kubernetes manifests
│   └── README.md            # Deployment documentation
├── scripts/                 # Build and development scripts
│   ├── start-dev-with-swagger.sh   # Development startup with Swagger UI
│   ├── swagger-demo.sh      # Swagger UI demonstration
│   ├── setup-grafana.sh     # Grafana setup script
│   ├── test-database.sh     # Database testing script
│   ├── test-infrastructure.sh  # Infrastructure testing
│   └── test-versions.sh     # Version validation script
├── docs/                    # Comprehensive user-focused documentation
│   ├── README.md            # Project overview and quick start
│   ├── QUICKSTART.md        # 5-minute quick start guide
│   ├── INSTALLATION.md      # Installation guide for multiple platforms
│   ├── USER_GUIDE.md        # Detailed user guide for device management
│   ├── API_REFERENCE.md     # Complete API documentation
│   ├── CONFIGURATION.md     # Configuration reference
│   ├── DEPLOYMENT.md        # Production deployment guide
│   ├── DEVELOPMENT.md       # Development environment setup
│   ├── TROUBLESHOOTING.md   # Troubleshooting guide
│   └── CONTRIBUTING.md      # Contribution guidelines
├── logs/                    # Service logs directory
├── build/                   # Compiled binaries
│   ├── api-gateway          # API Gateway binary
│   ├── data-service         # Data Service binary
│   ├── mtp-service          # MTP Service binary
│   ├── usp-service          # USP Service binary
│   ├── cwmp-service         # CWMP Service binary
│   ├── tr369-agent          # TR-369 agent binary
│   └── tr069-agent          # TR-069 agent binary
└── Makefile                 # Modern build system with standard targets
├── deployments/             # Docker, Kubernetes configs
├── scripts/                 # Build and deployment scripts
└── docs/                    # Documentation and specifications
```

## Coding Standards
- Follow standard Go formatting (gofmt, goimports)
- Use descriptive names following Go conventions
- Implement proper error handling (return errors, avoid panics)
- Add comprehensive comments for exported functions
- Use context.Context for cancellation and timeouts
- Implement structured logging with appropriate log levels
- Write unit tests with table-driven test patterns
- Use interfaces for better testability and decoupling

## TR-369/USP Specific Guidelines
- Follow TR-369 specification strictly for USP message handling
- Implement comprehensive USP record and message parsing for both v1.3 and v1.4
- Support automatic version detection between USP 1.3 and 1.4 protocols
- Implement proper USP endpoint validation and security
- Support all required USP operations (Get, Set, Add, Delete, Operate, Notify, GetSupportedDM, GetInstances)
- Handle USP error codes and responses according to specification with detailed logging
- Implement proper TR-181 parameter path validation using Device:2 v2.19.1 schema
- Support USP subscription and notification mechanisms
- Provide comprehensive message validation and error reporting
- Support all session context types (NoSessionContext, SessionContext, WebSocketConnect)
- Implement proper protocol buffer marshaling/unmarshaling for both versions

## Development Workflow
- Use Protocol Buffers for USP message definitions with dual version support (v1.3 and v1.4)
- Implement comprehensive USP parsing with automatic version detection
- Implement gRPC services with proper error handling and validation
- Create Docker containers for each microservice with health checks
- Use environment variables for configuration management
- Implement health checks and status endpoints for all services
- Add comprehensive metrics and observability from the start with detailed logging
- Test with obuspa agent for TR-369 compliance validation
- Use structured logging with operation-specific details for USP message processing
- Implement proper error collection and reporting in USP parser
- Validate parsed USP records and messages according to TR-369 specification

## Current Implementation Status

### ✅ **Completed Components**

1. **API Gateway Service** (`cmd/api-gateway/main.go`)
   - Modern REST API Gateway with comprehensive endpoints
   - Complete device management API (CRUD operations)
   - Parameter, alert, and session management endpoints
   - gRPC client integration for all backend operations
   - Health check and status endpoints with data service connectivity
   - Clean error handling and JSON responses
   - CORS support and proper HTTP status codes
   - **Swagger UI Integration**: Complete interactive API documentation at `/swagger/index.html`
   - Comprehensive OpenAPI 2.0 specification with all endpoints and data models
   - **Unified Binary**: Single binary with --consul flag and runtime configuration

2. **Data Service** (`cmd/data-service/main.go`)
   - Complete gRPC server implementation for database operations
   - Full CRUD operations for devices, parameters, alerts, sessions
   - PostgreSQL backend with GORM ORM integration
   - Bidirectional model conversion (database ↔ gRPC)
   - Health checks and status reporting
   - Graceful shutdown and connection management
   - **Unified Binary**: Single binary with --consul flag and version support

3. **MTP Service** (`cmd/mtp-service/main.go`)
   - Comprehensive USP record and message parsing for v1.3 and v1.4
   - Multi-protocol transport: MQTT, STOMP, WebSocket, Unix Domain Socket
   - Demo UI with WebSocket testing at `http://localhost:8081/usp`
   - Health check endpoint at `http://localhost:8081/health`
   - Automatic version detection and message validation
   - Upon receiving a USP message, the service detects the version (1.3 or 1.4) and processes it accordingly
   - Generates appropriate USP responses based on the operation and version
   - Updates device data (TR-181 datamodel, objects, parameters, alerts, sessions) via Data Service gRPC calls
   - **Unified Binary**: Single binary with --consul flag and runtime configuration

4. **CWMP Service** (`cmd/cwmp-service/main.go`)
   - Complete TR-069 protocol implementation for backward compatibility
   - SOAP/XML message processing with authentication
   - Full CWMP RPC methods: GetRPCMethods, GetParameterNames, GetParameterValues, SetParameterValues
   - TR-181 data model integration for parameter management
   - Session management with configurable timeouts
   - Health check endpoint at `http://localhost:7547/health`
   - Flexible SOAP parser supporting both `<soap:Envelope>` and `<Envelope>` formats
   - **Unified Binary**: Single binary with --consul flag and runtime configuration

5. **USP Core Service** (`cmd/usp-service/main.go`)
   - Dual USP version processing engine
   - TR-181 Device:2 data model integration
   - Device management and lifecycle support
   - **Unified Binary**: Single binary with --consul flag and runtime configuration

5. **Advanced USP Parser** (`internal/usp/parser.go`)
   - Dual protocol support with automatic version detection
   - Complete record parsing (NoSessionContext, SessionContext, WebSocketConnect)
   - Full message parsing for all USP operations
   - Comprehensive validation and error reporting
   - Structured logging with operation-specific details

6. **CWMP Message Processor** (`internal/cwmp/processor.go`)
   - SOAP envelope parsing and response generation
   - Complete CWMP message handling (Inform, GetRPCMethods, GetParameterNames, etc.)
   - TR-181 parameter validation and mock value generation
   - Session state management and fault handling
   - XML marshaling/unmarshaling for CWMP protocol

7. **gRPC Infrastructure** (`pkg/proto/dataservice/`, `internal/grpc/`)
   - Complete Protocol Buffer definitions for data service
   - Generated gRPC service and message code
   - gRPC server implementation with all CRUD operations
   - gRPC client wrapper for API Gateway integration
   - Bidirectional model converters (database models ↔ protobuf messages)

8. **Database Layer** (`internal/database/`)
   - GORM models for devices, parameters, alerts, sessions
   - PostgreSQL connection and initialization
   - Repository pattern with CRUD operations
   - Model converters for gRPC integration
   - Database migration and schema management

9. **USP Core Service** (`cmd/usp-service/main.go`)
   - Dual USP version processing engine
   - TR-181 Device:2 data model integration
   - Device management and lifecycle support

10. **TR-181 Data Model Integration**
    - Complete Device:2 v2.19.1 schema from usp.technology (822 objects loaded)
    - Data model structures and validation
    - Parameter path management with CWMP compatibility
    - Parameter type detection and writability validation

11. **Working Protocol Examples** (`examples/`)
    - **TR-369 USP Agent** (`examples/tr369-agent/main.go`): Complete WebSocket-based USP agent
    - **TR-069 Agent** (`examples/tr069-agent/main.go`): Complete SOAP/XML CWMP agent
    - Both examples work with the OpenUSP platform and demonstrate real protocol communication
    - Examples include proper binary message handling, authentication, and error handling

12. **Development Infrastructure**
    - **Swagger UI Documentation** (`docs/docs.go`, `docs/SWAGGER_UI.md`)
    - **Development Scripts** (`scripts/start-dev-with-swagger.sh`, `scripts/swagger-demo.sh`)
    - **Comprehensive Makefile** with all build, run, and testing targets
    - **Alternative Port Configuration** for conflict-free development

6. **Unified Configuration System** (`pkg/config/deployment.go`)
   - Environment-driven configuration with comprehensive defaults
   - **Consul enabled by default** for development (CONSUL_ENABLED=true)
   - Unified service configuration across all microservices
   - Runtime --consul flag support for all services
   - Comprehensive environment variable management

7. **Service Management** (`pkg/service/manager.go`)
   - Unified service lifecycle management
   - Consul service registration and discovery
   - Health check integration
   - Graceful shutdown handling

8. **Working Protocol Agents** (`examples/`)
   - **TR-369 USP Agent** (`examples/tr369-agent/main.go`): Complete WebSocket-based USP agent with onboarding
   - **TR-069 Agent** (`examples/tr069-agent/main.go`): Complete SOAP/XML CWMP agent with onboarding
   - Both agents work with the OpenUSP platform and demonstrate real protocol communication
   - Examples include proper binary message handling, authentication, and error handling

9. **Comprehensive Documentation** (`docs/`)
   - **README.md**: Project overview and quick start
   - **QUICKSTART.md**: 5-minute quick start guide
   - **INSTALLATION.md**: Installation guide for multiple platforms
   - **USER_GUIDE.md**: Detailed user guide for device management
   - **API_REFERENCE.md**: Complete API documentation
   - **CONFIGURATION.md**: Configuration reference
   - **DEPLOYMENT.md**: Production deployment guide
   - **DEVELOPMENT.md**: Development environment setup
   - **TROUBLESHOOTING.md**: Troubleshooting guide
   - **CONTRIBUTING.md**: Contribution guidelines

10. **Modern Build System** (`Makefile`)
    - **Infrastructure Management**: infra-up, infra-down, infra-status, infra-clean for PostgreSQL, Consul, RabbitMQ, MQTT, Grafana, Prometheus
    - **Service Building**: build-api-gateway, build-data-service, build-usp-service, build-mtp-service, build-cwmp-service
    - **Agent Building**: build-tr369-agent, build-tr069-agent for protocol agents
    - **Service Management**: start-*, stop-*, clean-* targets for all services
    - **Comprehensive Targets**: build-all, start-all, stop-all, clean-all
    - **Version Support**: All binaries support --version flag with proper LDFLAGS injection

### 🔧 **Key Features Implemented**
- **USP Message Processing**: Complete parsing pipeline for both v1.3 and v1.4
- **Protocol Buffer Support**: Native protobuf marshaling/unmarshaling
- **Version Detection**: Automatic USP version identification
- **Operation Support**: All TR-369 operations (Get, Set, Add, Delete, Operate, Notify, GetSupportedDM, GetInstances)
- **Response Generation**: Automatic USP response creation
- **Multi-Transport**: MQTT, STOMP, WebSocket, Unix Domain Socket support
- **Validation**: Comprehensive record and message validation
- **Error Handling**: Detailed error collection and reporting
- **REST API Gateway**: Complete device management API with gRPC backend
- **gRPC Microservices**: Internal service communication via Protocol Buffers
- **Database Abstraction**: Repository pattern with GORM ORM and PostgreSQL
- **Health Monitoring**: Comprehensive health checks across all services

## Critical Configuration Details

### **Environment Configuration** (`configs/openusp.env`)
```bash
# Service Discovery (Enabled by Default for Development)
CONSUL_ENABLED=true
CONSUL_ADDR=localhost:8500
CONSUL_DATACENTER=dc1

# Database Configuration
OPENUSP_DB_HOST=localhost
OPENUSP_DB_PORT=5433
OPENUSP_DB_NAME=openusp_db
OPENUSP_DB_USER=openusp
OPENUSP_DB_PASSWORD=openusp123
OPENUSP_DB_SSLMODE=disable

# Service Ports
OPENUSP_API_GATEWAY_PORT=6500     # REST API + Swagger UI
OPENUSP_MTP_SERVICE_PORT=8081     # WebSocket + HTTP
OPENUSP_CWMP_SERVICE_PORT=7547    # TR-069 standard port
OPENUSP_DATA_SERVICE_ADDR=localhost:56400  # gRPC

# Protocol Configuration
OPENUSP_USP_WS_URL=ws://localhost:8081/ws
OPENUSP_CWMP_ACS_URL=http://localhost:7547
OPENUSP_CWMP_USERNAME=acs
OPENUSP_CWMP_PASSWORD=acs123
```

### **Service Architecture**
- **Single Binary Design**: All services are unified binaries with --consul flag
- **Runtime Configuration**: CONSUL_ENABLED environment variable controls service discovery
- **Default Behavior**: Consul enabled by default for seamless development experience
- **Production Flexibility**: Can disable Consul with CONSUL_ENABLED=false

### **Build and Deployment**
```bash
# Core Infrastructure
make infra-up                    # Start PostgreSQL, Consul, RabbitMQ, MQTT, etc.

# Build Services
make build-all                   # Build all OpenUSP services + agents

# Start Services (with Consul enabled by default)
make start-data-service          # gRPC data service
make start-api-gateway          # REST API Gateway with Swagger UI
make start-mtp-service          # Message transport service
make start-cwmp-service         # CWMP/TR-069 service
make start-usp-service          # USP core service

# Start Working Protocol Agents
make start-tr369-agent          # TR-369 USP agent with onboarding
make start-tr069-agent          # TR-069 agent with onboarding

# Check Status
make consul-status              # View registered services
make infra-status              # Check infrastructure health
```

## Use Cases
- ISP device fleet management with TR-369 compliance
- IoT device provisioning and monitoring via USP
- Network equipment configuration management through multiple MTPs
- Automated device troubleshooting and maintenance with comprehensive logging
- Real-time device status and performance monitoring with dual USP version support

## Integration Points
- **Message Brokers**: RabbitMQ (STOMP), Mosquitto (MQTT)
- **Database**: PostgreSQL with TR-181 schema
- **Monitoring**: Prometheus metrics collection
- **External Testing**: obuspa agent for compliance testing
- **Deployment**: Kubernetes/Docker container orchestration

## Service Endpoints

### Production Endpoints (Default Ports)
- **API Gateway**: `http://localhost:8080` (REST API Gateway)
  - Health Check: `http://localhost:8080/health`
  - Status: `http://localhost:8080/status`
  - **Swagger UI**: `http://localhost:8080/swagger/index.html`
  - Devices API: `http://localhost:8080/api/v1/devices`
  - Parameters API: `http://localhost:8080/api/v1/parameters`
  - Alerts API: `http://localhost:8080/api/v1/alerts`
  - Sessions API: `http://localhost:8080/api/v1/sessions`
- **Data Service**: `localhost:9092` (gRPC internal service)
  - Health Check: gRPC HealthCheck method
  - Status: gRPC GetStatus method
  - All database operations via gRPC
- **MTP Service**: `http://localhost:8081` 
  - Demo UI: `http://localhost:8081/usp`
  - Health Check: `http://localhost:8081/health`
  - WebSocket: `ws://localhost:8081/ws`
- **CWMP Service**: `http://localhost:7547` (Standard TR-069 port)
  - Health Check: `http://localhost:7547/health`
  - Status: `http://localhost:7547/status`
  - Authentication: Basic Auth (acs/acs123)
- **USP Core Service**: gRPC internal communication

### Development Endpoints (Alternative Ports)
- **API Gateway**: `http://localhost:8082` (with Swagger UI)
  - **Swagger UI**: `http://localhost:8082/swagger/index.html`
  - All REST endpoints available with `/api/v1` prefix
- **MTP Service**: `http://localhost:8083`
- **CWMP Service**: `http://localhost:7548`  
- **Data Service**: `localhost:9093` (gRPC)
- **PostgreSQL**: `localhost:5433` (for development)

## Testing and Validation
- **USP Parsing Test**: `go run test/usp_parsing_demo.go`
- **MTP Service Demo**: Interactive WebSocket testing via browser UI
- **Protocol Compliance**: Tested with TR-369 specification requirements
- **Version Support**: Validated with both USP 1.3 and 1.4 messages
- **Multi-Transport**: WebSocket, MQTT, STOMP, Unix Domain Socket ready
- **Working Examples**: 
  - TR-369 USP Client: `make run-tr369-example`
  - TR-069 CWMP Client: `make run-tr069-example`
- **Swagger UI Testing**: Interactive API testing at `/swagger/index.html`
- **Development Workflows**:
  - Full platform: `make run-dev-swagger`
  - Swagger demo only: `./scripts/swagger-demo.sh`

## Latest Improvements (September 2025)

### ✅ **Recent Completions**

1. **Swagger UI Integration**
   - Complete OpenAPI 2.0 specification with all REST endpoints
   - Interactive web interface at `/swagger/index.html`
   - Comprehensive data model definitions for Device, Parameter, Alert, Session
   - Request/response examples with validation schemas
   - Production-ready API documentation

2. **Working Protocol Examples**
   - TR-369 USP Client with WebSocket MTP transport and binary Protocol Buffers
   - TR-069 CWMP Client with flexible SOAP/XML parsing
   - Both examples demonstrate real protocol communication with OpenUSP platform
   - Complete authentication, message handling, and error recovery

3. **Enhanced Development Infrastructure**
   - Alternative port configurations for conflict-free development
   - Automated startup scripts with proper service dependencies
   - Comprehensive Makefile targets for all development workflows
   - Clean project structure with unwanted files removed

4. **Protocol Improvements**
   - Flexible SOAP parser supporting multiple envelope formats
   - Enhanced WebSocket binary message handling
   - Improved error handling and logging across all services
   - Complete TR-181 parameter validation with 822 objects loaded

### 🎯 **Current State**
- **All Core Services**: Fully operational with health checks and status endpoints
- **API Documentation**: Complete Swagger UI with interactive testing capability
- **Protocol Compliance**: TR-369 USP and TR-069 CWMP fully implemented
- **Development Ready**: Clean codebase with comprehensive build and test infrastructure
- **Production Capabilities**: Scalable microservice architecture with proper separation of concerns

### 🚀 **Quick Start Commands**
```bash
# Start infrastructure and all services
make infra-up
make build-all
make start-all

# Run protocol agents
make start-tr369-agent
make start-tr069-agent

# Check service discovery status
make consul-status

# Test USP parsing
go run test/usp_parsing_demo.go
```

## Latest State (September 2025)

### ✅ **Final Architecture**
- **Single Binary Services**: All 5 services unified with runtime --consul flag
- **Consul Enabled by Default**: Service discovery enabled for seamless development
- **Clean Examples**: Only onboarding-enabled agents (tr369-agent, tr069-agent) remain
- **Complete Documentation**: User-focused documentation suite created from scratch
- **Production Ready**: Scalable microservice architecture with proper deployment guides

### 🏗️ **Project Recreation Instructions**

To recreate this entire project from scratch using these instructions:

#### **Step 1: Core Architecture Setup**
1. Create Go module with microservice structure
2. Implement single binary architecture with --consul flag support
3. Set up unified configuration system (`pkg/config/deployment.go`)
4. Create service manager with Consul integration (`pkg/service/manager.go`)

#### **Step 2: Core Services Implementation**
1. **Data Service**: gRPC server with PostgreSQL backend, GORM models, health checks
2. **API Gateway**: Gin-based REST API with gRPC client integration, Swagger UI
3. **MTP Service**: Multi-transport USP message processing (WebSocket, MQTT, STOMP, UDS)
4. **USP Service**: TR-369 protocol engine with dual version support (1.3/1.4)
5. **CWMP Service**: TR-069 protocol support with SOAP/XML processing

#### **Step 3: Protocol Implementation**
1. Implement USP parser with automatic version detection (`internal/usp/parser.go`)
2. Add CWMP processor with flexible SOAP parsing (`internal/cwmp/processor.go`)
3. Integrate TR-181 data model with 822 objects loaded
4. Support all TR-369 operations (Get, Set, Add, Delete, Operate, Notify, etc.)

#### **Step 4: Configuration and Infrastructure**
1. Set up environment configuration with Consul enabled by default
2. Configure Docker Compose for infrastructure (PostgreSQL, Consul, RabbitMQ, MQTT, Grafana, Prometheus)
3. Create comprehensive Makefile with modern build system
4. Implement health checks and metrics collection

#### **Step 5: Examples and Documentation**
1. Create onboarding-enabled protocol agents (tr369-agent, tr069-agent)
2. Generate complete user-focused documentation suite
3. Set up Swagger UI integration with interactive API testing
4. Create development and deployment scripts

#### **Step 6: Testing and Validation**
1. Implement comprehensive test suite
2. Add working protocol demonstrations
3. Validate TR-369 and TR-069 compliance
4. Test service discovery and microservice communication

### 📋 **Key Implementation Requirements**
- **All services must support --consul flag and --version flag**
- **Consul must be enabled by default in development (CONSUL_ENABLED=true)**
- **Single binary architecture - no dual consul/non-consul builds**
- **Environment-driven configuration with comprehensive defaults**
- **Complete gRPC internal communication with REST external API**
- **Working onboarding functionality in protocol agents**
- **User-focused documentation, not internal development docs**
- **Modern Makefile with standard targets (build, start, stop, clean, etc.)**
- **Complete TR-181 data model integration**
- **Dual USP version support (1.3 and 1.4) with automatic detection**

### 🎯 **Validation Checklist**
- [ ] All 5 services build and run successfully
- [ ] Consul service discovery works out of the box
- [ ] Protocol agents demonstrate real onboarding functionality
- [ ] Swagger UI provides interactive API testing
- [ ] Database operations work through gRPC data service
- [ ] Multi-transport message processing operational
- [ ] Health checks and metrics collection functional
- [ ] Complete documentation suite available
- [ ] Modern build system with comprehensive targets
- [ ] Production deployment guides complete

This project represents a complete, production-ready TR-369 User Service Platform implementation with modern microservice architecture, comprehensive protocol support, and user-focused documentation.
