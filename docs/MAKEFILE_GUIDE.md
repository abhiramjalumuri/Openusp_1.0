# OpenUSP Makefile Guide

A comprehensive guide to using the OpenUSP Makefile for development, deployment, and maintenance of the TR-369 User Service Platform.

## Table of Contents

- [Quick Start](#quick-start)
- [Infrastructure Management](#infrastructure-management)
- [Service Management](#service-management)
- [Development Workflow](#development-workflow)
- [Troubleshooting & Maintenance](#troubleshooting--maintenance)
- [Monitoring & Status](#monitoring--status)
- [Advanced Operations](#advanced-operations)
- [Common Issues & Solutions](#common-issues--solutions)

---

## Quick Start

### Complete Environment Setup
```bash
# Start everything from scratch
make infra-up          # PostgreSQL, Consul, RabbitMQ, MQTT, Grafana, Prometheus
make build-all         # Build all services and agents
make start-all         # Start all OpenUSP services
make dev-status        # Check everything is running
```

### Daily Development Workflow
```bash
# Morning startup (recommended)
make dev-status              # Check current state
make consul-cleanup-force    # Clean stale registrations
make dev-restart            # Clean restart of services

# During development
make build-usp-service      # Rebuild specific service
make restart-usp-service    # Restart specific service
make logs-usp-service       # Check service logs
```

---

## Infrastructure Management

### Core Infrastructure
The OpenUSP platform depends on several infrastructure services managed via Docker Compose.

```bash
# Start infrastructure services
make infra-up
# Services started:
# - PostgreSQL (port 5433)
# - Consul (port 8500) 
# - RabbitMQ (port 5672, management 15672)
# - Mosquitto MQTT (port 1883)
# - Prometheus (port 9090)
# - Grafana (port 3000)

# Check infrastructure status
make infra-status
# Shows Docker container status and health

# Stop infrastructure (preserves data volumes)
make infra-down

# Complete cleanup (⚠️ DESTROYS ALL DATA)
make infra-clean
```

### Infrastructure Endpoints
- **PostgreSQL**: `localhost:5433` (user: `openusp`, db: `openusp_db`)
- **Consul UI**: http://localhost:8500/ui/
- **RabbitMQ Management**: http://localhost:15672 (guest/guest)
- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090

---

## Service Management

### Core Services

The OpenUSP platform consists of 6 core microservices:

| Service | Purpose | Default Port | Health Check |
|---------|---------|--------------|--------------|
| `connection-manager` | gRPC connection pooling & circuit breaker | Dynamic | `/health` |
| `data-service` | Database operations via gRPC | Dynamic | `/health` |
| `api-gateway` | REST API & Swagger UI | Dynamic | `/health` |
| `usp-service` | TR-369 USP protocol engine | Dynamic | `/health` |
| `mtp-service` | Message Transport (WebSocket/MQTT/STOMP) | 8081 | `/health` |
| `cwmp-service` | TR-069 CWMP protocol support | 7547 | `/health` |

### Service Operations

```bash
# Build services
make build-all                    # Build all services
make build-{service-name}         # Build specific service
# Example: make build-usp-service

# Start services
make start-all                    # Start all services
make start-{service-name}         # Start specific service
# Example: make start-connection-manager

# Stop services
make stop-all                     # Stop all services
make stop-{service-name}          # Stop specific service

# Restart services
make restart-all                  # Restart all services
make restart-{service-name}       # Restart specific service

# View logs
make logs-{service-name}          # Tail service logs
# Example: make logs-mtp-service
```

### Protocol Agents

```bash
# Build agents
make build-usp-agent             # TR-369 USP agent
make build-cwmp-agent            # TR-069 CWMP agent

# Run agents (for testing)
make start-usp-agent             # Start USP agent
make start-cwmp-agent            # Start CWMP agent

# Agent binaries support version flags
./build/usp-agent -version 1.3   # Run with USP 1.3
./build/usp-agent -version 1.4   # Run with USP 1.4
```

---

## Development Workflow

### Recommended Daily Workflow

**1. Morning Setup**
```bash
make dev-status                  # Check current environment state
make consul-cleanup-force        # Clean any stale service registrations
make dev-restart                 # Clean restart of all services
```

**2. During Development**
```bash
# After code changes
make build-usp-service           # Rebuild changed service
make restart-usp-service         # Restart with new code
make logs-usp-service           # Check logs for issues

# Test the change
make start-usp-agent            # Test with USP agent
curl -s http://localhost:65121/api/v1/devices | jq .  # Check via API
```

**3. End of Day Cleanup**
```bash
make consul-cleanup-force        # Clean registrations
make stop-all                   # Stop services (infrastructure keeps running)
```

### Service Discovery & Port Management

All services use **Consul for service discovery** with dynamic port allocation:

```bash
# Check service registry
make consul-status              # List registered services
make dev-status                 # Complete environment overview

# Clean stale registrations
make consul-cleanup             # Remove only unhealthy instances
make consul-cleanup-force       # Remove ALL instances (before restart)
```

**Port Configuration:**
- Services get dynamic ports from Consul (except where fixed ports needed)
- WebSocket MTP: Fixed port 8081 (for client connections)
- CWMP Service: Fixed port 7547 (TR-069 standard)
- All other services: Dynamic ports via Consul

---

## Troubleshooting & Maintenance

### Development Environment Reset

When things go wrong, use these recovery procedures:

**1. Soft Reset (Recommended)**
```bash
make dev-restart                 # Restart services only
```

**2. Hard Reset (For persistent issues)**
```bash
make dev-reset                   # Full environment reset
# This will:
# - Stop all services
# - Force clean Consul registrations
# - Rebuild all services
# - Start everything fresh
```

**3. Nuclear Option (Last resort)**
```bash
make stop-all
make infra-down
make infra-clean                 # ⚠️ DESTROYS ALL DATA
make infra-up
make build-all
make start-all
```

### Common Issue Resolution

**Circuit Breaker Errors**
```bash
# Symptoms: "circuit breaker open", connection failures
make consul-cleanup-force        # Clean stale registrations
make restart-connection-manager  # Restart connection manager
make restart-usp-service        # Restart dependent services
```

**Port Conflicts**
```bash
# Check what's using ports
make dev-status
netstat -an | grep LISTEN | grep -E "(8081|7547|5433|8500)"

# Clean restart to get fresh ports
make consul-cleanup-force
make dev-restart
```

**Service Won't Start**
```bash
# Check logs
make logs-{service-name}

# Check dependencies
make infra-status               # Ensure infrastructure is running
make consul-status              # Check service discovery

# Force restart sequence
make stop-{service-name}
make consul-cleanup
make start-{service-name}
```

### Consul Cleanup Script

Advanced cleanup using the dedicated script:

```bash
# Clean only unhealthy services
./scripts/cleanup-consul.sh

# Force clean ALL services (before restart)
./scripts/cleanup-consul.sh --force

# Get help
./scripts/cleanup-consul.sh --help
```

---

## Monitoring & Status

### Status Commands

```bash
# Complete environment overview
make dev-status
# Shows:
# - Infrastructure container status
# - Consul service registry
# - OpenUSP service health
# - System resource usage

# Service-specific status
make consul-status               # Service registry
make service-status             # Service health only
make infra-status               # Infrastructure only
```

### Log Monitoring

```bash
# Individual service logs
make logs-connection-manager
make logs-data-service
make logs-usp-service
make logs-api-gateway
make logs-mtp-service
make logs-cwmp-service

# Follow logs in real-time
tail -f logs/usp-service.log
tail -f logs/connection-manager.log

# Search logs for errors
grep -i error logs/*.log
grep -i "circuit breaker" logs/*.log
```

### Health Checks

All services provide health endpoints:

```bash
# Check service health directly
curl http://localhost:56400/health    # Connection manager
curl http://localhost:65121/health    # API gateway (port varies)
curl http://localhost:8081/health     # MTP service

# Use Consul health checks
curl -s http://localhost:8500/v1/health/service/openusp-usp-service | jq '.[].Checks[].Status'
```

---

## Advanced Operations

### Service Dependencies

Services must start in the correct order due to dependencies:

```bash
# Dependency chain:
# 1. Infrastructure (PostgreSQL, Consul)
# 2. Connection Manager (provides connection pooling)
# 3. Data Service (database operations)
# 4. Core Services (USP, API Gateway, MTP, CWMP)

# The Makefile handles this automatically with wait-for-* targets
make start-all                   # Handles dependencies automatically
```

### Custom Configuration

```bash
# Environment-specific configuration
export CONSUL_ENABLED=false     # Disable service discovery
export OPENUSP_DB_HOST=remote-db # Use remote database
make start-data-service

# Load custom environment
source configs/production.env
make build-all
```

### Development vs Production

**Development Mode (Default)**
- Consul enabled for service discovery
- Dynamic port allocation
- Local PostgreSQL
- Debug logging
- Hot reload friendly

**Production Mode**
```bash
export CONSUL_ENABLED=false
export OPENUSP_DB_HOST=prod-db.example.com
export LOG_LEVEL=info
make build-all
# Deploy using Kubernetes or Docker Compose
```

---

## Common Issues & Solutions

### Issue: "Failed to get data service client via connection manager"

**Cause**: Connection manager port mismatch or stale registrations

**Solution**:
```bash
make consul-cleanup-force
make restart-connection-manager
make restart-usp-service
```

### Issue: "Circuit breaker open for service"

**Cause**: Too many connection failures, circuit breaker protecting system

**Solution**:
```bash
# 1. Clean stale registrations
make consul-cleanup-force

# 2. Restart connection manager (resets circuit breakers)
make restart-connection-manager

# 3. Restart affected services
make restart-usp-service restart-mtp-service
```

### Issue: Services can't find each other

**Cause**: Consul service discovery issues

**Solution**:
```bash
# Check Consul status
curl http://localhost:8500/v1/status/leader

# Clean and restart service discovery
make consul-cleanup-force
make dev-restart

# Verify services are registered
make consul-status
```

### Issue: Port already in use

**Cause**: Previous service instances not properly cleaned up

**Solution**:
```bash
# Find process using port
lsof -i :8081                   # Check specific port
netstat -an | grep LISTEN       # Check all listening ports

# Clean restart
make stop-all
make consul-cleanup-force
make start-all
```

### Issue: Database connection failures

**Cause**: PostgreSQL not running or wrong configuration

**Solution**:
```bash
# Check PostgreSQL
make infra-status

# Restart infrastructure if needed
make infra-down
make infra-up

# Check database connectivity
docker exec -it openusp-postgres-1 psql -U openusp -d openusp_db -c "SELECT version();"
```

---

## Testing & Validation

### End-to-End Testing

```bash
# 1. Start complete environment
make infra-up build-all start-all

# 2. Verify all services healthy
make dev-status

# 3. Test TR-369 protocol
make start-usp-agent            # Test USP onboarding

# 4. Test REST API
curl http://localhost:65121/api/v1/devices | jq .

# 5. Test Swagger UI
open http://localhost:65121/swagger/index.html
```

### Performance Testing

```bash
# Check system resources
make dev-status                 # Shows memory usage summary

# Monitor in real-time
watch -n 2 'make service-status'

# Load testing (example)
ab -n 1000 -c 10 http://localhost:65121/api/v1/devices
```

---

## Version Information

```bash
# Show version information
make version
# Displays:
# - OpenUSP version
# - Git commit
# - Build time
# - Protocol support (USP 1.3/1.4, CWMP)

# Show service endpoints
make endpoints
# Lists all public endpoints and their purposes
```

---

## Getting Help

```bash
# Show all available targets
make help

# Show this guide
make docs                       # Opens documentation

# Check specific service documentation
cat cmd/{service-name}/README.md
```

## Quick Reference Card

| Task | Command | Description |
|------|---------|-------------|
| **Setup** | `make infra-up build-all start-all` | Complete environment setup |
| **Daily Start** | `make dev-restart` | Clean restart for development |
| **Check Status** | `make dev-status` | Complete environment overview |
| **Clean Issues** | `make consul-cleanup-force` | Fix service discovery issues |
| **Reset Environment** | `make dev-reset` | Nuclear option for persistent issues |
| **View Logs** | `make logs-{service}` | Check service logs |
| **Test Protocol** | `make start-usp-agent` | Test TR-369 onboarding |
| **Documentation** | `make docs` | Show documentation guide |
| **Service Info** | `make consul-status` | Check service registry |
| **Emergency Stop** | `make stop-all` | Stop all services |

### New Development Tools (Recently Added)

| Tool | Command | Purpose |
|------|---------|---------|
| **Consul Cleanup** | `./scripts/cleanup-consul.sh` | Remove stale service registrations |
| **Force Cleanup** | `./scripts/cleanup-consul.sh --force` | Remove ALL registrations (before restart) |
| **Dev Status** | `make dev-status` | Complete environment health check |
| **Dev Reset** | `make dev-reset` | Full environment reset |
| **Dev Restart** | `make dev-restart` | Clean service restart |
| **Consul Status** | `make consul-status` | Service registry overview |

---

**💡 Pro Tips:**
- Always run `make dev-status` first to understand current state
- Use `make consul-cleanup-force` before major restarts
- Keep infrastructure running between service restarts for faster development
- Use service-specific restart commands for iterative development
- Monitor logs during development: `tail -f logs/usp-service.log`

For more detailed documentation, see:
- [QUICKSTART.md](QUICKSTART.md) - 5-minute setup guide
- [DEVELOPMENT.md](DEVELOPMENT.md) - Development environment setup
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Detailed troubleshooting
- [API_REFERENCE.md](API_REFERENCE.md) - REST API documentation