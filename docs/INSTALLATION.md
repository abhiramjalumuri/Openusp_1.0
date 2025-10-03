# 📦 Installation Guide

Complete installation instructions for OpenUSP on different platforms.

## System Requirements

### Minimum Requirements
- **OS**: Linux, macOS, or Windows with WSL2
- **RAM**: 4GB available
- **Disk**: 2GB free space
- **CPU**: 2 cores recommended

### Software Dependencies
- **Docker**: 20.10+ with Docker Compose
- **Go**: 1.21 or later
- **Git**: Any recent version
- **Make**: Build automation (usually pre-installed)

## Platform-Specific Installation

### 🐧 Linux (Ubuntu/Debian)

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install dependencies
sudo apt install -y git make curl

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER
newgrp docker

# Install Go
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify installation
go version
docker --version
```

### 🍎 macOS

```bash
# Install Homebrew (if not installed)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install dependencies
brew install git make go docker

# Start Docker Desktop
open -a Docker

# Verify installation
go version
docker --version
```

### 🪟 Windows (WSL2)

```powershell
# In PowerShell as Administrator
# Install WSL2 with Ubuntu
wsl --install -d Ubuntu

# Restart and open Ubuntu terminal
```

Then follow the Linux instructions above in the Ubuntu terminal.

## OpenUSP Installation

### Method 1: Quick Install (Recommended)

```bash
# Clone repository
git clone https://github.com/your-org/openusp.git
cd openusp

# Run installation script
./scripts/install.sh
```

### Method 2: Manual Installation

```bash
# Clone repository
git clone https://github.com/your-org/openusp.git
cd openusp

# Install Go dependencies
go mod download

# Start infrastructure
make infra-up

# Build services
make build

# Start services
make start

# Verify installation
make status
```

## Configuration

### Environment Variables

Copy and customize the configuration:

```bash
# Copy example configuration
cp configs/openusp.env.example configs/openusp.env

# Edit configuration
nano configs/openusp.env
```

Key settings to review:
```bash
# Service ports
OPENUSP_API_GATEWAY_PORT=6500
OPENUSP_MTP_SERVICE_PORT=8081

# Database settings
OPENUSP_DB_HOST=localhost
OPENUSP_DB_PORT=5433
OPENUSP_DB_NAME=openusp_db
```

### SSL/TLS Setup (Production)

For production deployments, configure SSL certificates:

```bash
# Create certificates directory
mkdir -p configs/ssl

# Copy your certificates
cp your-cert.pem configs/ssl/
cp your-key.pem configs/ssl/

# Update configuration
export OPENUSP_TLS_ENABLED=true
export OPENUSP_TLS_CERT_FILE=configs/ssl/your-cert.pem
export OPENUSP_TLS_KEY_FILE=configs/ssl/your-key.pem
```

## Verification

### Health Checks

```bash
# Check all services
make status

# Test API Gateway
curl http://localhost:6500/health

# Check service endpoints
make show-endpoints
```

### Web Interfaces

Verify these interfaces are accessible:

| Service | URL | Credentials |
|---------|-----|-------------|
| Swagger UI | http://localhost:6500/swagger/index.html | - |
| MTP Demo | http://localhost:8081/usp | - |
| Grafana | http://localhost:3000 | admin/admin |
| RabbitMQ | http://localhost:15672 | admin/admin |
| Adminer | http://localhost:8080 | openusp/openusp123 |

### Run Tests

```bash
# Run basic tests
make test

# Run example clients
make run-examples

# Test with real devices (advanced)
go run examples/tr369-agent/main.go
```

## Troubleshooting Installation

### Common Issues

#### Docker Permission Denied
```bash
# Linux: Add user to docker group
sudo usermod -aG docker $USER
newgrp docker
```

#### Port Already in Use
```bash
# Find what's using the port
sudo lsof -i :6500

# Kill the process or change OpenUSP ports
export OPENUSP_API_GATEWAY_PORT=6501
```

#### Go Module Issues
```bash
# Clear module cache
go clean -modcache

# Re-download dependencies
go mod download
```

#### Database Connection Failed
```bash
# Check PostgreSQL is running
make infra-status

# Restart database
docker restart openusp-postgres

# Check credentials in configs/openusp.env
```

### Getting Help

1. **Check Logs**: `make logs` or `tail -f logs/*.log`
2. **Service Status**: `make status`
3. **Infrastructure**: `make infra-status`
4. **GitHub Issues**: Report installation problems
5. **Discord/Slack**: Community support (if available)

## Uninstallation

### Remove OpenUSP Only
```bash
# Stop services
make stop

# Remove binaries
make clean

# Remove logs
rm -rf logs/
```

### Complete Removal
```bash
# Stop and remove everything
make infra-clean

# Remove repository
cd .. && rm -rf openusp/
```

## Next Steps

After successful installation:

1. **Quick Start**: Follow the [Quick Start Guide](QUICKSTART.md)
2. **User Guide**: Read the [User Guide](USER_GUIDE.md)
3. **Examples**: Try the [Examples](../README.md#-testing-and-validation)
4. **Development**: Set up [Development Environment](DEVELOPMENT.md)

## Production Deployment

For production environments, see:
- [Deployment Guide](DEPLOYMENT.md)
- [Configuration Reference](CONFIGURATION.md)
- [Security Best Practices](TROUBLESHOOTING.md#security-considerations)

Installation complete! 🎉