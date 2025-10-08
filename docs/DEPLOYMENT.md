# 🚀 Deployment Guide

Production deployment options and best practices for OpenUSP.

## Table of Contents

- [Deployment Options](#deployment-options)
- [Docker Deployment](#docker-deployment)
- [Kubernetes Deployment](#kubernetes-deployment)
- [Bare Metal Deployment](#bare-metal-deployment)
- [Cloud Provider Deployment](#cloud-provider-deployment)
- [High Availability Setup](#high-availability-setup)
- [Monitoring and Observability](#monitoring-and-observability)
- [Security Considerations](#security-considerations)

## Deployment Options

### Overview

OpenUSP supports multiple deployment patterns:

1. **Single Node**: All services on one machine
2. **Multi-Node**: Services distributed across multiple machines
3. **Container-Based**: Docker or Kubernetes deployment
4. **Cloud-Native**: Managed services and auto-scaling
5. **Hybrid**: Mix of on-premises and cloud components

### Architecture Patterns

#### Single Binary Deployment
```bash
# All services in one binary with runtime configuration
./openusp-api-gateway --consul=false
./openusp-data-service --consul=false
./openusp-mtp-service --consul=false
./openusp-usp-service --consul=false
./openusp-cwmp-service --consul=false
```

#### Microservices with Service Discovery
```bash
# With Consul service discovery
export CONSUL_ENABLED=true
./openusp-api-gateway --consul=true
./openusp-data-service --consul=true
./openusp-mtp-service --consul=true
./openusp-usp-service --consul=true
./openusp-cwmp-service --consul=true
```

## Docker Deployment

### Quick Start with Docker Compose

```bash
# Start infrastructure services
make infra-up

# Build and start OpenUSP services
make build
make start

# Or combine both
make run-all
```

### Docker Compose Configuration

Main deployment file: `deployments/docker-compose.infra.yml`

```yaml
version: '3.8'

services:
  # Database
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: openusp
      POSTGRES_USER: openusp
      POSTGRES_PASSWORD: openusp123
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./configs/init-db.sql:/docker-entrypoint-initdb.d/init-db.sql
    ports:
      - "5432:5432"
    restart: unless-stopped

  # Service Discovery
  consul:
    image: consul:1.15
    command: agent -server -ui -node=server-1 -bootstrap-expect=1 -client=0.0.0.0
    ports:
      - "8500:8500"
      - "8600:8600/udp"
    restart: unless-stopped

  # Message Brokers
  rabbitmq:
    image: rabbitmq:3-management
    environment:
      RABBITMQ_DEFAULT_USER: guest
      RABBITMQ_DEFAULT_PASS: guest
    volumes:
      - ./configs/rabbitmq-plugins:/etc/rabbitmq/enabled_plugins
    ports:
      - "5672:5672"
      - "15672:15672"
      - "61613:61613"
    restart: unless-stopped

  mosquitto:
    image: eclipse-mosquitto:2
    volumes:
      - ./configs/mosquitto.conf:/mosquitto/config/mosquitto.conf
    ports:
      - "1883:1883"
      - "9001:9001"
    restart: unless-stopped

  # Monitoring
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./configs/prometheus-dev.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"
    restart: unless-stopped

  grafana:
    image: grafana/grafana:latest
    environment:
      GF_SECURITY_ADMIN_PASSWORD: admin
    volumes:
      - ./configs/grafana-datasources.yml:/etc/grafana/provisioning/datasources/datasources.yml
      - ./configs/grafana-dashboard-provider.yml:/etc/grafana/provisioning/dashboards/dashboard-provider.yml
      - ./configs/grafana-dashboards:/var/lib/grafana/dashboards
    ports:
      - "3000:3000"
    restart: unless-stopped

volumes:
  postgres_data:
```

### Custom Docker Images

Individual service Dockerfiles are available in `deployments/docker/`:

```dockerfile
# Example: Dockerfile.api-gateway
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o api-gateway cmd/api-gateway/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/api-gateway .
COPY configs/ configs/

EXPOSE 8080
CMD ["./api-gateway"]
```

### Environment-Specific Deployments

#### Production
```bash
# Use production configuration
docker compose -f deployments/environments/docker-compose.prod.yml up -d
```

#### Testing
```bash
# Use test configuration
docker compose -f deployments/environments/docker-compose.test.yml up -d
```

### Docker Deployment Commands

```bash
# Build all images
make docker-build

# Start infrastructure only
make infra-up

# Start OpenUSP services
make docker-up

# View logs
make docker-logs

# Scale services
docker compose up -d --scale api-gateway=3

# Stop and cleanup
make docker-down
make docker-clean
```

## Kubernetes Deployment

### Prerequisites

- Kubernetes cluster (1.20+)
- kubectl configured
- Helm 3.x (optional)

### Deployment Files

Kubernetes manifests are in `deployments/kubernetes/`:

```
kubernetes/
├── 00-namespace.yaml        # Namespace creation
├── 01-postgres.yaml         # PostgreSQL database
├── 02-data-service.yaml     # Data service
├── 03-api-gateway.yaml      # API Gateway
├── 04-mtp-service.yaml      # MTP service
├── 05-usp-service.yaml      # USP service
├── 06-cwmp-service.yaml     # CWMP service
├── 07-consul.yaml           # Service discovery
├── 08-rabbitmq.yaml         # RabbitMQ broker
├── 09-mosquitto.yaml        # MQTT broker
├── 10-prometheus.yaml       # Monitoring
├── 11-grafana.yaml          # Dashboards
└── ingress.yaml             # Load balancer
```

### Basic Kubernetes Deployment

```bash
# Create namespace
kubectl apply -f deployments/kubernetes/00-namespace.yaml

# Deploy infrastructure services
kubectl apply -f deployments/kubernetes/01-postgres.yaml
kubectl apply -f deployments/kubernetes/07-consul.yaml
kubectl apply -f deployments/kubernetes/08-rabbitmq.yaml
kubectl apply -f deployments/kubernetes/09-mosquitto.yaml

# Deploy OpenUSP services
kubectl apply -f deployments/kubernetes/02-data-service.yaml
kubectl apply -f deployments/kubernetes/03-api-gateway.yaml
kubectl apply -f deployments/kubernetes/04-mtp-service.yaml
kubectl apply -f deployments/kubernetes/05-usp-service.yaml
kubectl apply -f deployments/kubernetes/06-cwmp-service.yaml

# Deploy monitoring
kubectl apply -f deployments/kubernetes/10-prometheus.yaml
kubectl apply -f deployments/kubernetes/11-grafana.yaml

# Configure ingress
kubectl apply -f deployments/kubernetes/ingress.yaml
```

### Example Service Definition

```yaml
# api-gateway service
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-gateway
  namespace: openusp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: api-gateway
  template:
    metadata:
      labels:
        app: api-gateway
    spec:
      containers:
      - name: api-gateway
        image: openusp/api-gateway:latest
        ports:
        - containerPort: 8080
        env:
        - name: CONSUL_ENABLED
          value: "true"
        - name: CONSUL_ADDRESS
          value: "consul:8500"
        - name: OPENUSP_DATABASE_HOST
          value: "postgres"
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
---
apiVersion: v1
kind: Service
metadata:
  name: api-gateway
  namespace: openusp
spec:
  selector:
    app: api-gateway
  ports:
  - port: 8080
    targetPort: 8080
  type: ClusterIP
```

### Helm Deployment

Create Helm chart:

```bash
helm create openusp
```

Values file (`deployments/helm/values.yaml`):

```yaml
# OpenUSP Helm Values
global:
  registry: "openusp"
  tag: "latest"
  consul:
    enabled: true
    address: "consul:8500"

apiGateway:
  replicaCount: 3
  image:
    repository: openusp/api-gateway
    tag: latest
  service:
    type: ClusterIP
    port: 8080
  ingress:
    enabled: true
    host: api.openusp.local

dataService:
  replicaCount: 2
  image:
    repository: openusp/data-service
    tag: latest
  service:
    type: ClusterIP
    port: 9092

database:
  host: postgres
  name: openusp
  username: openusp
  password: openusp123

monitoring:
  prometheus:
    enabled: true
  grafana:
    enabled: true
    adminPassword: admin
```

Deploy with Helm:

```bash
# Install chart
helm install openusp deployments/helm/openusp -f deployments/helm/values.yaml

# Upgrade
helm upgrade openusp deployments/helm/openusp -f deployments/helm/values.yaml

# Uninstall
helm uninstall openusp
```

### Kubernetes Commands

```bash
# Deploy all resources
make k8s-deploy

# Check deployment status
kubectl get pods -n openusp
kubectl get services -n openusp

# View logs
kubectl logs -f deployment/api-gateway -n openusp

# Scale services
kubectl scale deployment api-gateway --replicas=5 -n openusp

# Port forward for testing
kubectl port-forward service/api-gateway 8080:8080 -n openusp

# Clean up
make k8s-clean
```

## Bare Metal Deployment

### System Requirements

#### Minimum Requirements
- **CPU**: 4 cores
- **RAM**: 8 GB
- **Storage**: 50 GB SSD
- **Network**: 1 Gbps
- **OS**: Ubuntu 20.04+, CentOS 8+, or RHEL 8+

#### Recommended Requirements
- **CPU**: 8+ cores
- **RAM**: 16+ GB
- **Storage**: 100+ GB NVMe SSD
- **Network**: 10 Gbps
- **OS**: Ubuntu 22.04 LTS

### Installation Steps

#### 1. Prepare System

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install dependencies
sudo apt install -y curl wget git build-essential

# Install Go 1.21+
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install PostgreSQL
sudo apt install -y postgresql postgresql-contrib
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Install RabbitMQ
sudo apt install -y rabbitmq-server
sudo systemctl start rabbitmq-server
sudo systemctl enable rabbitmq-server
sudo rabbitmq-plugins enable rabbitmq_stomp

# Install Mosquitto
sudo apt install -y mosquitto mosquitto-clients
sudo systemctl start mosquitto
sudo systemctl enable mosquitto
```

#### 2. Database Setup

```bash
# Create database and user
sudo -u postgres psql <<EOF
CREATE DATABASE openusp;
CREATE USER openusp WITH ENCRYPTED PASSWORD 'openusp123';
GRANT ALL PRIVILEGES ON DATABASE openusp TO openusp;
\q
EOF

# Initialize schema
psql -h localhost -U openusp -d openusp -f configs/init-db.sql
```

#### 3. Build and Install OpenUSP

```bash
# Clone repository
git clone https://github.com/your-org/openusp.git
cd openusp

# Build services
make build

# Install binaries
sudo mkdir -p /opt/openusp/bin
sudo cp build/* /opt/openusp/bin/

# Install configuration
sudo mkdir -p /etc/openusp
sudo cp configs/openusp.env /etc/openusp/
sudo cp -r configs/ /opt/openusp/
```

#### 4. Create System Services

Create systemd service files:

```bash
# API Gateway service
sudo tee /etc/systemd/system/openusp-api-gateway.service > /dev/null <<EOF
[Unit]
Description=OpenUSP API Gateway
After=network.target postgresql.service

[Service]
Type=simple
User=openusp
Group=openusp
WorkingDirectory=/opt/openusp
EnvironmentFile=/etc/openusp/openusp.env
ExecStart=/opt/openusp/bin/api-gateway
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Data Service
sudo tee /etc/systemd/system/openusp-data-service.service > /dev/null <<EOF
[Unit]
Description=OpenUSP Data Service
After=network.target postgresql.service

[Service]
Type=simple
User=openusp
Group=openusp
WorkingDirectory=/opt/openusp
EnvironmentFile=/etc/openusp/openusp.env
ExecStart=/opt/openusp/bin/data-service
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

# Create remaining services...
```

#### 5. Start Services

```bash
# Create openusp user
sudo useradd -r -s /bin/false openusp
sudo chown -R openusp:openusp /opt/openusp

# Reload systemd and start services
sudo systemctl daemon-reload
sudo systemctl enable openusp-data-service
sudo systemctl enable openusp-api-gateway
sudo systemctl enable openusp-mtp-service
sudo systemctl enable openusp-usp-service
sudo systemctl enable openusp-cwmp-service

sudo systemctl start openusp-data-service
sudo systemctl start openusp-api-gateway
sudo systemctl start openusp-mtp-service
sudo systemctl start openusp-usp-service
sudo systemctl start openusp-cwmp-service

# Check status
sudo systemctl status openusp-*
```

## Cloud Provider Deployment

### AWS Deployment

#### Using ECS (Elastic Container Service)

```yaml
# ecs-task-definition.json
{
  "family": "openusp",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "1024",
  "memory": "2048",
  "executionRoleArn": "arn:aws:iam::account:role/ecsTaskExecutionRole",
  "containerDefinitions": [
    {
      "name": "api-gateway",
      "image": "your-registry/openusp-api-gateway:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "CONSUL_ENABLED",
          "value": "true"
        },
        {
          "name": "OPENUSP_DATABASE_HOST",
          "value": "your-rds-endpoint"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/openusp",
          "awslogs-region": "us-west-2",
          "awslogs-stream-prefix": "api-gateway"
        }
      }
    }
  ]
}
```

#### Using EKS (Elastic Kubernetes Service)

```bash
# Create EKS cluster
eksctl create cluster --name openusp --region us-west-2 --nodes 3

# Deploy OpenUSP
kubectl apply -f deployments/kubernetes/
```

### Google Cloud Platform

#### Using Cloud Run

```yaml
# cloudrun-api-gateway.yaml
apiVersion: serving.knative.dev/v1
kind: Service
metadata:
  name: openusp-api-gateway
  namespace: default
spec:
  template:
    metadata:
      annotations:
        autoscaling.knative.dev/maxScale: "10"
    spec:
      containers:
      - image: gcr.io/project-id/openusp-api-gateway
        ports:
        - containerPort: 8080
        env:
        - name: OPENUSP_DATABASE_HOST
          value: "cloud-sql-proxy"
        resources:
          limits:
            cpu: "1000m"
            memory: "512Mi"
```

#### Using GKE (Google Kubernetes Engine)

```bash
# Create GKE cluster
gcloud container clusters create openusp \
  --num-nodes=3 \
  --zone=us-central1-a \
  --enable-autoscaling \
  --min-nodes=1 \
  --max-nodes=10

# Deploy OpenUSP
kubectl apply -f deployments/kubernetes/
```

### Azure Deployment

#### Using Container Instances

```yaml
# azure-container-group.yaml
apiVersion: 2019-12-01
location: eastus
name: openusp
properties:
  containers:
  - name: api-gateway
    properties:
      image: your-registry/openusp-api-gateway:latest
      resources:
        requests:
          cpu: 1
          memoryInGb: 1
      ports:
      - port: 8080
        protocol: TCP
  - name: data-service
    properties:
      image: your-registry/openusp-data-service:latest
      resources:
        requests:
          cpu: 1
          memoryInGb: 1
      ports:
      - port: 9092
        protocol: TCP
  osType: Linux
  restartPolicy: Always
type: Microsoft.ContainerInstance/containerGroups
```

## High Availability Setup

### Multi-Zone Deployment

```yaml
# Kubernetes with pod anti-affinity
apiVersion: apps/v1
kind: Deployment
metadata:
  name: api-gateway
spec:
  replicas: 3
  template:
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchExpressions:
              - key: app
                operator: In
                values:
                - api-gateway
            topologyKey: kubernetes.io/hostname
          - labelSelector:
              matchExpressions:
              - key: app
                operator: In
                values:
                - api-gateway
            topologyKey: topology.kubernetes.io/zone
```

### Database High Availability

#### PostgreSQL Cluster

```yaml
# Using PostgreSQL operator
apiVersion: postgresql.cnpg.io/v1
kind: Cluster
metadata:
  name: postgres-cluster
spec:
  instances: 3
  primaryUpdateStrategy: unsupervised
  
  postgresql:
    parameters:
      max_connections: "200"
      shared_buffers: "256MB"
      effective_cache_size: "1GB"
  
  bootstrap:
    initdb:
      database: openusp
      owner: openusp
      secret:
        name: postgres-credentials
  
  storage:
    size: 100Gi
    storageClass: fast-ssd
```

### Load Balancing

#### NGINX Configuration

```nginx
upstream api_gateway {
    server api-gateway-1:8080;
    server api-gateway-2:8080;
    server api-gateway-3:8080;
}

server {
    listen 80;
    server_name api.openusp.local;
    
    location / {
        proxy_pass http://api_gateway;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
    
    location /health {
        access_log off;
        return 200 "healthy\n";
        add_header Content-Type text/plain;
    }
}
```

## Monitoring and Observability

### Prometheus Configuration

```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'api-gateway'
    static_configs:
      - targets: ['api-gateway:9090']
  
  - job_name: 'data-service'
    static_configs:
      - targets: ['data-service:9090']
  
  - job_name: 'mtp-service'
    static_configs:
      - targets: ['mtp-service:9090']

  - job_name: 'kubernetes-pods'
    kubernetes_sd_configs:
      - role: pod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
```

### Grafana Dashboards

Key metrics to monitor:

- **API Gateway**: Request rate, response time, error rate
- **Data Service**: Database connections, query performance
- **MTP Service**: Message throughput, connection count
- **System**: CPU, memory, disk usage, network I/O

### Logging

#### Centralized Logging with ELK Stack

```yaml
# filebeat.yml
filebeat.inputs:
- type: container
  paths:
    - '/var/lib/docker/containers/*/*.log'
  processors:
  - add_kubernetes_metadata:
      host: ${NODE_NAME}
      matchers:
      - logs_path:
          logs_path: "/var/lib/docker/containers/"

output.elasticsearch:
  hosts: ["elasticsearch:9200"]
  index: "openusp-%{+yyyy.MM.dd}"

logging.level: info
```

## Security Considerations

### Network Security

```bash
# Firewall rules (iptables)
# Allow SSH
iptables -A INPUT -p tcp --dport 22 -j ACCEPT

# Allow HTTP/HTTPS
iptables -A INPUT -p tcp --dport 80 -j ACCEPT
iptables -A INPUT -p tcp --dport 443 -j ACCEPT

# Allow OpenUSP services
iptables -A INPUT -p tcp --dport 8080 -j ACCEPT  # API Gateway
iptables -A INPUT -p tcp --dport 8081 -j ACCEPT  # MTP Service
iptables -A INPUT -p tcp --dport 7547 -j ACCEPT  # CWMP Service

# Drop all other traffic
iptables -A INPUT -j DROP
```

### SSL/TLS Configuration

```bash
# Generate certificates
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout openusp.key -out openusp.crt \
  -subj "/C=US/ST=State/L=City/O=Organization/CN=api.openusp.local"

# Configure environment
export OPENUSP_TLS_ENABLED=true
export OPENUSP_TLS_CERT_FILE=/etc/ssl/certs/openusp.crt
export OPENUSP_TLS_KEY_FILE=/etc/ssl/private/openusp.key
```

### Secrets Management

#### Using Kubernetes Secrets

```bash
# Create database secret
kubectl create secret generic postgres-credentials \
  --from-literal=username=openusp \
  --from-literal=password=secure-password

# Use in deployment
env:
- name: OPENUSP_DATABASE_PASSWORD
  valueFrom:
    secretKeyRef:
      name: postgres-credentials
      key: password
```

#### Using HashiCorp Vault

```bash
# Store secrets
vault kv put secret/openusp/database \
  username=openusp \
  password=secure-password

# Retrieve in application
OPENUSP_DATABASE_PASSWORD=$(vault kv get -field=password secret/openusp/database)
```

### Backup and Disaster Recovery

```bash
# Database backup
pg_dump -h localhost -U openusp openusp > backup-$(date +%Y%m%d).sql

# Automated backup script
#!/bin/bash
BACKUP_DIR="/backups/openusp"
DATE=$(date +%Y%m%d_%H%M%S)

# Create backup
pg_dump -h $DB_HOST -U $DB_USER $DB_NAME | gzip > $BACKUP_DIR/openusp_$DATE.sql.gz

# Cleanup old backups (keep 30 days)
find $BACKUP_DIR -name "openusp_*.sql.gz" -mtime +30 -delete

# Upload to S3 (optional)
aws s3 cp $BACKUP_DIR/openusp_$DATE.sql.gz s3://your-backup-bucket/openusp/
```

## Performance Optimization

### Database Optimization

```sql
-- Create indices for common queries
CREATE INDEX idx_devices_endpoint_id ON devices(endpoint_id);
CREATE INDEX idx_parameters_device_id ON parameters(device_id);
CREATE INDEX idx_parameters_path ON parameters(path);
CREATE INDEX idx_alerts_device_id ON alerts(device_id);
CREATE INDEX idx_alerts_created_at ON alerts(created_at);

-- Optimize PostgreSQL configuration
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET random_page_cost = 1.1;
ALTER SYSTEM SET checkpoint_completion_target = 0.7;
SELECT pg_reload_conf();
```

### Application Optimization

```bash
# Environment tuning
export GOGC=100                    # Go garbage collection
export GOMAXPROCS=8               # Max Go processes
export OPENUSP_API_GATEWAY_WORKERS=4  # HTTP workers
export OPENUSP_DATABASE_MAX_CONNECTIONS=50  # DB pool size
```

---

For more deployment examples and configuration options, see the `deployments/` directory in the repository.