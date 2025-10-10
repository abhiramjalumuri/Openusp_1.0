# OpenUSP Documentation

Welcome to the OpenUSP (Open User Service Platform) documentation. This project implements the Broadband Forum's TR-369 standard for device management and communication.

## 📚 Documentation Structure

### For All Users
- [**Quick Start Guide**](QUICKSTART.md) - Get up and running in 5 minutes
- [**Installation**](INSTALLATION.md) - Complete installation instructions
- [**User Guide**](USER_GUIDE.md) - How to use OpenUSP for device management
- [**API Reference**](API_REFERENCE.md) - REST API documentation
- [**Proactive Onboarding**](PROACTIVE_ONBOARDING.md) - Factory default device onboarding solution
- [**Examples**](../README.md#-testing-and-validation) - Working examples and tutorials

### Platform-Specific Guides
- [**Cross-Platform Development**](CROSS_PLATFORM.md) - Professional cross-platform system (Linux, macOS, Windows)
- [**Linux Deployment**](LINUX_DEPLOYMENT.md) - Linux-specific setup and troubleshooting

### For Developers
- [**Development Guide**](DEVELOPMENT.md) - Setting up development environment
- [**Makefile Guide**](MAKEFILE_GUIDE.md) - Complete Makefile reference and workflows
- [**Architecture**](../README.md#-architecture-overview) - System design and components
- [**Contributing**](CONTRIBUTING.md) - How to contribute to the project
- [**Testing**](../README.md#-testing-and-validation) - Running tests and validation

### For Advanced Users
- [**Configuration**](CONFIGURATION.md) - Advanced configuration options
- [**Deployment**](DEPLOYMENT.md) - Production deployment strategies
- [**Protocol Details**](../README.md#-key-features) - TR-369 USP and TR-069 CWMP details
- [**Troubleshooting**](TROUBLESHOOTING.md) - Common issues and solutions

## 🚀 What is OpenUSP?

OpenUSP is a cloud-native implementation of the TR-369 User Service Platform that provides:

- **Device Management**: Remote configuration and monitoring of TR-369 compliant devices
- **Protocol Support**: Full TR-369 USP and backward-compatible TR-069 CWMP support
- **Multi-Transport**: MQTT, STOMP, WebSocket, and Unix Domain Socket transports
- **Service Discovery**: Consul-based service mesh for scalable deployments
- **Modern Architecture**: Microservices-based design with REST APIs and gRPC communication

## 🎯 Key Features

- ✅ **TR-369 Compliant**: Full USP 1.3 and 1.4 protocol support
- ✅ **TR-069 Compatible**: Backward compatibility with existing CWMP systems
- ✅ **Multi-Transport**: Support for various message transport protocols
- ✅ **Cloud Native**: Docker-based deployment with service discovery
- ✅ **Developer Friendly**: Comprehensive APIs, examples, and documentation
- ✅ **Production Ready**: Monitoring, logging, and health checks included

## 🛠️ Quick Start

```bash
# Clone the repository
git clone https://github.com/your-org/openusp.git
cd openusp

# Start infrastructure services
make infra-up

# Build and start all services
make all && make start

# Access the Web UI
open http://localhost:6500/swagger/index.html
```

## 📖 Getting Help

- **GitHub Issues**: Report bugs or request features
- **Documentation**: Check the relevant guide in this docs folder
- **Examples**: Look at working protocol agents in `cmd/usp-agent/` and `cmd/cwmp-agent/`
- **API Reference**: Use the interactive Swagger UI at `/swagger/index.html`

## 🏗️ Project Status

OpenUSP is actively developed and maintained. Current version: **1.0.0 Genesis**

- 🟢 **Stable**: Core TR-369 USP functionality
- 🟢 **Stable**: TR-069 CWMP backward compatibility
- 🟢 **Stable**: Multi-transport message handling
- 🟢 **Beta**: Advanced deployment configurations
- 🟡 **Alpha**: High-availability clustering

## 📝 License

This project is licensed under the Apache License 2.0 - see the [LICENSE](../LICENSE) file for details.