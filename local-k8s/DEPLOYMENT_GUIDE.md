# 🚀 OpenUSP Local Kubernetes Deployment - Complete Guide

Perfect! I've created a comprehensive local Kubernetes setup for OpenUSP with complete Helm chart deployment. Here's everything you need to get started.

## 📁 What's Been Created

### 🛠️ Setup Scripts
- **`setup-cluster.sh`** - Creates Kind cluster with all required components
- **`deploy-openusp.sh`** - Deploys all OpenUSP services via Helm
- **`access-services.sh`** - Provides easy access to services and monitoring
- **`cleanup.sh`** - Complete cleanup of the local environment

### 📊 Helm Charts
- **Complete OpenUSP Helm chart** in `helm/openusp/`
- **Production-ready templates** with proper resource management
- **Monitoring integration** with Prometheus and Grafana
- **Local development optimizations**

## 🎯 Quick Start (3 Commands)

```bash
cd local-k8s

# 1. Setup local Kubernetes cluster
./setup-cluster.sh

# 2. Deploy all OpenUSP services
./deploy-openusp.sh

# 3. Access services and view status
./access-services.sh
```

## 🏗️ What Gets Deployed

### 🔧 Core Services
- **API Gateway** - http://localhost:8080
- **Data Service** - http://localhost:8081  
- **USP Service** - http://localhost:8082
- **MTP Service** - http://localhost:8083
- **CWMP Service** - http://localhost:7547
- **Connection Manager** - http://localhost:8084

### 🤖 Agents
- **USP Agent** - TR-369 client agent
- **CWMP Agent** - TR-069 client agent

### 🏛️ Infrastructure
- **PostgreSQL** - localhost:5432 (openusp/openusp123)
- **RabbitMQ** - http://localhost:15672 (guest/guest)
- **Mosquitto MQTT** - localhost:1883
- **Consul** - http://localhost:8500

### 📊 Monitoring Stack
- **Grafana** - http://localhost:3000 (admin/admin)
- **Prometheus** - http://localhost:9090
- **Complete dashboards** for all services

## 🎛️ Cluster Features

### ⚡ High-Performance Setup
- **3-node Kind cluster** (1 control-plane, 2 workers)
- **Local Docker registry** for fast image builds
- **Optimized port mappings** for all services
- **Persistent storage** with local storage class

### 🔒 Production-Ready Security  
- **Network policies** for service isolation
- **Security contexts** with non-root users
- **Resource quotas** and limits
- **Secrets management** for sensitive data

### 📈 Monitoring & Observability
- **Prometheus metrics** for all services
- **Grafana dashboards** with service-specific views
- **Health checks** and readiness probes
- **Comprehensive logging** with structured logs

## 🛠️ Management Commands

### Service Management
```bash
# View all services status
./access-services.sh status

# Check service health
./access-services.sh health

# View service logs
./access-services.sh logs api-gateway

# Follow logs in real-time
./access-services.sh follow data-service

# Port forward a service
./access-services.sh forward usp-service 9082 8082

# Execute shell into service
./access-services.sh shell mtp-service
```

### Monitoring
```bash
# Open Grafana dashboards
./access-services.sh grafana

# View resource usage
./access-services.sh resources

# View service access info
./access-services.sh access
```

### Kubernetes Operations
```bash
# View all pods
kubectl get pods -n openusp

# View services
kubectl get svc -n openusp

# View Helm releases
helm list -n openusp

# Upgrade deployment
helm upgrade openusp ./helm/openusp -n openusp
```

## 🔧 Development Workflow

### Local Development
1. **Services run in cluster** with persistent storage
2. **Local registry** for fast image iteration
3. **Hot-reload capable** (for development images)
4. **Complete service mesh** with inter-service communication

### Testing & Debugging
```bash
# Test all service endpoints
./access-services.sh health

# View comprehensive logs
./access-services.sh logs <service-name>

# Port forward for local debugging
./access-services.sh forward <service> <local-port> <service-port>

# Execute into pods for debugging
./access-services.sh shell <service-name>
```

## 📊 Monitoring & Dashboards

### Grafana Dashboards
- **OpenUSP Overview** - System-wide metrics
- **Service Performance** - Individual service metrics  
- **Infrastructure Health** - Node and cluster health
- **Protocol Metrics** - TR-369/TR-069 specific metrics

### Prometheus Metrics
- Custom metrics for each OpenUSP service
- Infrastructure metrics (CPU, memory, network)
- Application-specific metrics (requests, errors, latency)
- Business metrics (device connections, protocol usage)

## 🏗️ Architecture Highlights

### Microservices Design
- **Independent deployments** for each service
- **Service discovery** via Kubernetes DNS
- **Load balancing** with Kubernetes services
- **Auto-scaling** capabilities (configurable)

### Cloud-Native Features
- **12-factor app compliance**
- **Stateless services** with external state storage
- **Configuration via environment variables**
- **Health checks** and graceful shutdown

### Production Patterns
- **Circuit breakers** and retry logic
- **Distributed tracing** support
- **Centralized logging** aggregation
- **Secrets management** with Kubernetes secrets

## 🚀 Deployment Scenarios

### Development
```bash
# Quick local setup
./setup-cluster.sh && ./deploy-openusp.sh
```

### Testing
```bash
# Deploy with test configurations
helm upgrade openusp ./helm/openusp -n openusp --set development.enabled=true
```

### Production-like
```bash
# Deploy with production settings
helm upgrade openusp ./helm/openusp -n openusp -f values-production.yaml
```

## 🧹 Cleanup

```bash
# Complete environment cleanup
./cleanup.sh
```

This removes:
- Kind cluster and all services
- Docker images and registry
- All persistent data
- kubectl contexts

## 📋 Prerequisites Auto-Installation

The setup script automatically installs:
- **Kind** (Kubernetes in Docker)
- **Helm** (Package manager)
- **Ingress Controller** (NGINX)
- **Storage Classes** (Local storage)

## 🎉 Ready to Deploy!

Your complete local Kubernetes environment for OpenUSP is ready! This provides:

✅ **Production-grade local development**  
✅ **Complete service mesh with monitoring**  
✅ **Easy debugging and testing capabilities**  
✅ **Scalable architecture patterns**  
✅ **Enterprise-ready observability stack**

**Start your OpenUSP Kubernetes cluster now:**

```bash
cd local-k8s
./setup-cluster.sh
```

🚀 **Happy deploying!**