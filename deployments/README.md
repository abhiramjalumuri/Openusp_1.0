# OpenUSP Deployment Guide

This directory contains comprehensive deployment configurations for the OpenUSP (TR-369 User Service Platform) across different environments and orchestration platforms.

## 📁 Directory Structure

```
deployments/
├── docker/                          # Docker configurations
│   ├── Dockerfile                   # Multi-stage Dockerfile for all services
│   ├── Dockerfile.api-gateway      # API Gateway specific Dockerfile  
│   ├── Dockerfile.data-service     # Data Service specific Dockerfile
│   ├── Dockerfile.mtp-service      # MTP Service specific Dockerfile
│   └── Dockerfile.cwmp-service     # CWMP Service specific Dockerfile
├── environments/                    # Environment-specific configurations
│   ├── docker-compose.test.yml     # Testing environment (different ports)
│   ├── docker-compose.prod.yml     # Production environment with monitoring
│   ├── .env.test                   # Testing environment variables
│   └── .env.prod.template          # Production environment template
├── kubernetes/                      # Kubernetes manifests
│   ├── 00-namespace.yaml           # Namespace, ConfigMap, and Secrets
│   ├── 01-postgres.yaml            # PostgreSQL StatefulSet
│   ├── 02-data-service.yaml        # Data Service deployment
│   ├── 03-api-gateway.yaml         # API Gateway with Ingress
│   ├── 04-mtp-service.yaml         # MTP Service with LoadBalancer
│   ├── 05-cwmp-service.yaml        # CWMP Service with external access
│   ├── 06-usp-service.yaml         # USP Core Service
│   ├── 07-rabbitmq.yaml            # RabbitMQ StatefulSet with management
│   ├── 08-mosquitto.yaml           # Mosquitto MQTT broker
│   ├── 09-prometheus.yaml          # Prometheus monitoring
│   └── 10-grafana.yaml             # Grafana dashboards
├── docker-compose.postgres.yml      # PostgreSQL and Adminer services
├── docker-compose.infra.yml        # RabbitMQ, Mosquitto, Prometheus, Grafana
├── deploy-compose.sh                # Docker Compose deployment script
├── deploy-k8s.sh                   # Kubernetes deployment script
└── README.md                       # This file
```

## 🚀 Quick Start

### Docker Compose (Recommended for Development)

1. **Build and start testing environment:**
   ```bash
   ./deployments/deploy-compose.sh build
   ./deployments/deploy-compose.sh start test
   ```

2. **Access services:**
   - API Gateway: http://localhost:18080
   - MTP Service: http://localhost:18081/usp
   - CWMP Service: http://localhost:17547
   - Grafana: http://localhost:13000 (admin/admin)

### Kubernetes (Recommended for Production)

1. **Deploy to Kubernetes:**
   ```bash
   ./deployments/deploy-k8s.sh deploy
   ```

2. **Access services via port-forward:**
   ```bash
   kubectl port-forward -n openusp svc/api-gateway 8080:8080
   kubectl port-forward -n openusp svc/grafana 3000:3000
   ```

## 🐳 Docker Compose Deployments

### Testing Environment
- **Purpose**: Development and testing with isolated ports
- **File**: `environments/docker-compose.test.yml`
- **Ports**: Different from production (e.g., 18080 for API Gateway)
- **Features**: All services, separate PostgreSQL, lightweight monitoring

### Production Environment  
- **Purpose**: Production-ready deployment with full monitoring
- **File**: `environments/docker-compose.prod.yml`
- **Ports**: Standard production ports
- **Features**: Resource limits, restart policies, comprehensive logging, Prometheus + Grafana

### Commands

```bash
# Build all images
./deploy-compose.sh build

# Start testing environment
./deploy-compose.sh start test

# Start production environment  
./deploy-compose.sh start prod

# Check status
./deploy-compose.sh status

# View logs
./deploy-compose.sh logs test api-gateway

# Stop environment
./deploy-compose.sh stop test

# Clean up (including volumes)
./deploy-compose.sh cleanup
```

## ☸️ Kubernetes Deployments

### Architecture
- **Namespace**: `openusp` with dedicated resources
- **Services**: All OpenUSP services with proper health checks
- **Storage**: PostgreSQL StatefulSet with persistent volumes
- **Networking**: Services, Ingress controllers, LoadBalancers
- **Monitoring**: Prometheus + Grafana stack
- **Security**: Secrets management, RBAC, network policies

### Components

1. **Core Services**:
   - API Gateway (3 replicas, Ingress)
   - Data Service (2 replicas, gRPC)
   - MTP Service (2 replicas, WebSocket + multi-protocol)
   - CWMP Service (2 replicas, external access)
   - USP Service (2 replicas, gRPC)

2. **Infrastructure**:
   - PostgreSQL StatefulSet with persistent storage
   - RabbitMQ StatefulSet with STOMP plugin
   - Mosquitto MQTT broker

3. **Monitoring**:
   - Prometheus with service discovery
   - Grafana with pre-configured dashboards
   - Health checks and metrics endpoints

### Commands

```bash
# Deploy everything
./deploy-k8s.sh deploy

# Check deployment status
./deploy-k8s.sh status

# Show service endpoints
./deploy-k8s.sh endpoints

# Clean up
./deploy-k8s.sh cleanup
```

## 🔧 Configuration

### Environment Variables

**Testing Environment** (`.env.test`):
```bash
# Database
POSTGRES_HOST=postgres-test
POSTGRES_PORT=5432
POSTGRES_DB=openusp_test
POSTGRES_USER=openusp
POSTGRES_PASSWORD=openusp123

# Services
API_GATEWAY_PORT=18080
DATA_SERVICE_PORT=19092
MTP_SERVICE_PORT=18081
```

**Production Environment** (`.env.prod`):
```bash
# Database  
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_DB=openusp_prod
POSTGRES_USER=openusp
POSTGRES_PASSWORD=STRONG_PASSWORD_HERE

# Services
API_GATEWAY_PORT=8080
DATA_SERVICE_PORT=9092
MTP_SERVICE_PORT=8081

# Monitoring
PROMETHEUS_RETENTION=30d
GRAFANA_ADMIN_PASSWORD=SECURE_PASSWORD_HERE
```

### Service Ports

| Service | Test Port | Prod Port | Protocol |
|---------|-----------|-----------|----------|
| API Gateway | 18080 | 8080 | HTTP/REST |
| Data Service | 19092 | 9092 | gRPC |
| MTP Service | 18081 | 8081 | HTTP/WebSocket |
| CWMP Service | 17547 | 7547 | HTTP/SOAP |
| USP Service | 19093 | 9093 | gRPC |
| PostgreSQL | 15432 | 5432 | SQL |
| RabbitMQ Management | 15672 | 15672 | HTTP |
| Mosquitto MQTT | 11883 | 1883 | MQTT |
| Prometheus | 19090 | 9090 | HTTP |
| Grafana | 13000 | 3000 | HTTP |

## 🔍 Health Checks

All services include comprehensive health checks:

### HTTP Services
```bash
# API Gateway
curl http://localhost:8080/health

# MTP Service  
curl http://localhost:8081/health

# CWMP Service
curl http://localhost:7547/health
```

### gRPC Services
```bash
# Data Service (requires grpc_health_probe)
grpc_health_probe -addr=localhost:9092

# USP Service
grpc_health_probe -addr=localhost:9093
```

## 📊 Monitoring

### Prometheus Metrics
- **URL**: http://localhost:9090
- **Targets**: All OpenUSP services + infrastructure
- **Retention**: 15d (test) / 30d (prod)
- **Service Discovery**: Kubernetes pods and services

### Grafana Dashboards
- **URL**: http://localhost:3000
- **Credentials**: admin/admin (test) or admin/secure_password (prod)
- **Data Source**: Prometheus (auto-configured)
- **Dashboards**: OpenUSP services, infrastructure, Kubernetes

## 🔒 Security

### Secrets Management
- Database passwords
- Service authentication
- Grafana admin credentials
- RabbitMQ credentials
- TLS certificates (Kubernetes)

### Network Security
- Service-to-service communication via internal networks
- External access only through defined LoadBalancers/Ingress
- Basic authentication for management interfaces
- TLS termination at Ingress controllers

## 🚀 Deployment Strategies

### Development
1. Use Docker Compose testing environment
2. Different ports to avoid conflicts
3. Lightweight monitoring
4. Quick iteration cycles

### Staging
1. Use Docker Compose production environment
2. Production-like configuration
3. Full monitoring stack
4. Performance testing

### Production
1. Use Kubernetes deployment
2. High availability (multiple replicas)
3. Persistent storage
4. Comprehensive monitoring
5. Ingress controllers with TLS
6. Resource limits and quotas

## 🔧 Troubleshooting

### Common Issues

1. **Port Conflicts**:
   ```bash
   # Check port usage
   lsof -i :8080
   
   # Use testing environment with different ports
   ./deploy-compose.sh start test
   ```

2. **Database Connection Issues**:
   ```bash
   # Check PostgreSQL logs
   ./deploy-compose.sh logs test postgres
   
   # Verify connection
   docker exec -it postgres-container psql -U openusp -d openusp_test
   ```

3. **Service Discovery Issues**:
   ```bash
   # Check service status
   ./deploy-compose.sh status
   
   # Restart specific service
   docker-compose restart api-gateway
   ```

### Log Analysis

```bash
# All services
./deploy-compose.sh logs test

# Specific service
./deploy-compose.sh logs test api-gateway

# Follow logs
./deploy-compose.sh logs test api-gateway | tail -f
```

## 📝 Development Workflow

1. **Code Changes**: Make changes to service code
2. **Build**: `./deploy-compose.sh build`
3. **Test**: `./deploy-compose.sh start test`
4. **Verify**: Check health endpoints and functionality
5. **Staging**: `./deploy-compose.sh start prod`
6. **Production**: `./deploy-k8s.sh deploy`

## 🏗️ Infrastructure Requirements

### Minimum Requirements
- **CPU**: 4 cores
- **Memory**: 8GB RAM
- **Storage**: 20GB available space
- **Network**: Internet access for image downloads

### Production Requirements
- **CPU**: 8+ cores
- **Memory**: 16GB+ RAM  
- **Storage**: 100GB+ SSD
- **Network**: Load balancer, ingress controller
- **Kubernetes**: v1.24+ cluster

## 📚 Additional Resources

- [OpenUSP Documentation](../docs/)
- [API Gateway README](../cmd/api-gateway/README.md)
- [Data Service Documentation](../docs/DATA_SERVICE.md)
- [Versioning Guide](../docs/VERSIONING.md)
- [TR-369 Specification](https://www.broadband-forum.org/tr-369)
- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)

## 🤝 Contributing

When adding new deployment configurations:
1. Follow the existing naming conventions
2. Include proper health checks
3. Add resource limits
4. Update this README
5. Test in both environments

## 📄 License

This deployment configuration is part of the OpenUSP project and follows the same license terms.