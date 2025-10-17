# Changelog

All notable changes to the OpenUSP project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.3.0] - 2025-10-17

### 🚀 Major Features

#### Complete Bidirectional Kafka Communication
- **Protocol Services**: Implemented full bidirectional communication for USP and CWMP services
  - USP Service ↔ API Gateway (usp.api.request/response topics)
  - USP Service ↔ Data Service (usp.data.request/response topics)
  - CWMP Service ↔ API Gateway (cwmp.api.request/response topics)
  - CWMP Service ↔ Data Service (cwmp.data.request/response topics)
- **Kafka Topics**: Expanded from 20 to 26 topics for complete bidirectional flows
- **Message Envelope**: Standardized USPMessageEvent JSON envelope format
  - endpoint_id: Agent identifier (proto::usp.agent.XXX)
  - message_id: Unique message correlation ID
  - message_type: Request/Response/Notify classification
  - payload: Protobuf binary data
  - mtp_protocol: Transport protocol identification (websocket/stomp/mqtt/http)

#### MTP Protocol Endpoint Routing
- **WebSocket MTP**: Implemented endpoint-to-client mapping for response routing
  - Dynamic endpoint ID extraction from USP Record protobuf
  - Client connection tracking with automatic cleanup on disconnect
  - Verified bidirectional communication with multiple concurrent agents
- **STOMP MTP**: Enhanced endpoint routing with RabbitMQ integration
  - Queue-based routing: /queue/agent.request and /queue/agent.response
  - Endpoint pattern matching for agent identification
  - Complete bidirectional flow verification
- **MQTT & HTTP MTPs**: Applied critical consumer fixes for outbound message delivery

### 🐛 Critical Bug Fixes

#### USP Service Envelope Unwrapping (Commit: 694f386)
- **Issue**: USP service could not detect USP version from Boot! events
- **Root Cause**: Kafka messages wrapped in USPMessageEvent JSON envelope, but service treated them as raw protobuf
- **Solution**: Unmarshal USPMessageEvent envelope, extract payload, then process as protobuf Record
- **Impact**: Device onboarding via Boot! events now fully operational
- **Verification**: Agent registration and REST API device access working

#### Kafka Consumer Start() Bug - CRITICAL (Commit: 32d222f)
- **Issue**: All MTP services receiving Kafka messages but not processing them
- **Root Cause**: Services calling Subscribe() but not Start() on Kafka consumers
  - Subscribe() only registers message handler
  - Start() method required to begin consumeLoop() for actual message consumption
- **Solution**: Added kafkaConsumer.Start() after Subscribe() in all MTP services:
  - cmd/mtp-services/websocket/main.go
  - cmd/mtp-services/stomp/main.go
  - cmd/mtp-services/mqtt/main.go
  - cmd/mtp-services/http/main.go
- **Impact**: Outbound message delivery now working across ALL MTP protocols

#### Endpoint ID Extraction (WebSocket MTP)
- **Issue**: Endpoint IDs extracted with extra binary characters from protobuf
- **Root Cause**: String termination not properly handled in binary protobuf data
- **Solution**: Parse only valid alphanumeric characters, dots, dashes, and underscores
- **Impact**: Clean endpoint mapping and correct message routing

### ✅ Verification & Testing

#### End-to-End Agent Onboarding
- **WebSocket Agents** (proto::usp.agent.001, .003):
  - Boot! NOTIFY_RESP received: 122 bytes ✅
  - GET_RESP received: 242 bytes ✅
  - "TR-369 USP Client demonstration completed!" ✅
- **STOMP Agent** (proto::usp.agent.002):
  - GET_RESP via /queue/agent.response: 242 bytes ✅
  - "TR-369 USP Client demonstration completed!" ✅

#### REST API Device Registration
- **Endpoint**: http://localhost:6500/api/v1/devices
- **Result**: 3 devices successfully registered and accessible
  - proto::usp.agent.001 (websocket, Plume-USP-SER001) - online ✅
  - proto::usp.agent.002 (stomp, Plume-USP-SER002) - online ✅
  - proto::usp.agent.003 (websocket, Plume-USP-SER003) - online ✅

#### Complete Data Flow Validation
- **Inbound**: Agent → MTP → Kafka (inbound) → USP Service → Data Service → PostgreSQL ✅
- **Outbound**: USP Service → Kafka (outbound) → MTP → Agent ✅
- **REST API**: Client → API Gateway → Kafka → Data Service → PostgreSQL → Response ✅

### 📝 Documentation
- Added comprehensive verification summary documenting:
  - System architecture with bidirectional flows
  - Critical bug fixes and their root causes
  - MTP protocol verification results
  - Complete end-to-end data flow diagrams

### 🔧 Technical Improvements
- **Message Routing**: Implemented endpoint-based routing in all MTP services
- **Connection Management**: Added proper cleanup of endpoint mappings on disconnect
- **Envelope Processing**: Standardized USPMessageEvent unwrapping across services
- **Consumer Lifecycle**: Fixed Kafka consumer initialization pattern across all MTP services

### ⚠️ Migration Notes
- All MTP services now require both Subscribe() and Start() for Kafka consumers
- USPMessageEvent envelope format is now mandatory for all Kafka message exchanges
- Endpoint ID extraction patterns standardized across WebSocket and STOMP MTPs

### 🎯 Production Readiness
- ✅ Complete bidirectional communication verified
- ✅ Multiple MTP protocols operational simultaneously
- ✅ Device onboarding working via TR-369 Boot! events
- ✅ REST API device management functional
- ✅ All critical bugs resolved and tested

## [1.2.0] - 2025-10-11

### 🔄 Refactors & Cleanup
- Unified agent configuration: merged `unified_agents.go` into `agent.go` introducing `LoadAgents` for single-pass load of USP (TR-369) & CWMP (TR-069) configs.
- Removed legacy env-based agent loader (`agent_env.go`) – YAML is now the sole source of agent config.
- Eliminated "static" terminology from service config types, functions, logs, and comments; neutral wording uses "fixed ports".
- Deleted obsolete `docker-compose.infra.static.yml`; consolidated infrastructure on `docker-compose.infra.yml`.
- Removed residual references to "static port configuration" in all service mains and error messages.

### 🧹 Deprecated / Removed
- Deleted: `pkg/config/agent_env.go`, `pkg/config/unified_agents.go` (logic merged).
- Removed specialized static infra compose file.

### 📝 Documentation
- Pending doc example updates to demonstrate `LoadAgents` instead of separate per-agent loaders.

### ⚠️ Migration Notes
- External code using `LoadUSPAgentUnified` / `LoadCWMPAgentUnified` remains supported (they now delegate to `LoadAgents`). Prefer adopting `LoadAgents` directly.
- Any former reliance on environment variable fallbacks for agent identity should be moved into `openusp.yml`.

### ✅ Integrity
- Build verified with `go build ./...` after refactor; behavior unchanged aside from naming & loader consolidation.

## [1.2.0] - 2025-10-11

### 🚀 Major Protocol Upgrades

#### TR-069 CWMP Service - v1.2 Protocol Upgrade
- **TR-069 v1.2 Support**: Complete upgrade from v1.0 to v1.2 with enhanced capabilities
- **Enhanced RPC Methods**: Added Download, Upload, GetAllQueuedTransfers, ScheduleInform, SetVouchers, GetOptions
- **Parameter Attributes**: Implemented TR-069 v1.2 enhanced parameter attribute structures with access control
- **Namespace Update**: Migrated from `urn:dslforum-org:cwmp-1-0` to `urn:dslforum-org:cwmp-1-2`
- **Backward Compatibility**: Maintained support for TR-069 v1.0, v1.1, and v1.2

#### API Gateway - TR-181 Compliance & Cleanup
- **TR-181 REST Endpoints**: Implemented comprehensive TR-181 compliant REST API structure
- **Device Management**: Updated endpoints from `/devices/{id}` to `/devices/{device_id}` with proper terminology
- **Parameter Operations**: Added bulk get/set, search, and filtering capabilities
- **Object Management**: Implemented object instance creation, update, and deletion
- **Command Execution**: Added TR-181 command execution with async support
- **Subscription Management**: Implemented parameter change notification subscriptions
- **Data Model Operations**: Added schema, data model, and object tree endpoints
- **API Cleanup**: Removed deprecated `/parameters/endpoint/{endpoint_id}` API
- **Swagger Documentation**: Comprehensive update with TR-181 compliant API documentation

### 🔧 Configuration Updates
- **CWMP Agent**: Updated default TR-069 version to 1.2 in `configs/cwmp-agent.yaml`
- **Service Health**: Enhanced health endpoints with protocol version information

### 📝 Documentation Improvements
- **API Documentation**: Complete Swagger specification update with TR-181 endpoints
- **TR-181 Test Suite**: Added comprehensive test script for all TR-181 operations
- **Protocol Documentation**: Updated README with TR-069 v1.2 feature descriptions

### 🧪 Testing & Validation
- **Integration Testing**: Verified TR-069 v1.2 protocol compatibility
- **API Testing**: Comprehensive TR-181 endpoint validation
- **Backward Compatibility**: Confirmed support for multiple TR-069 versions

## [1.1.0] - 2025-10-10

### 🎯 Major Architecture Changes

#### Complete Consul Removal & Static Port Configuration
- **Service Discovery Removal**: Eliminated Consul service discovery dependency entirely
- **Static Port Configuration**: Implemented predictable static port assignments for all services:
  - API Gateway: 6500 (Health: 6501, gRPC: 6502)
  - Data Service: 6100 (Health: 6101, gRPC: 6102)
  - Connection Manager: 6200 (Health: 6201, gRPC: 6202)
  - USP Service: 6400 (Health: 6401, gRPC: 6402)
  - CWMP Service: 7547 (Health: 7548, gRPC: 7549)
  - MTP Service: 8081 WebSocket (Health: 8082, gRPC: 8083)
- **Configuration Management**: Created `configs/services.yaml` for centralized static service configuration
- **Cross-Platform Compatibility**: Implemented host network mode for monitoring infrastructure

#### Infrastructure Simplification
- **Docker Compose Restructure**: Updated to `docker-compose.infra.yml` without Consul dependency
- **Prometheus Configuration**: Migrated to static target configuration (`prometheus.yml`)
- **Grafana Integration**: Enhanced dashboard provisioning with 4 comprehensive dashboards
- **Host Network Mode**: Implemented for cross-platform compatibility (macOS/Linux/Windows)

#### Service Architecture Improvements
- **gRPC Communication**: Maintained inter-service gRPC communication with static endpoints
- **Health Monitoring**: Separate health endpoints for all services
- **Metrics Collection**: Prometheus scraping all services via static configuration
- **Error Resolution**: Fixed CWMP service HTTP 405 errors by correcting metrics port configuration

### 🔧 Technical Improvements

#### Code Quality & Cleanup
- **Package Removal**: Completely removed `pkg/consul/` package and all dependencies
- **Service Refactoring**: Updated all service main.go files to use static configuration
- **Import Cleanup**: Removed all Consul-related imports across the codebase
- **Configuration Updates**: Updated `pkg/config/` packages to support static configuration

#### Build & Deployment
- **Makefile Enhancement**: Updated all build and run targets for static port configuration
- **Docker Configuration**: Simplified infrastructure setup without service discovery
- **Cross-Platform Scripts**: Added platform-specific deployment scripts

#### Documentation & Guides
- **Migration Guide**: Created comprehensive static port migration documentation
- **Cross-Platform Setup**: Detailed host network mode configuration guide
- **gRPC Services Guide**: Documentation for static inter-service communication
- **Configuration Reference**: Complete service port reference documentation

### 🌐 Cross-Platform Support

#### Host Network Implementation
- **Universal Compatibility**: Works seamlessly on macOS, Linux, and Windows
- **Direct Access**: Eliminated host.docker.internal issues with localhost access
- **Performance Improvement**: Reduced network overhead with host networking
- **Simplified Configuration**: No complex bridge network setup required

#### Monitoring Stack Enhancement
- **Grafana Dashboards**: 4 comprehensive dashboards for platform monitoring
- **Prometheus Metrics**: All 6 services exposing metrics on dedicated ports
- **Real-time Monitoring**: Live service health and performance monitoring
- **Alert Integration**: Ready for alerting and notification setup

### 🐛 Bug Fixes & Stability

#### Service Communication
- **Port Conflicts**: Resolved all service port conflicts with static assignments
- **HTTP 405 Errors**: Fixed CWMP service metrics endpoint configuration
- **Service Discovery**: Eliminated service discovery failures and timeouts
- **Build Issues**: Resolved all compilation errors from Consul removal

#### Infrastructure
- **Dashboard Loading**: Fixed Grafana dashboard provisioning issues
- **Metrics Scraping**: Corrected Prometheus target configuration for all services
- **Database Connectivity**: Maintained stable PostgreSQL connections
- **Message Queuing**: Preserved RabbitMQ and Mosquitto functionality

### 📋 Development Experience

#### Simplified Workflow
- **Predictable Ports**: No more dynamic port allocation complexity
- **Quick Testing**: Easy to test individual services on known ports
- **Debug-Friendly**: Simple localhost connections for debugging
- **Fast Startup**: No service discovery registration delays

#### Enhanced Tooling
- **Status Commands**: Improved service status checking with static targets
- **Health Checks**: Direct HTTP health endpoint access
- **Log Analysis**: Streamlined logging without service discovery noise
- **Build Process**: Faster builds without Consul dependency compilation

## [1.0.2] - 2025-10-03

### Fixed

#### TR-369 Compliance Fix
- **TR-369 Compliance**: Removed non-compliant direct API Gateway registration from USP agent
- **Device Registration**: Device registration now properly uses USP Notify OnBoard messages via MTP service
- **Architecture Verification**: Confirmed correct TR-369 flow: Agent → MTP → USP Service → Data Service

#### Agent Architecture Consolidation
- **Agent Restructuring**: Moved protocol agents to proper locations under `cmd/` directory:
  - `examples/tr369-agent` → `cmd/usp-agent` (TR-369 USP agent)
  - `examples/tr069-agent` → `cmd/cwmp-agent` (TR-069 CWMP agent)
- **YAML Configuration**: Agents now use YAML configuration files:
  - `configs/usp-agent.yaml` for TR-369 USP agent
  - `configs/cwmp-agent.yaml` for TR-069 CWMP agent
- **Makefile Updates**: Updated build targets to reflect new agent locations:
  - `make build-usp-agent`, `make start-usp-agent`
  - `make build-cwmp-agent`, `make start-cwmp-agent`

#### Service Configuration Improvements
- **Dual-Port Architecture**: Enhanced service configuration with separate ports for protocol and health endpoints
- **Service Discovery**: Improved Consul integration with proper service registration
- **Port Management**: Better port conflict resolution in development environments

#### Code Cleanup
- **Configuration Consolidation**: Removed redundant example configurations and consolidated to main configs
- **Deprecated Code Removal**: Removed unused test utilities and configuration files
- **Documentation Updates**: Updated all documentation to reflect new agent structure

### Enhanced

#### Documentation
- **README Updates**: Updated project structure and usage examples
- **Quick Start Guide**: Revised to use new agent commands and structure
- **Configuration Guide**: Added YAML configuration documentation for agents
- **Development Guide**: Updated build and development workflows

## [1.0.1] - 2025-10-01

### Fixed

#### API Gateway & Swagger UI Integration
- **Swagger UI 404 Fix**: Fixed health endpoint 404 errors in Swagger UI by adding `/api/v1/health` endpoint
- **Dynamic Port Configuration**: Swagger host now properly updates with dynamic port allocation
- **Backward Compatibility**: Maintained both `/health` and `/api/v1/health` endpoints
- **Service Discovery**: Enhanced Consul service registration cleanup and management

#### TR-069 Agent Dynamic Service Discovery
- **CWMP Service Discovery**: TR-069 agent now dynamically discovers CWMP service via Consul API
- **Onboarding Process**: Complete SOAP/XML TR-069 client with proper Inform message structure
- **Dynamic Endpoints**: Agent adapts to dynamic CWMP service ports automatically

#### Grafana Integration & Monitoring
- **Dashboard Data Sources**: Fixed Grafana dashboard data source UID mismatches
- **Login Issues**: Automated admin password reset functionality
- **Data Visualization**: All OpenUSP platform dashboards now display live metrics correctly

#### Documentation & Tooling
- **Troubleshooting Guide**: Added comprehensive troubleshooting guide (`docs/TROUBLESHOOTING.md`)
- **API Gateway Scripts**: Created dynamic port discovery script (`scripts/check-api-gateway.sh`)
- **Configuration Guide**: Detailed configuration and monitoring guide (`docs/CONFIGURATION.md`)

### Enhanced

#### Agent Integration
- **Dynamic Integration**: Complete documentation for agent dynamic service discovery (`AGENT_DYNAMIC_INTEGRATION.md`)
- **Protocol Compliance**: Enhanced TR-369 and TR-069 agent implementations with proper onboarding

## [1.0.0] - 2025-09-27

### Added

#### TR-369 USP Protocol Support
- **USP 1.3 and 1.4 Support**: Complete implementation with automatic version detection
- **Advanced USP Parser**: Comprehensive protocol buffer parsing and validation (`internal/usp/parser.go`)
- **Multi-Transport Protocols**: MQTT, STOMP, WebSocket, Unix Domain Socket MTPs
- **Complete USP Operations**: Get, Set, Add, Delete, Operate, Notify, GetSupportedDM, GetInstances
- **Session Context Support**: NoSessionContext, SessionContext, WebSocketConnect

#### MTP Service (`cmd/mtp-service/main.go`)
- **Multi-Protocol Transport**: Full support for MQTT, STOMP, WebSocket, Unix Domain Socket
- **WebSocket Demo UI**: Interactive testing interface at `http://localhost:8081/usp`
- **Health Monitoring**: Comprehensive health checks and status endpoints
- **Message Validation**: Complete USP record and message validation
- **Response Generation**: Automatic USP response creation for both versions

#### TR-069 CWMP Backward Compatibility
- **CWMP Service**: Complete TR-069 protocol implementation (`cmd/cwmp-service/main.go`)
- **SOAP/XML Processing**: Full CWMP message handling with SOAP envelope support
- **Standard RPC Methods**: GetRPCMethods, GetParameterNames, GetParameterValues, SetParameterValues
- **CWMP Message Processor**: Comprehensive SOAP processing (`internal/cwmp/processor.go`)
- **Session Management**: Multi-session support with configurable timeouts
- **Basic Authentication**: HTTP Basic Auth with configurable credentials

#### TR-181 Data Model Integration
- **Device:2 v2.19.1**: Complete Broadband Forum data model (822 objects loaded)
- **TR-181 Manager**: Parameter management and validation (`internal/tr181/manager.go`)
- **Data Model Structures**: Comprehensive XML schema parsing (`internal/tr181/datamodel.go`)
- **Parameter Operations**: Type checking, writability validation, mock value generation
- **CWMP Compatibility**: Seamless parameter management across USP and CWMP protocols

#### API Gateway Service
- **Gin Framework**: REST API gateway (`cmd/api-gateway/main.go`)
- **TR-181 Integration**: Data model namespace support
- **Health Endpoints**: Service status and health monitoring
- **External Interface**: Ready for client integrations

#### USP Core Service
- **Dual Protocol Engine**: USP 1.3/1.4 processing (`cmd/usp-service/main.go`)
- **Device Lifecycle**: Agent/device onboarding and management
- **TR-181 Integration**: Full data model support

#### Testing and Validation
- **USP Parsing Demo**: Complete parsing test (`test/usp_parsing_demo.go`)
- **CWMP Test Client**: Protocol testing client (`cmd/cwmp-test-client/main.go`)
- **Message Validation**: Both USP versions with comprehensive error reporting
- **Interactive Testing**: WebSocket UI for real-time USP message testing

#### Infrastructure and Architecture
- **Microservices Design**: Event-driven, stateless services with clear separation
- **Protocol Buffers**: USP v1.3 and v1.4 definitions (`pkg/proto/`)
- **Health Monitoring**: Comprehensive health checks across all services
- **Structured Logging**: Operation-specific logging with detailed context
- **Error Handling**: Comprehensive error collection and reporting

### Technical Details

#### Dependencies
- **Go 1.24+**: Latest Go version with enhanced performance
- **Protocol Buffers**: USP message encoding/decoding
- **Gin Framework**: HTTP REST API framework
- **Gorilla WebSocket**: WebSocket transport support
- **MQTT Client**: Eclipse Paho MQTT library
- **PostgreSQL**: Database support for TR-181 data persistence

#### Service Endpoints
- **MTP Service**: Port 8081 (USP protocol, multi-transport)
- **CWMP Service**: Port 7547 (TR-069 protocol, SOAP/XML)
- **API Gateway**: Port 8080 (REST API, external interface)
- **Health Monitoring**: All services include `/health` endpoints

#### Standards Compliance
- **TR-369**: User Services Platform specification compliance
- **TR-181**: Device:2 Data Model v2.19.1 support
- **TR-069**: CWMP protocol backward compatibility
- **Protocol Buffers**: USP 1.3 and 1.4 message encoding
- **SOAP/XML**: CWMP message encoding standards

### Configuration
- **Environment Variables**: Configurable ports, authentication, session limits
- **Authentication**: Basic HTTP auth for CWMP service
- **Session Management**: Configurable timeouts and concurrent session limits
- **Health Monitoring**: Comprehensive service status reporting

### Documentation
- **README.md**: Comprehensive project documentation
- **Copilot Instructions**: Detailed development guidelines
- **Architecture Overview**: Service interaction diagrams
- **API Documentation**: Endpoint specifications and examples

### Code Quality & Cleanup
- **Deprecated API Fixes**: Replaced `io/ioutil` with `io` and `os` packages (Go 1.19+ compatibility)
- **Static Analysis**: Fixed all `staticcheck` warnings and unused variables
- **Code Formatting**: Applied `go fmt` across entire codebase
- **Dependency Management**: Cleaned up with `go mod tidy`
- **Build Optimization**: Fixed MQTT client API compatibility issues
- **File Organization**: Moved demo files to `examples/` directory
- **Import Cleanup**: Removed unused imports and dependencies

### Infrastructure
- **Build System**: Complete Makefile with all development targets
- **Git Configuration**: Comprehensive `.gitignore` for Go projects
- **Docker Support**: Multi-stage Dockerfile and docker-compose setup
- **CI/CD Pipeline**: GitHub Actions workflow for automated testing

---

**Initial Release**: Complete TR-369 USP and TR-069 CWMP implementation with enterprise-grade architecture.