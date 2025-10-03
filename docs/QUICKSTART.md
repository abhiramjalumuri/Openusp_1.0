# 🚀 Quick Start Guide

Get OpenUSP running in 5 minutes with this step-by-step guide.

## Prerequisites

- **Docker & Docker Compose**: For infrastructure services
- **Go 1.21+**: For building the services
- **Git**: For cloning the repository

## Step 1: Clone and Setup

```bash
# Clone the repository
git clone https://github.com/your-org/openusp.git
cd openusp

# Verify Go installation
go version
```

## Step 2: Start Infrastructure

```bash
# Start PostgreSQL, RabbitMQ, MQTT, and other services
make infra-up

# Wait for services to be ready (about 30 seconds)
make infra-status
```

You should see all services as "healthy".

## Step 3: Build and Start OpenUSP

```bash
# Build all OpenUSP services and agents
make build-all

# Start all services
make start-all
```

This will start:
- **API Gateway** (REST API + Swagger UI)
- **Data Service** (Database operations)  
- **MTP Service** (Message transport)
- **USP Service** (TR-369 protocol engine)
- **CWMP Service** (TR-069 compatibility)

## Step 4: Verify Installation

### Web Interfaces

Open these URLs in your browser:

- **📊 Swagger UI**: http://localhost:6500/swagger/index.html
- **🧪 MTP Demo**: http://localhost:8081/usp
- **📈 Grafana**: http://localhost:3000 (admin/admin)
- **🐰 RabbitMQ**: http://localhost:15672 (admin/admin)

### API Test

```bash
# Test the API Gateway
curl http://localhost:6500/health

# Should return: {"status":"healthy"}
```

### Service Status

```bash
# Check all services
make ou-status

# View service endpoints
make show-static-endpoints
```

## Step 5: Try Protocol Agents

```bash
# Run TR-369 USP agent (YAML-configured)
make start-usp-agent

# Run TR-069 CWMP agent (YAML-configured) 
make start-cwmp-agent

# Or run individually with custom configs
./build/usp-agent --config configs/usp-agent.yaml
./build/cwmp-agent --config configs/cwmp-agent.yaml
```

## 🎯 What's Next?

### For Normal Users
- Explore the [**Swagger UI**](http://localhost:6500/swagger/index.html) to test the APIs
- Read the [**User Guide**](USER_GUIDE.md) for device management
- Check out [**Examples**](../README.md#-testing-and-validation) for common use cases

### For Developers
- Set up the [**Development Environment**](DEVELOPMENT.md)
- Read the [**Architecture Guide**](DEPLOYMENT.md#architecture-patterns)
- Explore the [**API Reference**](API_REFERENCE.md)

## 🛑 Stopping Services

```bash
# Stop all services
make stop-all

# Stop only infrastructure
make infra-down

# Clean everything (removes containers and volumes)
make infra-clean
```

## ❓ Troubleshooting

### Services Won't Start
```bash
# Check infrastructure is running
make infra-status

# Check logs (example for api-gateway)
make logs-api-gateway

# Or manually check specific service logs
tail -f logs/api-gateway.log

# Restart everything
make stop-all && make start-all
```

### Port Conflicts
The default ports are:
- API Gateway: 6500
- MTP Service: 8081
- CWMP Service: 7547

If these conflict, see [Configuration](CONFIGURATION.md) for customization.

### Need Help?
- Check [Troubleshooting Guide](TROUBLESHOOTING.md)
- Look at service logs in the `logs/` directory
- Open an issue on GitHub

## 🎉 Success!

You now have OpenUSP running locally. The system provides:

- **REST API** for device management
- **WebSocket** for real-time communication
- **Monitoring** dashboards
- **Example clients** for testing

Ready to manage TR-369 devices! 🚀