# OpenUSP - TR-369 User Service Platform

[![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![TR-369](https://img.shields.io/badge/TR--369-Compliant-green)](https://www.broadband-forum.org/technical/download/TR-369.pdf)
[![TR-181](https://img.shields.io/badge/TR--181-Device%3A2%20v2.19.1-blue)](https://usp.technology/specification/index.htm#sec:data-model-definition)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

A cloud-native microservice implementation of the Broadband Forum's **TR-369 based User Service Platform (USP)** with comprehensive **TR-069 CWMP backward compatibility**. Built for modern device management at scale.

## 🚀 Key Features

### TR-369 USP Protocol Support
- **✅ Dual Version Support**: Native USP 1.3 and 1.4 with automatic version detection
- **✅ Complete USP Operations**: Get, Set, Add, Delete, Operate, Notify, GetSupportedDM, GetInstances
- **✅ Advanced Message Parsing**: Comprehensive protocol buffer parsing and validation
- **✅ Multi-Transport Protocols**: MQTT, STOMP, WebSocket, Unix Domain Socket MTPs
- **✅ Session Contexts**: NoSessionContext, SessionContext, WebSocketConnect support
- **🆕 Proactive Device Onboarding**: Factory default agent support without initial NOTIFY messages

### TR-069 CWMP Backward Compatibility
- **✅ Full CWMP Protocol**: Complete TR-069 implementation with SOAP/XML processing
- **✅ Standard RPC Methods**: GetRPCMethods, GetParameterNames, GetParameterValues, SetParameterValues
- **✅ Authentication**: Basic HTTP authentication with session management
- **✅ TR-181 Integration**: Seamless parameter management across both protocols

### TR-181 Data Model
- **✅ Device:2 v2.19.1**: Complete Broadband Forum data model (822 objects)
- **✅ Parameter Validation**: Type checking and writability enforcement
- **✅ Namespace Management**: Hierarchical parameter path resolution

### Enterprise Architecture
- **✅ Microservices Design**: Event-driven, stateless services with clear separation
- **✅ API Gateway**: Gin-based REST API for external integrations
- **✅ Health Monitoring**: Comprehensive health checks and status endpoints
- **✅ Observability**: Structured logging with operation-specific details

## 🏗️ Architecture Overview

### Static Port Configuration (v1.1.0+)
OpenUSP now uses **static port configuration** for predictable, reliable service deployment:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   API Gateway   │    │   MTP Service   │    │  CWMP Service   │
│   (Port 6500)   │    │   (Port 8081)   │    │   (Port 7547)   │
│                 │    │                 │    │                 │
│ • REST API      │    │ • USP 1.3/1.4   │    │ • TR-069        │
│ • External      │    │ • Multi-MTP     │    │ • SOAP/XML      │
│   Interface     │    │ • WebSocket UI  │    │ • Session Mgmt  │
│ • Health: 6501  │    │ • Health: 8082  │    │ • Health: 7548  │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Data Service   │    │ Connection Mgr  │    │  USP Service    │
│   (Port 6100)   │    │   (Port 6200)   │    │   (Port 6400)   │
│                 │    │                 │    │                 │
│ • Database      │    │ • Service       │    │ • USP Protocol  │
│ • TR-181 Data   │    │   Discovery     │    │ • Message       │
│ • gRPC: 6102    │    │ • Connection    │    │   Processing    │
│ • Health: 6101  │    │   Pooling       │    │ • Health: 6401  │
└─────────────────┘    │ • gRPC: 6202    │    └─────────────────┘
                       │ • Health: 6201  │
                       └─────────────────┘
                  ┌─────────────────────────────┐
                  │      USP Core Service       │
                  │                             │
                  │ • Dual Protocol Support     │
                  │ • Device Lifecycle Mgmt     │
                  │ • TR-181 Integration        │
                  └─────────────────────────────┘
                                 │
                  ┌─────────────────────────────┐
                  │    TR-181 Data Service      │
                  │                             │
                  │ • Device:2 Data Model       │
                  │ • Parameter Management      │
                  │ • PostgreSQL Storage        │
                  └─────────────────────────────┘
```

### 🎯 Static Port Benefits (v1.1.0+)

- **✅ Predictable**: Same ports every time, no dynamic allocation
- **✅ Simple**: No service discovery complexity or dependencies  
- **✅ Fast**: Instant service startup without registration delays
- **✅ Reliable**: No service discovery failures or timeouts
- **✅ Debug-Friendly**: Easy to test individual services on known ports
- **✅ Cross-Platform**: Works seamlessly on macOS, Linux, and Windows

## 🛠️ Technology Stack

- **Language**: Go 1.24+
- **Protocols**: USP 1.3/1.4 (Protocol Buffers), TR-069 CWMP (SOAP/XML)
- **Transport**: MQTT (Eclipse Paho), STOMP (RabbitMQ), WebSocket (Gorilla), Unix Domain Socket
- **Web Framework**: Gin (REST API), Gorilla WebSocket
- **Database**: PostgreSQL with TR-181 schema
- **Monitoring**: Prometheus metrics, structured logging
- **Testing**: Broadband Forum's obuspa agent

## 🚦 Quick Start

### Prerequisites

```bash
# Install Go 1.24+
# Install PostgreSQL (optional for data persistence)
# Install Docker (optional for containerized deployment)
```

### Clone and Build

```bash
git clone https://github.com/plume-design-inc/openusp.git
cd openusp
go mod tidy
```

### Build with Version Information

Production build with embedded version information:
```bash
make build
```

Development build (without version injection):
```bash
make build-dev
```

Check version information:
```bash
make version
./build/mtp-service --version
./build/cwmp-service --version
./build/api-gateway --version
./build/usp-service --version
```

### Start Infrastructure & Services

#### 1. Start Infrastructure (PostgreSQL, Prometheus, Grafana)
```bash
make infra-up
```

#### 2. Start All Services
```bash
make run-services
```

#### 3. Verify Service Status
```bash
make status           # Comprehensive status check
make service-status   # Quick accessibility check
```

#### 4. Access Service Endpoints

**OpenUSP Services:**
- **API Gateway**: http://localhost:6500 (Swagger: /swagger/index.html)
- **MTP Service**: http://localhost:8081 (Demo UI: /usp, WebSocket: /ws)
- **CWMP Service**: http://localhost:7547 (TR-069 CWMP endpoint)
- **Data Service**: http://localhost:6100 (Health: /health, Status: /status)
- **USP Service**: http://localhost:6400 (Health: /health)
- **Connection Manager**: http://localhost:6200 (Health: /health)

**Infrastructure:**
- **Grafana**: http://localhost:3000 (admin/openusp123)
- **Prometheus**: http://localhost:9090
- **Database**: localhost:5433 (openusp/openusp123)

#### 2. CWMP Service (TR-069 Protocol)
```bash
go run cmd/cwmp-service/main.go
```
- **Endpoint**: http://localhost:7547
- **Health Check**: http://localhost:7547/health
- **Authentication**: Basic Auth (acs/acs123)

#### 3. API Gateway
```bash
go run cmd/api-gateway/main.go
```
- **REST API**: http://localhost:8080
- **Health Check**: http://localhost:8080/health

#### 4. Data Service (Database Operations)
```bash
# Start PostgreSQL (first time)
docker compose -f deployments/docker-compose.postgres.yml up -d

# Start Data Service
DB_PORT=5433 go run cmd/data-service/main.go
```
- **REST API**: http://localhost:8082
- **Health Check**: http://localhost:8082/health
- **Status**: http://localhost:8082/status
- **Database GUI**: http://localhost:8080 (Adminer)
- **Database**: postgresql://openusp:openusp123@localhost:5433/openusp_db

## 🧪 Testing & Validation

### Protocol Agent Testing
```bash
# Test TR-369 USP agent
make start-usp-agent
# or manually:
go run cmd/usp-agent/main.go --config configs/usp-agent.yaml

# Test TR-069 CWMP agent  
make start-cwmp-agent  
# or manually:
go run cmd/cwmp-agent/main.go --config configs/cwmp-agent.yaml
```

### USP Protocol Testing
```bash
# Test USP message parsing (both v1.3 and v1.4)
go run test/usp_parsing_demo.go

# Interactive WebSocket testing
# Open browser: http://localhost:8081/usp
```

### CWMP Protocol Testing
```bash
# Test CWMP service health
curl -s http://localhost:7547/health

# Test GetRPCMethods
curl -X POST -H "Content-Type: text/xml" -u acs:acs123 \
  -d '<?xml version="1.0"?><soap:Envelope xmlns:soap="http://schemas.xmlsoap.org/soap/envelope/" xmlns:cwmp="urn:dslforum-org:cwmp-1-0"><soap:Body><cwmp:GetRPCMethods/></soap:Body></soap:Envelope>' \
  http://localhost:7547
```

### Database Testing
```bash
# Comprehensive database testing
./scripts/test-database.sh

# Manual API testing
curl http://localhost:8082/health
curl http://localhost:8082/status
curl http://localhost:8082/api/v1/devices

# Create a test device
curl -X POST http://localhost:8082/api/v1/devices \
  -H "Content-Type: application/json" \
  -d '{"endpoint_id":"test-001","manufacturer":"OpenUSP","model_name":"Test Device"}'
```

## 📁 Project Structure

```
openusp/
├── cmd/                       # Application entry points
│   ├── api-gateway/          # REST API Gateway service
│   ├── usp-service/          # USP core service
│   ├── mtp-service/          # Message Transfer Protocol service
│   ├── cwmp-service/         # CWMP/TR-069 service
│   ├── data-service/         # Database service
│   ├── usp-agent/            # TR-369 USP agent (TR-369 protocol client)
│   └── cwmp-agent/           # TR-069 CWMP agent (TR-069 protocol client)
├── internal/                 # Private application code
│   ├── usp/                  # USP protocol implementation
│   │   └── parser.go         # USP 1.3/1.4 parser
│   ├── cwmp/                 # CWMP protocol implementation
│   │   └── processor.go      # SOAP/XML message processor
│   ├── mtp/                  # MTP protocol implementations
│   ├── tr181/                # TR-181 data model
│   │   ├── datamodel.go      # Data model structures
│   │   └── manager.go        # Parameter management
│   └── database/             # Database layer
├── pkg/                      # Public library code
│   ├── proto/                # Protocol buffer definitions
│   │   ├── v1_3/            # USP 1.3 protocol buffers
│   │   └── v1_4/            # USP 1.4 protocol buffers
│   ├── datamodel/           # TR-181 XML schemas
│   │   ├── tr-181-2-19-1-usp-full.xml  # Device:2 schema
│   │   └── tr-106-types.xml             # Data types
│   └── common/              # Shared utilities
├── test/                    # Test files and demonstrations
├── deployments/             # Docker and K8s configs
├── scripts/                 # Build and deployment scripts
└── docs/                    # Documentation
```

## 🔧 Configuration

### Environment Variables

```bash
# MTP Service
export MTP_PORT=8081
export MTP_WEBSOCKET_PORT=8081

# CWMP Service  
export CWMP_PORT=7547
export CWMP_USERNAME=acs
export CWMP_PASSWORD=acs123
export CWMP_MAX_SESSIONS=100

# API Gateway
export API_GATEWAY_PORT=8080

# Database (optional)
export DB_HOST=localhost
export DB_PORT=5432
export DB_NAME=openusp
```

## 📊 Service Endpoints

| Service | Port | Health Check | Description |
|---------|------|--------------|-------------|
| **MTP Service** | 8081 | `/health` | USP Protocol, Multi-transport |
| **CWMP Service** | 7547 | `/health` | TR-069 Protocol, SOAP/XML |
| **API Gateway** | 8080 | `/health` | REST API, External interface |
| **USP Core** | gRPC | Internal | Device management core |
| **Data Service** | gRPC | Internal | TR-181 data operations |

## 🌐 Use Cases

- **ISP Device Fleet Management**: Manage thousands of CPE devices with TR-369 compliance
- **IoT Device Provisioning**: Automated device onboarding and configuration
- **Network Equipment Management**: Remote configuration and monitoring
- **Legacy System Integration**: TR-069 backward compatibility for existing deployments
- **Multi-Protocol Device Support**: Unified management across USP and CWMP protocols

## 🔗 Standards Compliance

- **[TR-369](https://www.broadband-forum.org/technical/download/TR-369.pdf)**: User Services Platform specification
- **[TR-181](https://www.broadband-forum.org/technical/download/TR-181_Issue-2.pdf)**: Device:2 Data Model for TR-069/TR-369
- **[TR-069](https://www.broadband-forum.org/technical/download/TR-069.pdf)**: CPE WAN Management Protocol
- **Protocol Buffers**: USP message encoding (versions 1.3 and 1.4)
- **SOAP/XML**: CWMP message encoding for TR-069 compatibility

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- **Broadband Forum** for TR-369, TR-181, and TR-069 specifications
- **usp.technology** for TR-181 Device:2 data model schemas
- **obuspa** project for USP protocol testing and validation
- **Go community** for excellent libraries and tools

---

**Built with ❤️ for modern device management**