# Changelog

All notable changes to the OpenUSP project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
- **Quick Fix Guide**: Added comprehensive troubleshooting guide (`docs/quickFix.md`)
- **API Gateway Scripts**: Created dynamic port discovery script (`scripts/check-api-gateway.sh`)
- **Grafana Troubleshooting**: Detailed monitoring troubleshooting guide (`docs/GRAFANA_TROUBLESHOOTING.md`)

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